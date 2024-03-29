package gds

import (
	"container/list"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrTimeWheelArgument      = errors.New("incorrect task argument")
	ErrTimeWheelArgumentDelay = errors.New("parameter 'delay' must be large than zero")
	ErrTimeWheelStop          = errors.New("time wheel is stopped")
)

type (
	// A TimeWheel is a time wheel object to schedule tasks.
	TimeWheel struct {
		// TimeWheel is a time wheel for scheduling tasks.
		interval       time.Duration
		ticker         *time.Ticker
		slots          []*list.List
		timers         sync.Map
		currentPos     int
		slotNum        int
		addTaskChan    chan timeWheelSlot
		removeTaskChan chan any
		moveTaskChan   chan baseSlot
		stopChan       chan struct{}
		// assigned by internal if not specify key
		currentTaskID uint64
	}
	// TimeWheelTaskCallback defined the method to run task while timeout.
	TimeWheelTaskCallback func(taskID any, data any)

	timeWheelTaskID uint64

	timeWheelTask struct {
		taskID any
		// Data is the data of the task
		data any
		// Func is callback when timeout
		callback TimeWheelTaskCallback
	}

	timeWheelSlot struct {
		baseSlot
		circle  int // while 0 ,trigger the task
		diffPos int // the diff position of the ori position if moving task
		task    *timeWheelTask
		removed bool
	}

	baseSlot struct {
		delay  time.Duration
		taskID any
	}

	timeWheelPos struct {
		pos  int
		item *timeWheelSlot
	}
)

// NewTimeWheel create a new time wheel with the given interval and slot number.
//
// once the time wheel is created, it will start to run tasks in a goroutine.
func NewTimeWheel(interval time.Duration, slotNum int) (*TimeWheel, error) {
	if interval <= 0 || slotNum <= 0 {
		return nil, errors.New("invalid parameter 'interval' or 'slotNum' must be large than zero")
	}

	return newTimeWheelWithTicker(interval, slotNum, time.NewTicker(interval))
}

func newTimeWheelWithTicker(interval time.Duration, slotNum int, ticker *time.Ticker) (*TimeWheel, error) {
	tw := &TimeWheel{
		interval:       interval,
		ticker:         ticker,
		slotNum:        slotNum,
		slots:          make([]*list.List, slotNum),
		currentPos:     slotNum - 1, // when run, currentPos will be start 0
		addTaskChan:    make(chan timeWheelSlot),
		removeTaskChan: make(chan any),
		moveTaskChan:   make(chan baseSlot),
		stopChan:       make(chan struct{}),
	}

	tw.initSlots()
	go tw.start()
	return tw, nil
}

func (tw *TimeWheel) initSlots() {
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = list.New()
	}
}

// start time wheel. to handle all chan listener in the loop
func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case task := <-tw.addTaskChan:
			tw.addTask(&task)
		case taskID := <-tw.removeTaskChan:
			tw.removeTask(taskID)
		case task := <-tw.moveTaskChan:
			tw.moveTask(task)
		case <-tw.stopChan:
			tw.ticker.Stop()
			return
		}
	}
}

// Stop stops the time wheel.
func (tw *TimeWheel) Stop() {
	close(tw.stopChan)
}

// AddTask add a task to the time wheel, return the task id
func (tw *TimeWheel) AddTask(data any, delay time.Duration, callback TimeWheelTaskCallback) (taskID any, err error) {
	if delay <= 0 {
		return 0, ErrTimeWheelArgumentDelay
	}

	tid := timeWheelTaskID(tw.currentTaskID)
	atomic.AddUint64(&tw.currentTaskID, 1)
	err = tw.AddTimer(tid, data, delay, callback)
	return tid, err
}

// AddTimer add a timer task, if task id exists, do reset operator
func (tw *TimeWheel) AddTimer(taskID, data any, delay time.Duration, callback TimeWheelTaskCallback) error {
	if delay <= 0 {
		return ErrTimeWheelArgumentDelay
	}
	select {
	case tw.addTaskChan <- timeWheelSlot{
		baseSlot: baseSlot{delay: delay, taskID: taskID},
		task:     &timeWheelTask{taskID: taskID, data: data, callback: callback},
	}:
		return nil
	case <-tw.stopChan:
		return ErrTimeWheelStop
	}
}

// ResetTask reset timer by the given key to the given delay.
func (tw *TimeWheel) ResetTask(taskID any, delay time.Duration) error {
	if delay <= 0 || taskID == nil {
		return ErrTimeWheelArgument
	}
	select {
	case tw.moveTaskChan <- baseSlot{delay: delay, taskID: taskID}:
		return nil
	case <-tw.stopChan:
		return ErrTimeWheelStop
	}
}

func (tw *TimeWheel) RemoveTask(taskID any) error {
	if taskID == nil {
		return ErrTimeWheelArgument
	}
	select {
	case tw.removeTaskChan <- taskID:
		return nil
	case <-tw.stopChan:
		return ErrTimeWheelStop
	}
}

func (tw *TimeWheel) addTask(taskSlot *timeWheelSlot) {
	if taskSlot.delay < tw.interval {
		taskSlot.delay = tw.interval
	}
	val, ok := tw.timers.Load(taskSlot.taskID)
	if ok {
		posItem := val.(*timeWheelPos)
		posItem.item.task = taskSlot.task
		tw.moveTask(posItem.item.baseSlot)
		return
	}
	pos, circle := tw.getPositionAndCircle(taskSlot.delay)
	taskSlot.circle = circle
	tw.slots[pos].PushBack(taskSlot)
	if taskSlot.taskID != nil {
		tw.timers.Store(taskSlot.taskID, &timeWheelPos{
			pos:  pos,
			item: taskSlot,
		})
	}
}

func (tw *TimeWheel) moveTask(task baseSlot) {
	val, ok := tw.timers.Load(task.taskID)
	if !ok {
		return
	}
	posItem := val.(*timeWheelPos)
	pos, circle := tw.getPositionAndCircle(task.delay)
	if pos >= posItem.pos {
		posItem.item.circle = circle
		posItem.item.diffPos = pos - posItem.pos
	} else if circle > 0 {
		circle--
		posItem.item.circle = circle
		posItem.item.diffPos = tw.slotNum - posItem.pos + pos
	} else {
		posItem.item.removed = true
		newItem := &timeWheelSlot{
			baseSlot: task,
			task:     posItem.item.task,
		}
		tw.slots[pos].PushBack(newItem)
		tw.updatePosition(posItem.item, pos)
	}
}

func (tw *TimeWheel) updatePosition(task *timeWheelSlot, pos int) {
	val, ok := tw.timers.Load(task.taskID)
	if ok {
		posItem := val.(*timeWheelPos)
		posItem.pos = pos
		posItem.item = task
		return
	}
	tw.timers.Store(task.taskID, &timeWheelPos{
		pos:  pos,
		item: task,
	})
}

func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (pos, circle int) {
	steps := int(d / tw.interval)
	pos = (tw.currentPos + steps) % tw.slotNum
	circle = (steps - 1) / tw.slotNum

	return
}

func (tw *TimeWheel) removeTask(taskID any) {
	position, ok := tw.timers.Load(taskID)
	if !ok { // taskID not exist
		return
	}
	posItem := position.(*timeWheelPos)
	posItem.item.removed = true
	tw.timers.Delete(taskID)
}

func (tw *TimeWheel) tickHandler() {
	tw.currentPos = (tw.currentPos + 1) % tw.slotNum
	l := tw.slots[tw.currentPos]
	tw.scanAndRunTask(l)
}

func (tw *TimeWheel) scanAndRunTask(l *list.List) {
	var tasks []*timeWheelTask
	for e := l.Front(); e != nil; {
		taskSlot := e.Value.(*timeWheelSlot)
		if taskSlot.removed {
			next := e.Next()
			l.Remove(e)
			e = next
			continue
		} else if taskSlot.circle > 0 {
			taskSlot.circle--
			e = e.Next()
			continue
		} else if taskSlot.diffPos > 0 {
			next := e.Next()
			l.Remove(e)
			pos := (tw.currentPos + taskSlot.diffPos) % tw.slotNum
			tw.slots[pos].PushBack(taskSlot)
			tw.updatePosition(taskSlot, pos)
			taskSlot.diffPos = 0
			e = next
			continue
		}
		tasks = append(tasks, taskSlot.task)
		next := e.Next()
		l.Remove(e)
		tw.timers.Delete(taskSlot.taskID)
		e = next
	}
	tw.doTasks(tasks)
}

func (tw *TimeWheel) doTasks(tasks []*timeWheelTask) {
	if len(tasks) == 0 {
		return
	}
	go func() {
		for _, task := range tasks {
			task.callback(task.taskID, task.data)
		}
	}()
}

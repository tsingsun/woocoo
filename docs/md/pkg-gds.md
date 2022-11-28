---
title: 数据结构
---

## 时间轮

时间轮是一种高效来利用线程资源来进行批量化调度的一种调度模型。
把大批量的调度任务全部都绑定到同一个的调度器上面，使用这一个调度器来进行所有任务的管理，触发以及运行,
能够高效的管理各种延时任务，周期任务，通知任务等等。

不过，时间轮调度器的时间精度可能不是很高，对于精度要求特别高的调度任务可能不太适合。
因为时间轮算法的精度取决于，时间段“指针”单元的最小粒度大小，比如时间轮的格子是一秒跳一次，那么调度精度小于一秒的任务就无法被时间轮所调度。

使用方式:

```go
func Example(){
    tw, err := timewheel.New(100*time.Millisecond, 300)
    if err != nil {
    fmt.Println(err)
        return
    }
	cb := func(taskID,data any) {
		// do something
    }
	data := map[string]int{"uid": 100, "age": 16},
	tw.AddTask(data, time.Minute,cb)
	// key and value
	tw.AddTimer("key","value",time.Minute,cb)
	// stop the time wheel
	tw.Stop()
}
```
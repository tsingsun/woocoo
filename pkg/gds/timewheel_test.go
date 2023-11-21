package gds

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

const (
	defaultTickerDuration = time.Millisecond * 100
	waitTime              = defaultTickerDuration
)

func TestNewTimeWheel(t *testing.T) {
	_, err := NewTimeWheel(0, 10)
	assert.Error(t, err)
}

func TestTimeWheel_AddTimerOnce(t *testing.T) {
	ticker := time.NewTicker(waitTime)
	tw, _ := NewTimeWheel(defaultTickerDuration, 10)
	var cb = func(k, v any) {
		assert.Equal(t, "any", k)
		assert.Equal(t, 3, v.(int))
		ticker.Stop()
	}
	defer tw.Stop()
	assert.NoError(t, tw.AddTimer("any", 3, defaultTickerDuration, cb))
	time.Sleep(waitTime)
}

func TestTimeWheel_AddTimerTwice(t *testing.T) {
	ticker := time.NewTicker(waitTime)

	var cb = func(k, v any) {
		assert.Equal(t, "any", k)
		assert.Equal(t, 5, v.(int))
		ticker.Stop()
	}
	tw, _ := NewTimeWheel(defaultTickerDuration, 10)
	defer tw.Stop()
	assert.NoError(t, tw.AddTimer("any", 3, defaultTickerDuration*4, cb))
	assert.NoError(t, tw.AddTimer("any", 5, defaultTickerDuration*7, cb))
	time.Sleep(defaultTickerDuration * 8)
}

func TestTimeWheel_AddWrongDelay(t *testing.T) {
	ticker := time.NewTicker(waitTime)
	tw, _ := newTimeWheelWithTicker(defaultTickerDuration, 10, ticker)
	defer tw.Stop()
	assert.NotPanics(t, func() {
		tw.AddTask(3, -defaultTickerDuration, func(key any, data any) {})
	})
}

func TestTimeWheel_AddAfterStop(t *testing.T) {
	tw, _ := NewTimeWheel(defaultTickerDuration, 10)
	tw.Stop()
	_, err := tw.AddTask("data", defaultTickerDuration, func(key any, data any) {})
	assert.Error(t, err)
	assert.Error(t, tw.AddTimer("any", "data", defaultTickerDuration, func(key any, data any) {}))
}

func TestTimeWheel_AddTimerAndRun(t *testing.T) {
	tests := []struct {
		slots      int
		delayCount time.Duration
	}{
		{
			slots:      5,
			delayCount: 5,
		},
		{
			slots:      5,
			delayCount: 7,
		},
		{
			slots:      5,
			delayCount: 10,
		},
		{
			slots:      5,
			delayCount: 12,
		},
		{
			slots:      5,
			delayCount: 7,
		},
		{
			slots:      5,
			delayCount: 10,
		},
		{
			slots:      5,
			delayCount: 12,
		},
	}

	for _, tt := range tests {
		delayCount := tt.delayCount
		slots := tt.slots
		t.Run(strconv.Itoa(rand.Int()), func(t *testing.T) {
			t.Parallel()

			var count int32
			ticker := time.NewTicker(waitTime)
			var actual int32
			done := make(chan struct{})
			var cb = func(k, v any) {
				assert.Equal(t, 1, k.(int))
				assert.Equal(t, 2, v.(int))
				actual = atomic.LoadInt32(&count)
				close(done)
			}
			tw, err := newTimeWheelWithTicker(defaultTickerDuration, uint16(slots), ticker)
			assert.Nil(t, err)
			defer tw.Stop()

			tw.AddTimer(1, 2, defaultTickerDuration*delayCount, cb)
			for {
				select {
				case <-done:
					assert.EqualValues(t, delayCount, actual)
					return
				default:
					atomic.AddInt32(&count, 1)
					time.Sleep(defaultTickerDuration)
				}
			}
		})
	}
}

func TestTimeWheel_ResetTask(t *testing.T) {
	count := int64(0)
	cb := func(k, v any) {
		assert.Equal(t, "any", k)
		assert.Equal(t, 3, v.(int))
		assert.EqualValues(t, atomic.LoadInt64(&count), 1)
		atomic.AddInt64(&count, 1)
	}
	tw, _ := NewTimeWheel(defaultTickerDuration, 3)
	assert.NoError(t, tw.AddTimer("any", 3, defaultTickerDuration*4, cb))
	assert.NoError(t, tw.ResetTask("any", defaultTickerDuration*7))
	assert.Error(t, tw.ResetTask("any", -defaultTickerDuration))
	assert.NoError(t, tw.ResetTask("any", defaultTickerDuration))
	atomic.AddInt64(&count, 1)
	time.Sleep(defaultTickerDuration * 8)
	tw.Stop()
	assert.Error(t, tw.ResetTask("any", time.Millisecond))
}

func TestMoveAndRemoveTask(t *testing.T) {
	var keys []int
	cb := func(id, data any) {
		assert.Equal(t, "any", id)
		assert.Equal(t, 3, data.(int))
		keys = append(keys, data.(int))
	}
	tw, _ := NewTimeWheel(defaultTickerDuration, 10)
	defer tw.Stop()
	tw.AddTimer("any", 3, defaultTickerDuration*8, cb)
	tw.ResetTask("any", defaultTickerDuration*7)
	tw.RemoveTask("any")
	time.Sleep(defaultTickerDuration)
	assert.Equal(t, 0, len(keys))
}

func BenchmarkTimingWheel(b *testing.B) {
	b.ReportAllocs()
	cb := func(taskID any, data any) {
		// do nothing
	}
	tw, _ := NewTimeWheel(time.Second, 100)
	for i := 0; i < b.N; i++ {
		tid, err := tw.AddTask(i, time.Second, cb)
		assert.NoError(b, err)
		tw.AddTimer(b.N+i, b.N+i, time.Second, cb)
		tw.ResetTask(tid, time.Second*time.Duration(i))
		tw.removeTask(tid)
	}
}

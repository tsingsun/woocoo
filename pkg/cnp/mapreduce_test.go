package cnp

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var errDummy = errors.New("dummy")

func TestParallel(t *testing.T) {
	var total uint32
	err := Parallel(func() error {
		atomic.AddUint32(&total, 2)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 3)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 5)
		return nil
	})

	assert.Equal(t, uint32(10), atomic.LoadUint32(&total))
	assert.Nil(t, err)
}

func TestParallelNone(t *testing.T) {
	assert.Nil(t, Parallel())
}

func TestParallelVoidNone(t *testing.T) {
	ParallelVoid()
}

func TestParallelErr(t *testing.T) {
	var total uint32
	err := Parallel(func() error {
		atomic.AddUint32(&total, 2)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 3)
		return errDummy
	}, func() error {
		atomic.AddUint32(&total, 5)
		return nil
	})

	assert.Equal(t, errDummy, err)
}

func TestParallelVoid(t *testing.T) {
	var total uint32
	ParallelVoid(func() {
		atomic.AddUint32(&total, 2)
	}, func() {
		atomic.AddUint32(&total, 3)
	}, func() {
		atomic.AddUint32(&total, 5)
	})

	assert.Equal(t, uint32(10), atomic.LoadUint32(&total))
}

func TestMap(t *testing.T) {
	tests := []struct {
		mapper MapperFunc
		expect int
	}{
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				writer.Write(v * v)
			},
			expect: 30,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%2 == 0 {
					return
				}
				writer.Write(v * v)
			},
			expect: 10,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%2 == 0 {
					panic(v)
				}
				writer.Write(v * v)
			},
			expect: 10,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestMap:%d", i), func(t *testing.T) {
			channel := MapReduce(func(source chan<- interface{}) {
				for i := 1; i < 5; i++ {
					source <- i
				}
			}, WithWorkers(-1)).Map(test.mapper).MapResult()

			var result int
			for v := range channel {
				result += v.(int)
			}

			assert.Equal(t, test.expect, result)
		})
	}
}

func TestMapReduce(t *testing.T) {
	tests := []struct {
		mapper      MapperFunc
		reducer     ReducerFunc
		expectErr   error
		expectValue interface{}
	}{
		{
			expectErr:   nil,
			expectValue: 30,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(errDummy)
				}
				writer.Write(v * v)
			},
			expectErr: errDummy,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(nil)
				}
				writer.Write(v * v)
			},
			expectErr:   ErrCancelWithNil,
			expectValue: nil,
		},
		{
			reducer: func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
				var result int
				for item := range pipe {
					result += item.(int)
					if result > 10 {
						cancel(errDummy)
					}
				}
				writer.Write(result)
			},
			expectErr: errDummy,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestMapReduce:%d", i), func(t *testing.T) {
			if test.mapper == nil {
				test.mapper = func(item interface{}, writer Writer, cancel func(error)) {
					v := item.(int)
					writer.Write(v * v)
				}
			}
			if test.reducer == nil {
				test.reducer = func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
					var result int
					for item := range pipe {
						result += item.(int)
					}
					writer.Write(result)
				}
			}
			value, err := MapReduce(func(source chan<- interface{}) {
				for i := 1; i < 5; i++ {
					source <- i
				}
			}, WithWorkers(runtime.NumCPU())).Map(test.mapper).Reduce(test.reducer).Result()

			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectValue, value)
		})
	}
}

func TestMapReduceWithReduerWriteMoreThanOnce(t *testing.T) {
	assert.Panics(t, func() {
		MapReduce(func(source chan<- interface{}) {
			for i := 0; i < 10; i++ {
				source <- i
			}
		}).Map(func(item interface{}, writer Writer, cancel func(error)) {
			writer.Write(item)
		}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			drain(pipe)
			writer.Write("one")
			writer.Write("two")
		}).Result()
	})
}

func TestMapReduceVoid(t *testing.T) {
	var value uint32
	tests := []struct {
		mapper      MapperFunc
		reducer     ReducerFunc
		expectValue uint32
		expectErr   error
	}{
		{
			expectValue: 30,
			expectErr:   nil,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(errDummy)
				}
				writer.Write(v * v)
			},
			expectErr: errDummy,
		},
		{
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(nil)
				}
				writer.Write(v * v)
			},
			expectErr: ErrCancelWithNil,
		},
		{
			reducer: func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
				for item := range pipe {
					result := atomic.AddUint32(&value, uint32(item.(int)))
					if result > 10 {
						cancel(errDummy)
					}
				}
				NotifyDone(writer)
			},
			expectErr: errDummy,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestMapReduceVoid:%d", i), func(t *testing.T) {
			atomic.StoreUint32(&value, 0)

			if test.mapper == nil {
				test.mapper = func(item interface{}, writer Writer, cancel func(error)) {
					v := item.(int)
					writer.Write(v * v)
				}
			}
			if test.reducer == nil {
				test.reducer = func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
					for item := range pipe {
						atomic.AddUint32(&value, uint32(item.(int)))
					}
					NotifyDone(writer)
				}
			}
			err := MapReduce(func(source chan<- interface{}) {
				for i := 1; i < 5; i++ {
					source <- i
				}
			}).Map(test.mapper).Reduce(test.reducer).Dry()

			assert.Equal(t, test.expectErr, err)
			if err == nil {
				assert.Equal(t, test.expectValue, atomic.LoadUint32(&value))
			}
		})
	}
}

func TestMapReduceVoidWithDelay(t *testing.T) {
	var result []int
	err := MapReduce(func(source chan<- interface{}) {
		source <- 0
		source <- 1
	}).Map(func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == 0 {
			time.Sleep(time.Millisecond * 50)
		}
		writer.Write(i)
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
		NotifyDone(writer)
	}).Dry()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 1, result[0])
	assert.Equal(t, 0, result[1])
}

func TestMapReducePanic(t *testing.T) {
	v, err := MapReduce(func(source chan<- interface{}) {
		source <- 0
		source <- 1
	}).Map(func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		writer.Write(i)
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		for range pipe {
			panic("panic")
		}
	}).Result()
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.Equal(t, "panic", err.Error())
}

func TestMapReduceVoidCancel(t *testing.T) {
	var result []int
	err := MapReduce(func(source chan<- interface{}) {
		source <- 0
		source <- 1
	}).Map(func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == 1 {
			cancel(errors.New("anything"))
		}
		writer.Write(i)
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
		NotifyDone(writer)
	}).Dry()
	assert.NotNil(t, err)
	assert.Equal(t, "anything", err.Error())
}

func TestMapReduceVoidCancelWithRemains(t *testing.T) {
	var done atomic.Value
	var result []int
	err := MapReduce(func(source chan<- interface{}) {
		for i := 0; i < defaultWorkers*2; i++ {
			source <- i
		}
		done.Store(true)
	}).Map(func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == defaultWorkers/2 {
			cancel(errors.New("anything"))
		}
		writer.Write(i)
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
		NotifyDone(writer)
	}).Dry()
	assert.NotNil(t, err)
	assert.Equal(t, "anything", err.Error())
	assert.True(t, done.Load().(bool))
}

func TestMapReduceWithoutReducerWrite(t *testing.T) {
	uids := []int{1, 2, 3}
	res, err := MapReduce(func(source chan<- interface{}) {
		for _, uid := range uids {
			source <- uid
		}
	}).Map(func(item interface{}, writer Writer, cancel func(error)) {
		writer.Write(item)
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		drain(pipe)
		// not calling writer.Write(...), should not panic
	}).Result()
	assert.Equal(t, ErrReduceNoOutput, err)
	assert.Nil(t, res)
}

func BenchmarkMapReduce(b *testing.B) {
	b.ReportAllocs()

	mapper := func(v interface{}, writer Writer, cancel func(error)) {
		writer.Write(v.(int64) * v.(int64))
	}
	reducer := func(input <-chan interface{}, writer Writer, cancel func(error)) {
		var result int64
		for v := range input {
			result += v.(int64)
		}
		writer.Write(result)
	}

	for i := 0; i < b.N; i++ {
		MapReduce(func(input chan<- interface{}) {
			for j := 0; j < 2; j++ {
				input <- int64(j)
			}
		}).Map(mapper).Reduce(reducer).Result()
	}
}

// Package cnp
//
// concurrence and parallel tools
// mapreduce core source from go-zero
package cnp

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	defaultWorkers = 16
	minWorkers     = 1
)

var (
	// ErrCancelWithNil is an error that mapreduce was cancelled with nil.
	ErrCancelWithNil = errors.New("mapreduce cancelled with nil")
	// ErrReduceNoOutput is an error that reduce did not output a value.
	ErrReduceNoOutput = errors.New("reduce not writing value")
)

type (
	// GenerateFunc is used to let callers send elements into source.
	GenerateFunc func(source chan<- interface{})
	// MapperFunc is used to do element processing and write the output to writer,
	// use cancel func to cancel the processing.
	MapperFunc func(item interface{}, writer Writer, cancel func(error))
	// ReducerFunc is used to reduce all the mapping output and write to writer,
	// use cancel func to cancel the processing.
	ReducerFunc func(pipe <-chan interface{}, writer Writer, cancel func(error))
	// Option defines the method to customize the mapreduce.
	Option func(opts *mapReduceOptions)

	mapReduceOptions struct {
		workers int
	}

	// Writer interface wraps Write method.
	Writer interface {
		Write(v interface{})
	}
)

type mapReduce struct {
	generate GenerateFunc
	mapper   MapperFunc
	reducer  ReducerFunc
	opts     *mapReduceOptions
}

func (mr *mapReduce) Map(mapper MapperFunc) *mapReduce {
	if mr.mapper != nil {
		panic("only accept a mapper function")
	}
	mr.mapper = mapper
	return mr
}

func (mr *mapReduce) Reduce(reducerFunc ReducerFunc) *mapReduce {
	mr.reducer = reducerFunc
	return mr
}

// Result return map and reduce result
func (mr mapReduce) Result() (interface{}, error) {
	source := buildSource(mr.generate)
	if mr.reducer == nil {
		panic("miss reducer function")
	}
	return runMapperWithSource(source, mr.mapper, mr.reducer, mr.opts.workers)
}

// Dry return error in map and reduce process
func (mr *mapReduce) Dry() error {
	source := buildSource(mr.generate)
	if mr.reducer == nil { //just map func
		mr.reducer = func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			drain(pipe)
			NotifyDone(writer)
		}
		//drain(mr.MapResult())
		//return nil
	}
	_, err := runMapperWithSource(source, mr.mapper, mr.reducer, mr.opts.workers)
	return err
}

// MapResult return result, only execute map function
func (mr *mapReduce) MapResult() chan interface{} {
	source := buildSource(mr.generate)
	collector := make(chan interface{}, mr.opts.workers)
	done := NewDoneChan()
	go runMapper(mr.mapper, source, collector, done.Done(), mr.opts.workers)
	return collector
}

// NotifyDone notify the process,reduce has done. if you reduce function do not output value ,you must call this method
func NotifyDone(writer Writer) {
	writer.Write(struct{}{})
}

func runMapper(mapper MapperFunc, input <-chan interface{}, collector chan<- interface{}, done <-chan struct{}, workers int) {
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		close(collector)
	}()
	pool := make(chan struct{}, workers)
	writer := newGuardedWriter(collector, done)
	for {
		select {
		case <-done:
			return
		case pool <- struct{}{}:
			item, ok := <-input
			if !ok {
				<-pool
				return
			}

			wg.Add(1)
			// better to safely run caller defined method
			go func() {
				defer func() {
					wg.Done()
					<-pool
				}()

				mapper(item, writer, nil)
			}()
		}
	}
}

// MapReduceWithSource maps all elements from source, and reduce the output elements with given reducer.
func runMapperWithSource(source <-chan interface{}, mapper MapperFunc, reducer ReducerFunc, workers int) (interface{}, error) {
	output := make(chan interface{})
	defer func() {
		for range output {
			panic("more than one element written in reducer")
		}
	}()

	collector := make(chan interface{}, workers)
	done := NewDoneChan()
	writer := newGuardedWriter(output, done.Done())
	var closeOnce sync.Once
	var retErr atomic.Value
	finish := func() {
		closeOnce.Do(func() {
			done.Close()
			close(output)
		})
	}
	cancel := once(func(err error) {
		if err == nil {
			retErr.Store(ErrCancelWithNil)
		} else {
			retErr.Store(err)
		}

		drain(source)
		finish()
	})

	go func() {
		defer func() {
			drain(collector)

			if r := recover(); r != nil {
				cancel(fmt.Errorf("%v", r))
			} else {
				finish()
			}
		}()

		reducer(collector, writer, cancel)
	}()

	go runMapper(func(item interface{}, w Writer, c func(error)) {
		mapper(item, w, cancel)
	}, source, collector, done.Done(), workers)

	value, ok := <-output
	if v := retErr.Load(); v != nil {
		return nil, v.(error)
	} else if ok {
		return value, nil
	} else {
		return nil, ErrReduceNoOutput
	}
}

func Parallel(fns ...func() error) error {
	if len(fns) == 0 {
		return nil
	}
	err := MapReduce(func(source chan<- interface{}) {
		for _, fn := range fns {
			source <- fn
		}
	}, WithWorkers(len(fns))).Map(func(item interface{}, writer Writer, cancel func(error)) {
		fn := item.(func() error)
		if err := fn(); err != nil {
			cancel(err)
		}
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		drain(pipe)
		// We need to write a placeholder to let MapReduce to continue on reducer done,
		// otherwise, all goroutines are waiting. The placeholder will be discarded by MapReduce.
		writer.Write(struct{}{})
	}).Dry()
	return err
}

func ParallelVoid(fns ...func()) {
	if len(fns) == 0 {
		return
	}
	MapReduce(func(source chan<- interface{}) {
		for _, fn := range fns {
			source <- fn
		}
	}, WithWorkers(len(fns))).Map(func(item interface{}, writer Writer, cancel func(error)) {
		fn := item.(func())
		fn()
	}).Reduce(func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		drain(pipe)
		// We need to write a placeholder to let MapReduce to continue on reducer done,
		// otherwise, all goroutines are waiting. The placeholder will be discarded by MapReduce.
		NotifyDone(writer)
	}).Dry()
}

func MapReduce(gen GenerateFunc, opts ...Option) *mapReduce {
	mr := &mapReduce{
		generate: gen,
	}
	mr.opts = buildOptions(opts...)
	return mr
}

// WithWorkers customizes a mapreduce processing with given workers.
func WithWorkers(workers int) Option {
	return func(opts *mapReduceOptions) {
		if workers < minWorkers {
			opts.workers = minWorkers
		} else {
			opts.workers = workers
		}
	}
}

func buildOptions(opts ...Option) *mapReduceOptions {
	options := newOptions()
	for _, opt := range opts {
		opt(options)
	}

	return options
}

func buildSource(generate GenerateFunc) chan interface{} {
	source := make(chan interface{})
	go func() {
		defer close(source)
		generate(source)
	}()

	return source
}

// drain drains the channel.
func drain(channel <-chan interface{}) {
	// drain the channel
	for range channel {
	}
}

func newOptions() *mapReduceOptions {
	return &mapReduceOptions{
		workers: defaultWorkers,
	}
}

func once(fn func(error)) func(error) {
	once := new(sync.Once)
	return func(err error) {
		once.Do(func() {
			fn(err)
		})
	}
}

type guardedWriter struct {
	channel chan<- interface{}
	done    <-chan struct{}
}

func newGuardedWriter(channel chan<- interface{}, done <-chan struct{}) guardedWriter {
	return guardedWriter{
		channel: channel,
		done:    done,
	}
}

func (gw guardedWriter) Write(v interface{}) {
	select {
	case <-gw.done:
		return
	default:
		gw.channel <- v
	}
}

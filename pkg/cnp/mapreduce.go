// Package cnp
//
// concurrence and parallel tools
// mapreduce core source from go-zero
package cnp

import (
	"context"
	"errors"
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
	GenerateFunc func(source chan<- any)
	// MapperFunc is used to do element processing and write the output to writer,
	// use cancel func to cancel the processing.
	MapperFunc func(item any, writer Writer, cancel func(error))
	// ReducerFunc is used to reduce all the mapping output and write to writer,
	// use cancel func to cancel the processing.
	ReducerFunc func(pipe <-chan any, writer Writer, cancel func(error))
	// Option defines the method to customize the mapreduce.
	Option func(opts *mapReduceOptions)

	mapReduceOptions struct {
		ctx     context.Context
		workers int
	}

	// Writer interface wraps Write method.
	Writer interface {
		Write(v any)
	}
)

// mapReduce hold the map reduce process
type mapReduce struct {
	generate  GenerateFunc
	mapper    MapperFunc
	reducer   ReducerFunc
	opts      *mapReduceOptions
	panicChan *onceChan
}

func (mr *mapReduce) Map(mapper MapperFunc) *mapReduce {
	if mr.mapper != nil {
		panic("only accept a mapper function")
	}
	mr.mapper = mapper
	mr.panicChan = &onceChan{channel: make(chan any)}
	return mr
}

func (mr *mapReduce) Reduce(reducerFunc ReducerFunc) *mapReduce {
	mr.reducer = reducerFunc
	return mr
}

// Result return map and reduce result
func (mr *mapReduce) Result() (any, error) {
	source := buildSource(mr.generate)
	if mr.reducer == nil {
		panic("miss reducer function")
	}
	return mr.mapReduceWithSource(source)
}

// Dry is the void executor and only return error in map and reduce process ,and do not return result
func (mr *mapReduce) Dry() error {
	source := buildSource(mr.generate)
	if mr.reducer == nil {
		mr.reducer = func(pipe <-chan any, writer Writer, cancel func(error)) {
			drain(pipe)
			NotifyDone(writer)
		}
	}
	_, err := mr.mapReduceWithSource(source)
	return err
}

// MapResult return result, only execute map function
func (mr *mapReduce) MapResult() chan any {
	source := buildSource(mr.generate)
	collector := make(chan any, mr.opts.workers)
	done := make(chan struct{})
	go mr.executeMappers(mr.mapper, source, collector, done)
	return collector
}

// NotifyDone notify the process,reduce has done. if you reduce function do not output value ,you must call this method
func NotifyDone(writer Writer) {
	writer.Write(struct{}{})
}

// executeMappers execute mappers
func (mr *mapReduce) executeMappers(mapper MapperFunc, source <-chan any, collector chan<- any, done <-chan struct{}) {
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		close(collector)
		drain(source)
	}()

	var failed int32
	pool := make(chan struct{}, mr.opts.workers)
	writer := newGuardedWriter(mr.opts.ctx, collector, done)
	for atomic.LoadInt32(&failed) == 0 {
		select {
		case <-mr.opts.ctx.Done():
			return
		case <-done:
			return
		case pool <- struct{}{}:
			item, ok := <-source
			if !ok {
				<-pool
				return
			}

			wg.Add(1)
			// better to safely run caller defined method
			go func() {
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt32(&failed, 1)
						mr.panicChan.write(r)
					}
					wg.Done()
					<-pool
				}()

				mapper(item, writer, nil)
			}()
		}
	}
}

// MapReduceWithSource maps all elements from source, and reduce the output elements with given reducer.
func (mr *mapReduce) mapReduceWithSource(source <-chan any) (any, error) {
	output := make(chan any)
	defer func() {
		for range output {
			panic("more than one element written in reducer")
		}
	}()

	collector := make(chan any, mr.opts.workers)
	done := make(chan struct{})
	writer := newGuardedWriter(mr.opts.ctx, output, done)
	var closeOnce sync.Once
	var retErr atomic.Value
	finish := func() {
		closeOnce.Do(func() {
			close(done)
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
				mr.panicChan.write(r)
			}
			finish()
		}()

		mr.reducer(collector, writer, cancel)
	}()

	go mr.executeMappers(func(item any, w Writer, c func(error)) {
		mr.mapper(item, w, cancel)
	}, source, collector, done)

	select {
	case <-mr.opts.ctx.Done():
		cancel(context.DeadlineExceeded)
		return nil, context.DeadlineExceeded
	case v := <-mr.panicChan.channel:
		panic(v)
	case value, ok := <-output:
		if v := retErr.Load(); v != nil {
			return nil, v.(error)
		} else if ok {
			return value, nil
		} else {
			return nil, ErrReduceNoOutput
		}
	}
}

// Parallel runs fns parallel, cancelled on any error
func Parallel(fns ...func() error) error {
	if len(fns) == 0 {
		return nil
	}
	err := MapReduce(func(source chan<- any) {
		for _, fn := range fns {
			source <- fn
		}
	}, WithWorkers(len(fns))).Map(func(item any, writer Writer, cancel func(error)) {
		fn := item.(func() error)
		if err := fn(); err != nil {
			cancel(err)
		}
	}).Reduce(func(pipe <-chan any, writer Writer, cancel func(error)) {
		drain(pipe)
		// We need to write a placeholder to let MapReduce to continue on reducer done,
		// otherwise, all goroutines are waiting. The placeholder will be discarded by MapReduce.
		writer.Write(struct{}{})
	}).Dry()
	return err
}

// ParallelVoid runs void functions parallel
func ParallelVoid(fns ...func()) {
	if len(fns) == 0 {
		return
	}
	MapReduce(func(source chan<- any) {
		for _, fn := range fns {
			source <- fn
		}
	}, WithWorkers(len(fns))).Map(func(item any, writer Writer, cancel func(error)) {
		fn := item.(func())
		fn()
	}).Reduce(func(pipe <-chan any, writer Writer, cancel func(error)) {
		drain(pipe)
		// We need to write a placeholder to let MapReduce to continue on reducer done,
		// otherwise, all goroutines are waiting. The placeholder will be discarded by MapReduce.
		NotifyDone(writer)
	}).Dry()
}

// MapReduce runs mapper and reducer, is the entry method of mapreduce
func MapReduce(gen GenerateFunc, opts ...Option) *mapReduce {
	mr := &mapReduce{
		generate: gen,
	}
	mr.opts = buildOptions(opts...)
	return mr
}

// WithContext customizes a mapreduce processing accepts a given ctx.
func WithContext(ctx context.Context) Option {
	return func(opts *mapReduceOptions) {
		opts.ctx = ctx
	}
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

func buildSource(generate GenerateFunc) chan any {
	source := make(chan any)
	go func() {
		defer close(source)
		generate(source)
	}()

	return source
}

// drain drains the channel.
func drain(channel <-chan any) {
	// drain the channel
	for range channel {
	}
}

func newOptions() *mapReduceOptions {
	return &mapReduceOptions{
		ctx:     context.Background(),
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
	ctx     context.Context
	channel chan<- any
	done    <-chan struct{}
}

func newGuardedWriter(ctx context.Context, channel chan<- any, done <-chan struct{}) guardedWriter {
	return guardedWriter{
		ctx:     ctx,
		channel: channel,
		done:    done,
	}
}

func (gw guardedWriter) Write(v any) {
	select {
	case <-gw.ctx.Done():
		return
	case <-gw.done:
		return
	default:
		gw.channel <- v
	}
}

type onceChan struct {
	channel chan any
	wrote   int32
}

func (oc *onceChan) write(val any) {
	if atomic.CompareAndSwapInt32(&oc.wrote, 0, 1) {
		oc.channel <- val
	}
}

package wctest

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/logtest"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

// Configuration returns a testdata configuration for test.
func Configuration() *conf.Configuration {
	return conf.New(
		conf.WithGlobal(true),
		conf.WithBaseDir(testdata.BaseDir()),
		conf.WithLocalPath(testdata.Path(testdata.DefaultConfigFile)),
	).Load()
}

// InitGlobalLogger sets a sample logger as the global logger for pkg log test.
func InitGlobalLogger(disableStacktrace bool) {
	glog := log.InitGlobalLogger()
	glog.Apply(conf.NewFromBytes([]byte(fmt.Sprintf(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: %s`, strconv.FormatBool(disableStacktrace)))))
	glog.AsGlobal()
}

// InitBuffWriteSyncer returns a Memory WriteSyncer for log test
func InitBuffWriteSyncer(opts ...zap.Option) *logtest.Buffer {
	opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	logdata := &logtest.Buffer{}
	zl := logtest.NewBuffLogger(logdata, opts...)
	glog := log.Global().Logger()
	opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zl.Core()
	}))
	glog.WithOptions(opts...).AsGlobal()
	return logdata
}

// RunWait runs some long time task functions for given duration to start, and the tasks keep running after timeout.
// log is a function to log error message.
func RunWait(log func(args ...any), timeout time.Duration, fns ...func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	wg := sync.WaitGroup{}
	for _, fn := range fns {
		fn := fn
		eg.Go(func() error {
			<-ctx.Done()
			err := ctx.Err()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return err
			}
			return nil
		})
		wg.Add(1)
		go func() {
			wg.Done()
			if err := fn(); err != nil {
				log(err)
				cancel()
			}
		}()
	}
	wg.Wait()
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// RunWaitDone runs the given functions thought goroutine and waits for a time. If all tasks done, return nil error.
// if task return error,Run will return context.Canceled error immediately. you should handle the error in the task.
func RunWaitDone(timeout time.Duration, fns ...func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	wg := sync.WaitGroup{}
	doneCh := make(chan int)
	allDone := false
	for _, fn := range fns {
		fn := fn
		eg.Go(func() error {
			<-ctx.Done()
			err := ctx.Err()
			if err != nil {
				// if canceled but all tasks done, return nil
				if errors.Is(err, context.Canceled) && allDone {
					return nil
				}
				return err
			}
			return nil
		})
		wg.Add(1)
		go func() {
			wg.Done()
			if err := fn(); err != nil {
				cancel()
				return
			}
			doneCh <- 1
		}()
	}
	wg.Wait()
	go func() {
		tf := len(fns)
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-doneCh:
				if !ok {
					return
				}
				tf -= d
				if tf == 0 {
					allDone = true
					cancel()
				}
			}
		}
	}()
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

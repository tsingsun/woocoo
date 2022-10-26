package wctest

import (
	"context"
	"errors"
	"fmt"
	"github.com/tsingsun/woocoo/internal/logtest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
	"testing"
	"time"
)

func Configuration() *conf.Configuration {
	return conf.New(
		conf.WithBaseDir(testdata.BaseDir()),
		conf.WithLocalPath(testdata.Path(testdata.DefaultConfigFile)),
	).Load()
}

func ApplyGlobal(disableStacktrace bool) {
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

func RunWait(t *testing.T, timeout time.Duration, fns ...func() error) error {
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
				t.Error(err)
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

package woocoo

import (
	"context"
	"errors"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// App is the application.
type App struct {
	opts options

	ctx    context.Context
	cancel func()
}

func New(opts ...Option) *App {
	app := &App{}
	app.opts = options{
		quitCh:      []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT},
		StopTimeout: time.Second * 5,
	}
	for _, opt := range opts {
		opt(&app.opts)
	}
	if app.opts.cnf == nil {
		app.opts.cnf = &conf.AppConfiguration{Configuration: conf.New().Load()}
	}
	if app.opts.cnf.IsSet("log") {
		log.NewBuiltIn()
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	return app
}

func (a *App) AppConfiguration() *conf.AppConfiguration {
	return a.opts.cnf
}

func (a *App) RegisterServer(servers ...Server) {
	a.opts.servers = append(a.opts.servers, servers...)
}

func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.ctx)
	wg := sync.WaitGroup{}
	for _, srv := range a.opts.servers {
		srv := srv
		eg.Go(func() error {
			<-ctx.Done()
			stopCtx, cancel := context.WithTimeout(context.Background(), a.opts.StopTimeout)
			defer cancel()
			return srv.Stop(stopCtx)
		})
		wg.Add(1)
		eg.Go(func() error {
			wg.Done()
			return srv.Start(context.Background())
		})
	}
	wg.Wait()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, a.opts.quitCh...)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-quit:
			return a.Stop()
		}
	})
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// Stop the application.
func (a *App) Stop() error {
	a.cancel()
	return nil
}

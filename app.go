package woocoo

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

// Server is the interface that can run in App.
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// App is the application with a universal mechanism to manage goroutine lifecycles.
type App struct {
	opts options

	ctx    context.Context
	cancel func()
}

// New creates an application by Option.
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
		app.opts.cnf = &conf.AppConfiguration{Configuration: conf.New(conf.WithGlobal(true))}
		if app.opts.cnf.Exists() {
			app.opts.cnf.Load()
		}
	}
	if app.opts.cnf.IsSet("log") {
		ll := log.NewFromConf(app.opts.cnf.Sub("log"))
		ll.AsGlobal()
	} else {
		log.InitGlobalLogger() // reset global logger, assign component logger.
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

// Run all Server concurrently.
// Run returns when all Server have exited.
// Run returns the first non-nil error (if any) from them.
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
			return srv.Start(a.ctx)
		})
		if a.opts.interval > 0 {
			time.Sleep(a.opts.interval)
		}
	}
	wg.Wait()
	if len(a.opts.quitCh) == 0 {
		eg.Go(func() error {
			<-ctx.Done()
			return ctx.Err()
		})
	} else {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, a.opts.quitCh...)
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return nil // hup app out not return error
			case <-quit:
				return a.Stop()
			}
		})
	}
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

// Sync calls some resources suck as logger flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (a *App) Sync() error {
	// sync global logger at last, ignore error because of "sync /dev/stderr: invalid argument"
	_ = log.Sync()
	return nil
}

// miniServer is an adapter that wraps start/stop functions into a Server.
type miniServer struct {
	start func(ctx context.Context) error
	stop  func(ctx context.Context) error
}

func (s *miniServer) Start(ctx context.Context) error {
	return s.start(ctx)
}

func (s *miniServer) Stop(ctx context.Context) error {
	return s.stop(ctx)
}

// MiniApp provides a simplified way to run multiple servers with lifecycle management.
// It returns a run function that starts all servers and a stop function to gracefully shut them down.
// If timeout > 0, the context will be cancelled after the timeout duration.
func MiniApp(ctx context.Context, timeout time.Duration, servers ...Server) (run, stop func() error) {
	app := &App{}
	if timeout > 0 {
		app.ctx, app.cancel = context.WithTimeout(ctx, timeout)
	} else {
		app.ctx, app.cancel = context.WithCancel(ctx)
	}
	app.RegisterServer(servers...)
	return func() error {
		return app.Run()
	}, app.Stop
}

// Group implementation is derived from github.com/oklog/run (MIT License).
// Copyright (c) 2017 Peter Bourgon
// See https://github.com/oklog/run/blob/master/LICENSE for the full license.

type actor struct {
	execute   func() error
	interrupt func(error)
}

// Group collects actors (functions) and runs them concurrently.
// When one actor (function) returns, all actors are interrupted.
// The zero value of a Group is useful.
type Group struct {
	actors []actor
}

// Add an actor (function) to the group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
func (g *Group) Add(execute func() error, interrupt func(error)) *Group {
	g.actors = append(g.actors, actor{execute, interrupt})
	return g
}

// Run all actors (functions) concurrently.
// When the first actor returns, all others are interrupted.
// Run only returns when all actors have exited.
// Run returns the error returned by the first exiting actor.
func (g *Group) Run() error {
	if len(g.actors) == 0 {
		return nil
	}

	// Run each actor.
	errs := make(chan error, len(g.actors))
	for _, a := range g.actors {
		go func(a actor) {
			errs <- a.execute()
		}(a)
	}

	// Wait for the first actor to stop.
	err := <-errs

	// Signal all actors to stop.
	for _, a := range g.actors {
		a.interrupt(err)
	}

	// Wait for all actors to stop.
	for i := 1; i < cap(errs); i++ {
		<-errs
	}

	// Return the original error.
	return err
}

package web

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

const (
	defaultAddr = ":8080"
)

var (
	logger = log.Component(log.WebComponentName)
)

type ServerOptions struct {
	Addr           string              `json:"addr" yaml:"addr"`
	TLS            *conf.TLS           `json:"tls" yaml:"tls"`
	configuration  *conf.Configuration // not root configuration
	handlerManager *HandlerManager     // middleware manager
	gracefulStop   bool                // run with grace full shutdown
}

type Server struct {
	// opts is struct server parameter
	opts ServerOptions
	// hold the gin router
	router *Router
	// low level
	httpSrv *http.Server
}

// New create a web server
func New(opts ...Option) *Server {
	s := &Server{
		opts: ServerOptions{
			Addr:           defaultAddr,
			handlerManager: NewHandlerManager(),
		},
	}
	for _, o := range opts {
		o(&s.opts)
	}
	if s.router == nil {
		s.router = NewRouter(&s.opts)
	}
	if s.opts.configuration != nil {
		if err := s.Apply(s.opts.configuration); err != nil {
			panic(err)
		}
	}
	s.httpSrv = &http.Server{
		Addr:    s.opts.Addr,
		Handler: s.router.Engine,
	}
	return s
}

// ServerOptions return a setting used by web server
func (s *Server) ServerOptions() ServerOptions {
	return s.opts
}

// HandlerManager return server's handler manager,it's convenient to process handler
func (s *Server) HandlerManager() *HandlerManager {
	return s.opts.handlerManager
}

func (s *Server) Router() *Router {
	return s.router
}

// Apply implement conf.Configuration
func (s *Server) Apply(cfg *conf.Configuration) error {
	if k := "server"; cfg.IsSet(k) {
		if err := cfg.Parser().Unmarshal(k, &s.opts); err != nil {
			return err
		}
	}
	if k := "server.tls"; cfg.IsSet(k) {
		s.opts.TLS = conf.NewTLS(cfg.Sub(k))
	}
	if k := "engine"; cfg.IsSet(k) {
		if err := s.router.Apply(cfg.Sub(k)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) beforeRun() error {
	if s.opts.Addr == "" {
		return errors.New("web server configuration incorrect: miss listen address")
	}
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	logger.Info(fmt.Sprintf("listening and serving HTTP on %s", s.opts.Addr))
	err := s.ListenAndServe()
	if err != nil && errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop http server and clear resource
func (s *Server) Stop(ctx context.Context) error {
	err := s.httpServerStop(ctx)
	if err != nil {
		logger.Error("web Server close err", zap.Error(err))
	}
	if hm := s.opts.handlerManager; hm != nil {
		hm.Shutdown(ctx) //nolint:errcheck
	}
	// ignore error handling,see https://github.com/uber-go/zap/issues/880
	log.Sync() //nolint:errcheck
	return nil
}

// ListenAndServe Starts Http Server
//
// return
//
//	http.ErrServerClosed or other error
func (s *Server) ListenAndServe() (err error) {
	if err = s.beforeRun(); err != nil {
		return
	}
	if s.opts.TLS != nil {
		err = s.httpSrv.ListenAndServeTLS(s.opts.TLS.Cert, s.opts.TLS.Key)
	} else {
		err = s.httpSrv.ListenAndServe()
	}
	return
}

// Run builtin run the server.
//
// you can process whole yourself
func (s *Server) Run() error {
	ch := make(chan error)
	quitFunc := func() {
		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.Stop(ctx) //nolint:errcheck
	}
	go func() {
		err := s.Start(context.Background())
		if err != nil {
			ch <- err
			return
		}
		close(ch)
	}()
	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		logger.Info(fmt.Sprintf("web server on %s shutdown", s.opts.Addr))
		defer quitFunc()
	case err := <-ch:
		if err != nil {
			defer quitFunc()
			return err
		}
	}
	return nil
}

func (s *Server) httpServerStop(ctx context.Context) error {
	if s.opts.gracefulStop {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			logger.Error("server forced to runAndClose", zap.Error(err))
		}
	} else {
		if err := s.httpSrv.Close(); err != nil {
			logger.Error("Server forced to runAndClose", zap.Error(err))
		}
	}
	return nil
}

package web

import (
	"context"
	"errors"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	configPath  = "web"
	defaultAddr = ":8080"
)

type ServerSetting struct {
	Addr              string `json:"addr" yaml:"addr"`
	SSLCertificate    string `json:"ssl_certificate" yaml:"ssl_certificate"`
	SSLCertificateKey string `json:"ssl_certificate_key" yaml:"ssl_certificate_key"`
	Development       bool
}

type Server struct {
	// serverSetting is struct server parameter
	serverSetting ServerSetting
	// middleware manager
	handlerManager *handler.Manager
	// hold the gin router
	router *Router
	// configuration is application level Configuration
	configuration *conf.Configuration
	logger        *log.Logger

	quit chan os.Signal
}

func New(opts ...Option) *Server {
	srv := &Server{
		serverSetting: ServerSetting{
			Addr: defaultAddr,
		},
		quit:           make(chan os.Signal),
		handlerManager: handler.NewManager(),
	}
	for _, o := range opts {
		o(srv)
	}

	log.PrintLogo()
	if srv.router == nil {
		srv.router = NewRouter(srv)
	}

	return srv
}

func NewBuiltIn(opts ...Option) *Server {
	srv := New(opts...)
	// Config must first check
	if srv.configuration == nil {
		Config()(srv)
	}
	if srv.logger == nil {
		UseLogger()(srv)
	}

	srv.Apply(srv.configuration, configPath)
	return srv
}

// ServerSetting return a setting used by web server
func (s *Server) ServerSetting() ServerSetting {
	return s.serverSetting
}

func (s *Server) Router() *Router {
	return s.router
}

func (s *Server) Logger() *log.Logger {
	return s.logger
}

func (s *Server) Apply(cfg *conf.Configuration, path string) {
	if s.configuration == nil {
		s.configuration = cfg
	}
	cc, err := cfg.Parser().Sub(path)
	if err != nil {
		panic(err)
	}

	if err = cc.Unmarshal("server", &s.serverSetting); err != nil {
		panic(err)
	}
	s.serverSetting.Development = cfg.Development

	if err = s.router.Apply(cfg.Sub(path), "engine"); err != nil {
		panic(err)
	}
}

func (s *Server) beforeRun() error {
	if s.serverSetting.Addr == "" {
		return errors.New("server configuration is not correct: miss listen")
	}
	return s.router.RehandleRule()

}

func (s *Server) stop() {
	s.handlerManager.Shutdown()
	// ignore,see https://github.com/uber-go/zap/issues/880
	if err := s.logger.Sync(); err != nil {
		log.StdPrintln(err)
	}
}

// ForceQuit the http server.
func (s *Server) ForceQuit() {
	close(s.quit)
}

// Run Starts Http Server
func (s *Server) Run(graceful bool) (err error) {
	defer s.stop()
	if err = s.beforeRun(); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    s.serverSetting.Addr,
		Handler: s.router.Engine,
	}
	if graceful {
		s.runAndGracefulShutdown(srv)
	} else {
		s.runAndClose(srv)
	}
	return nil
}

func (s *Server) runAndGracefulShutdown(srv *http.Server) {
	runSSL := s.serverSetting.SSLCertificate != "" && s.serverSetting.SSLCertificateKey != ""
	go func() {
		var err error
		log.StdPrintf("start listening on %s", s.serverSetting.Addr)
		if runSSL {
			err = srv.ListenAndServeTLS(s.serverSetting.SSLCertificate, s.serverSetting.SSLCertificateKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.StdPrintf("listen: %s", err)
		}
		close(s.quit)
	}()
	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(s.quit, syscall.SIGINT, syscall.SIGTERM)
	<-s.quit

	log.StdPrintln("Shutting down server...")
	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.StdPrintf("Server forced to runAndClose: %v", err)
	}
}

func (s *Server) runAndClose(srv *http.Server) {
	runSSL := s.serverSetting.SSLCertificate != "" && s.serverSetting.SSLCertificateKey != ""

	go func() {
		var err error
		log.StdPrintf("start listening on %s", s.serverSetting.Addr)
		if runSSL {
			err = srv.ListenAndServeTLS(s.serverSetting.SSLCertificate, s.serverSetting.SSLCertificateKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.StdPrintf("listen: %s", err)
		}
		close(s.quit)
	}()

	signal.Notify(s.quit, syscall.SIGINT, syscall.SIGTERM)
	<-s.quit
	log.StdPrintln("Shutting down server...")
	if err := srv.Close(); err != nil {
		log.StdPrintln("Server Close:", err)
	}
}

package web

import (
	"context"
	"errors"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	configPath = "web"
)

type ServerConfig struct {
	Addr              string `json:"addr" yaml:"addr"`
	SSLCertificate    string `json:"ssl_certificate" yaml:"ssl_certificate"`
	SSLCertificateKey string `json:"ssl_certificate_key" yaml:"ssl_certificate_key"`
	Development       bool
}

type Server struct {
	config        *ServerConfig
	configuration *conf.Configuration
	router        *Router
	logger        *log.Logger
}

func New(opts ...Option) *Server {
	srv := &Server{
		config: &ServerConfig{
			Addr: ":8080",
		},
	}
	for _, o := range opts {
		o(srv)
	}
	return srv
}

func Default(opts ...Option) *Server {
	srv := New(
		Config(),
		UseLogger(),
		UseRedisCache(),
	)
	srv.Apply(srv.configuration, configPath)
	return srv
}

func (s Server) ServerConfig() ServerConfig {
	return *s.config
}
func (s *Server) Router() *Router {
	return s.router
}

func (s Server) Logger() *log.Logger {
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

	if err = cc.UnmarshalByJson("server", s.config); err != nil {
		panic(err)
	}
	s.config.Development = cfg.Development

	//must last apply
	if s.router == nil {
		s.router = NewRouter(s)
	}
	if err = s.router.Apply(cfg.Sub(path), "engine"); err != nil {
		panic(err)
	}
}

func (s Server) beforeRun() error {
	if s.config.Addr == "" {
		return errors.New("server configuration is not correct: miss listen")
	}
	return s.router.RehandleRule()

}

func (s *Server) stop() {
	s.logger.Sync()
}

func (s Server) Run(graceful bool) (err error) {
	defer s.stop()
	if err = s.beforeRun(); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    s.config.Addr,
		Handler: s.router.Engine,
	}
	if graceful {
		s.runAndGracefulShutdown(srv)
	} else {
		s.runAndClose(srv)
	}
	return nil
}

func (s Server) runAndGracefulShutdown(srv *http.Server) {
	runSSL := s.config.SSLCertificate != "" && s.config.SSLCertificateKey != ""
	go func() {
		var err error
		if runSSL {
			err = srv.ListenAndServeTLS(s.config.SSLCertificate, s.config.SSLCertificateKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("listen: %s\n", err)
		}
	}()
	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to runAndClose:", err)
	}
}

func (s Server) runAndClose(srv *http.Server) {
	var err error
	runSSL := s.config.SSLCertificate != "" && s.config.SSLCertificateKey != ""
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		log.Info("Shutting down server...")
		if err := srv.Close(); err != nil {
			log.Fatal("Server Close:", err)
		}
	}()

	if runSSL {
		err = srv.ListenAndServeTLS(s.config.SSLCertificate, s.config.SSLCertificateKey)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && errors.Is(err, http.ErrServerClosed) {
		log.Errorf("listen: %s\n", err)
	}
}

package woocoo

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"os"
	"time"
)

// Server is the interface that can run in App.
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Option func(o *options)

type options struct {
	cnf *conf.AppConfiguration
	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	quitCh []os.Signal

	servers []Server
	// StopTimeout is the timeout for stopping the server.
	StopTimeout time.Duration
}

// WithAppConfiguration set up the configuration of the web server by a configuration instance
func WithAppConfiguration(cnf *conf.Configuration) Option {
	return func(s *options) {
		s.cnf = &conf.AppConfiguration{Configuration: cnf}
	}
}

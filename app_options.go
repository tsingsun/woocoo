package woocoo

import (
	"os"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
)

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
	// interval time for App starting every server with time.Sleep in the server slice.
	interval time.Duration
	// StopTimeout is the timeout for stopping the server.
	StopTimeout time.Duration
}

// WithAppConfiguration set up the configuration of the web server by a configuration instance
func WithAppConfiguration(cnf *conf.Configuration) Option {
	return func(s *options) {
		s.cnf = &conf.AppConfiguration{Configuration: cnf}
	}
}

// WithInterval controls the interval time for App starting every server with time.Sleep if servers have some dependencies.
func WithInterval(interval time.Duration) Option {
	return func(s *options) {
		s.interval = interval
	}
}

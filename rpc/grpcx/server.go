package grpcx

import (
	"context"
	"errors"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var logger = log.Component(log.GrpcComponentName)

const (
	defaultPort    = 9080
	defaultNetwork = "tcp"
)

type serverOptions struct {
	// Network is the network protocol. default is tcp
	Network string `json:"network" yaml:"network"`
	Addr    string `json:"addr" yaml:"addr"`
	UseIPv6 bool   `json:"ipv6" yaml:"ipv6"`
	// Namespace will pass to registry component.default is Application NameSpace
	Namespace string `json:"namespace,omitempty" yaml:"namespace"`
	// Version is the grpc server version,default is Application Version which is set in the Application level config file
	Version string `json:"version" yaml:"version"`
	// RegistryMeta is the metadata for the registry service
	RegistryMeta map[string]string `json:"registryMeta" yaml:"registryMeta"`

	// listener is the net.Listener
	listener net.Listener
	host     string
	port     int

	grpcOptions []grpc.ServerOption
	// configuration is the grpc service Configuration
	configuration *conf.Configuration
	gracefulStop  bool
}

// Server is extended the native grpc server
type Server struct {
	opts   serverOptions
	engine *grpc.Server
	exit   chan chan error

	registry     registry.Registry
	registryDone bool
	// ServiceInfos is for service discovery, it converts from grpc service info
	ServiceInfos []*registry.ServiceInfo

	mu sync.RWMutex
}

// New creates a new grpc server.
func New(opts ...Option) *Server {
	s := &Server{
		opts: serverOptions{
			Network:      defaultNetwork,
			Addr:         fmt.Sprintf(":%d", defaultPort),
			port:         defaultPort,
			RegistryMeta: map[string]string{},
		},
		exit: make(chan chan error),
	}
	interceptor.UseContextLogger()
	for _, o := range opts {
		o(&s.opts)
	}
	if s.opts.configuration == nil {
		s.opts.configuration = conf.Global().Sub("grpc")
	}
	if cnf := s.opts.configuration; cnf != nil {
		s.opts.Version = cnf.Root().Version()
		s.opts.Namespace = cnf.Root().Namespace()
		if err := s.Apply(s.opts.configuration); err != nil {
			panic(err)
		}
	}
	s.engine = grpc.NewServer(s.opts.grpcOptions...)
	return s
}

func (s *Server) applyNetwork() (err error) {
	if s.opts.listener == nil {
		if s.opts.Network == "" {
			s.opts.Network = defaultNetwork
		}
		s.opts.listener, err = net.Listen(s.opts.Network, s.opts.Addr)
		if err != nil {
			return err
		}
	}
	s.opts.Addr = s.opts.listener.Addr().String()
	if tcpaddr, ok := s.opts.listener.Addr().(*net.TCPAddr); ok {
		s.opts.port = tcpaddr.Port
		s.opts.host = conf.GetIP(s.opts.UseIPv6)
		if tcpaddr.IP.IsLoopback() {
			s.opts.host = tcpaddr.IP.String()
		}
	}
	return nil
}

// Apply the configuration to the server.
func (s *Server) Apply(cfg *conf.Configuration) error {
	err := cfg.Parser().Unmarshal("server", &s.opts)
	if err != nil {
		panic(err)
	}
	if k := "registry"; cfg.IsSet(k) {
		rgcfg := cfg.Sub(k)
		scheme := rgcfg.String("scheme")
		drv, ok := registry.GetRegistry(scheme)
		if !ok {
			return fmt.Errorf("registry driver not found:%s", scheme)
		}
		if s.registry, err = drv.CreateRegistry(rgcfg); err != nil {
			return err
		}
	}
	// engine
	if k := "engine"; cfg.IsSet(k) {
		cnfOpts := optionsManager.BuildServerOptions(cfg, k)
		s.opts.grpcOptions = append(cnfOpts, s.opts.grpcOptions...)
	}
	return nil
}

// ListenAndServe call net listen to start grpc server and registry service
func (s *Server) ListenAndServe() (err error) {
	err = s.applyNetwork()
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("start grpc server on %s", s.opts.Addr))
	// registry run
	if s.registry != nil {
		for name := range s.engine.GetServiceInfo() {
			nd := &registry.ServiceInfo{
				Name:      name,
				Host:      s.opts.host,
				Port:      s.opts.port,
				Namespace: s.opts.Namespace,
				Version:   s.opts.Version,
				Metadata:  s.opts.RegistryMeta,
				Protocol:  s.opts.Network,
			}
			s.ServiceInfos = append(s.ServiceInfos, nd)
		}
		for _, serviceInfo := range s.ServiceInfos {
			if err := s.registry.Register(serviceInfo); err != nil {
				s.deregisterServices() // deregister all services if one fails
				return err
			}
		}
		if len(s.ServiceInfos) > 0 {
			s.mu.Lock()
			s.registryDone = true
			s.mu.Unlock()
		}
		go func() {
			t := new(time.Ticker)
			// only process if it exists
			if s.registry.TTL() > time.Duration(0) {
				// new ticker
				t = time.NewTicker(s.registry.TTL())
			}
			var ch chan error
			for {
				select {
				case <-t.C:
					for _, serviceInfo := range s.ServiceInfos {
						go func(info *registry.ServiceInfo) {
							if err := s.registry.Register(info); err != nil {
								grpclog.Errorf("grpcx: failed to register %s:%d to service %s(%s) at ttl: %v",
									info.Host, info.Port, info.Name, info.Namespace, err)
							}
						}(serviceInfo)
					}
				case ch = <-s.exit:
					t.Stop()
					s.deregisterServices()
					ch <- err
					return
				}
			}
		}()
	}
	// grpc Serve run, it will return a non-nil error unless Stop or WithGracefulStop is called.
	// so director check err
	err = s.engine.Serve(s.opts.listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		err = nil
	}
	return err
}

func (s *Server) deregisterServices() {
	for _, info := range s.ServiceInfos {
		if err := s.registry.Unregister(info); err != nil {
			grpclog.Errorf("grpcx: failed to register %s:%d to service %s(%s) at ttl: %v",
				info.Host, info.Port, info.Name, info.Namespace, err)
		}
	}
}

func (s *Server) Engine() *grpc.Server {
	return s.engine
}

func (s *Server) Start(_ context.Context) error {
	return s.ListenAndServe()
}

func (s *Server) Stop(_ context.Context) (err error) {
	if s.registry != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.registryDone {
			ch := make(chan error)
			s.exit <- ch
			err = <-ch
		}
	}
	if s.opts.gracefulStop {
		s.engine.GracefulStop()
	} else {
		s.engine.Stop()
	}
	return err
}

// Run is a sample way to start the grpc server with gracefulStop stop
func (s *Server) Run() error {
	quitFunc := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.Stop(ctx) //nolint:errcheck
	}
	ch := make(chan error)
	go func() {
		err := s.Start(context.Background())
		if err != nil {
			ch <- err
			return
		}
		close(ch) // normal stop
	}()

	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	// kill (no param) default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		logger.Info(fmt.Sprintf("grpc server on %s shutdown", s.opts.Addr))
		defer quitFunc()
		return nil
	case err := <-ch:
		if err != nil {
			defer quitFunc()
			return err
		}
	}
	return nil
}

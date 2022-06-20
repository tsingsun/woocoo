package grpcx

import (
	"errors"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type serverOptions struct {
	Addr              string `json:"addr" yaml:"addr"`
	UseIPv6           bool   `json:"ipv6" yaml:"ipv6"`
	SSLCertificate    string `json:"sslCertificate" yaml:"sslCertificate"`
	SSLCertificateKey string `json:"sslCertificateKey" yaml:"sslCertificateKey"`
	// Namespace is the registry service prefix,when grpc register service,it will use namespace+service as service name
	// so the Registry use the prefix to watch all services in grpc server
	Namespace string `json:"namespace" yaml:"namespace"`
	// Version is the grpc server version,default is Application Version which is set in the Application level config file
	Version string `json:"version" yaml:"version"`
	// RegistryMeta is the metadata for the registry service
	RegistryMeta map[string]string `json:"registryMeta" yaml:"registryMeta"`

	grpcOptions []grpc.ServerOption
	// configuration is the grpc service Configuration
	configuration *conf.Configuration
	gracefulStop  bool
}

// Server is the grpcx server
type Server struct {
	opts   serverOptions
	engine *grpc.Server
	exit   chan chan error

	registry registry.Registry
	// ServiceInfos is for service discovery,it converts from grpc service info
	ServiceInfos []*registry.ServiceInfo
}

func (s *Server) Apply(cfg *conf.Configuration) {
	err := cfg.Parser().Unmarshal("server", &s.opts)
	if err != nil {
		panic(err)
	}
	if k := strings.Join([]string{"registry"}, conf.KeyDelimiter); cfg.IsSet(k) {
		drv, ok := registry.GetRegistry(cfg.String(strings.Join([]string{"registry", "scheme"}, conf.KeyDelimiter)))
		if !ok {
			panic(fmt.Errorf("registry driver not found"))
		}
		if s.registry, err = drv.CreateRegistry(cfg.Sub(k)); err != nil {
			panic(err)
		}
	}
	// engine
	if k := strings.Join([]string{"engine"}, conf.KeyDelimiter); cfg.IsSet(k) {
		s.opts.grpcOptions = cGrpcServerOptions.Apply(cfg, k)
	}
}

func New(opts ...Option) *Server {
	s := &Server{
		opts: serverOptions{
			Addr:         ":9080",
			RegistryMeta: map[string]string{},
		},
		exit: make(chan chan error),
	}
	for _, o := range opts {
		o(&s.opts)
	}
	if s.opts.configuration != nil {
		s.opts.Version = s.opts.configuration.Root().Version()
		s.Apply(s.opts.configuration)
	}
	s.engine = grpc.NewServer(s.opts.grpcOptions...)
	return s
}

func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}
	// registry run
	if s.registry != nil {
		port := lis.Addr().(*net.TCPAddr).Port

		for name := range s.engine.GetServiceInfo() {
			nd := &registry.ServiceInfo{
				Name:      name,
				Host:      conf.GetIP(s.opts.UseIPv6),
				Port:      port,
				Namespace: s.opts.Namespace,
				Version:   s.opts.Version,
				Metadata:  s.opts.RegistryMeta,
				Protocol:  lis.Addr().Network(),
			}
			s.ServiceInfos = append(s.ServiceInfos, nd)
		}
		for _, serviceInfo := range s.ServiceInfos {
			if err := s.registry.Register(serviceInfo); err != nil {
				s.deregisterServices()
				return err
			}
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
	// grpc Serve run, it will return a non-nil error unless Stop or GracefulStop is called.
	// so director check err
	err = s.engine.Serve(lis)
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

func (s *Server) Stop() (err error) {
	if s.registry != nil {
		ch := make(chan error)
		s.exit <- ch
		err = <-ch
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
	defer s.Stop() //nolint:errcheck
	ch := make(chan error)
	go func() {
		grpclog.Infof("%s start grpc server on %s", s.opts.Namespace, s.opts.Addr)
		err := s.ListenAndServe()
		ch <- err
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
		grpclog.Info("grpc server shutdown.")
		return nil
	case err := <-ch:
		return err
	}
}

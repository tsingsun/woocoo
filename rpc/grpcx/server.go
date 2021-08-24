package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	configPath = "service"
)

type ServerConfig struct {
	Addr              string              `json:"addr" yaml:"addr"`
	SSLCertificate    string              `json:"ssl_certificate" yaml:"ssl_certificate"`
	SSLCertificateKey string              `json:"ssl_certificate_key" yaml:"ssl_certificate_key"`
	Location          string              `json:"location" yaml:"location"`
	Version           string              `json:"version" yaml:"version"`
	grpcOptions       []grpc.ServerOption `json:"-" yaml:"-"`
}

type Server struct {
	exit             chan chan error
	configuration    *conf.Configuration
	configurationKey string
	config           *ServerConfig
	logger           *log.Logger
	engine           *grpc.Server
	registry         registry.Registry
	NodeInfo         *registry.NodeInfo
}

func (s *Server) Apply(cfg *conf.Configuration, path string) {
	if s.configurationKey == "" && path != "" {
		s.configurationKey = path
	}
	if err := cfg.Sub(path).Parser().UnmarshalByJson("server", s.config); err != nil {
		panic(err)
	}
	if k := strings.Join([]string{path, "registry"}, conf.KeyDelimiter); cfg.IsSet(k) {
		s.registry = registry.GetRegistry(cfg.String(strings.Join([]string{path, "registry", "schema"}, conf.KeyDelimiter)))
		if ap, ok := s.registry.(conf.Configurable); ok {
			ap.Apply(cfg, k)
		}
	}
	//engine
	if k := strings.Join([]string{path, "engine"}, conf.KeyDelimiter); cfg.IsSet(k) {
		s.config.grpcOptions = cGrpcServerOptions.Apply(cfg, k)
	}

}

func (s *Server) applyGrpcConfiguration(cnf *conf.Configuration, path string) {

}

func New(opts ...Option) *Server {
	srv := &Server{
		config: &ServerConfig{
			Addr: ":9080",
		},
		configurationKey: configPath,
		exit:             make(chan chan error),
	}
	for _, o := range opts {
		o(srv)
	}
	if srv.configuration != nil && srv.configurationKey != "" {
		srv.Apply(srv.configuration, srv.configurationKey)
	}
	return srv
}

func Default(opts ...Option) *Server {
	defaultOpts := []Option{
		Config(),
		UseLogger(),
	}
	defaultOpts = append(defaultOpts, opts...)
	srv := New(opts...)
	return srv
}

func (s *Server) ApplyOptions(opts ...Option) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return err
	}

	s.runAndGracefulStop(lis)
	return nil
}

func (s *Server) Engine() *grpc.Server {
	if s.engine == nil {
		s.engine = grpc.NewServer(s.config.grpcOptions...)
	}
	return s.engine
}

func (s *Server) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	err := <-ch
	s.engine.GracefulStop()
	return err
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "error"
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("Unable to determine local IP address (non loopback). Exiting.")
}

func (s *Server) runAndGracefulStop(lis net.Listener) {
	go func() {
		log.Debug("starting grpc server")
		err := s.Engine().Serve(lis)
		if err != nil {
			log.Fatalf("could not serve: %v", err)
		}
	}()

	if s.registry != nil {
		port := lis.Addr().(*net.TCPAddr).Port

		s.NodeInfo = &registry.NodeInfo{
			ID:              getIP() + "-" + strconv.Itoa(port),
			ServiceLocation: s.config.Location,
			ServiceVersion:  s.config.Version,
			Address:         lis.Addr().String(),
		}
		if err := s.registry.Register(s.NodeInfo); err != nil {
			log.Fatalf("could not register server: %v", err)
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
					if err := s.registry.Register(s.NodeInfo); err != nil {
						log.Fatalf("could not register server: %v", err)
					}
				case ch = <-s.exit:
					t.Stop()
					err := s.registry.Unregister(s.NodeInfo)
					if err != nil {
						log.Errorf("registry unregister err: %v", err)
					}
					ch <- err
					return
				}
			}
		}()
	}

	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Debug("Shutting down server...")
	if err := s.Stop(); err != nil {
		log.Fatalf("Server forced to runAndClose: %v", err)
	}
}

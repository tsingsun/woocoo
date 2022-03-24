package grpcx

import (
	"errors"
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

type serverOptions struct {
	Addr              string `json:"addr" yaml:"addr"`
	SSLCertificate    string `json:"ssl_certificate" yaml:"ssl_certificate"`
	SSLCertificateKey string `json:"ssl_certificate_key" yaml:"ssl_certificate_key"`
	Location          string `json:"location" yaml:"location"`
	Version           string `json:"version" yaml:"version"`
	grpcOptions       []grpc.ServerOption
	// configuration is the grpc service Configuration
	configuration *conf.Configuration
	logger        log.ComponentLogger
}

type Server struct {
	opts   serverOptions
	engine *grpc.Server
	exit   chan chan error

	registry registry.Registry
	NodeInfo *registry.NodeInfo
}

func (s *Server) Apply(cfg *conf.Configuration) {
	if err := cfg.Parser().Unmarshal("server", &s.opts); err != nil {
		panic(err)
	}
	if k := strings.Join([]string{"registry"}, conf.KeyDelimiter); cfg.IsSet(k) {
		s.registry = registry.GetRegistry(cfg.String(strings.Join([]string{"registry", "schema"}, conf.KeyDelimiter)))
		if ap, ok := s.registry.(conf.Configurable); ok {
			ap.Apply(cfg.Sub(k))
		}
	}
	//engine
	if k := strings.Join([]string{"engine"}, conf.KeyDelimiter); cfg.IsSet(k) {
		s.opts.grpcOptions = cGrpcServerOptions.Apply(cfg, k)
	}
}

func New(opts ...Option) *Server {
	s := &Server{
		opts: serverOptions{
			Addr:   ":9080",
			logger: log.Component("grpc"),
		},
		exit: make(chan chan error),
	}
	for _, o := range opts {
		o(&s.opts)
	}
	if s.opts.configuration != nil {
		s.Apply(s.opts.configuration)
	}
	s.engine = grpc.NewServer(s.opts.grpcOptions...)
	return s
}

func (s *Server) applyOptions(opts ...Option) {
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}
	//registry run
	if s.registry != nil {
		port := lis.Addr().(*net.TCPAddr).Port

		s.NodeInfo = &registry.NodeInfo{
			ID:              getIP() + "-" + strconv.Itoa(port),
			ServiceLocation: s.opts.Location,
			ServiceVersion:  s.opts.Version,
			Address:         lis.Addr().String(),
		}
		if err := s.registry.Register(s.NodeInfo); err != nil {
			log.StdPrintf("could not register server: %v", err)
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
						log.StdPrintf("could not register server: %v", err)
					}
				case ch = <-s.exit:
					t.Stop()
					err := s.registry.Unregister(s.NodeInfo)
					if err != nil {
						log.StdPrintf("registry unregister err: %v", err)
					}
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

func (s *Server) Engine() *grpc.Server {
	return s.engine
}

func (s *Server) Stop() (err error) {
	if s.registry != nil {
		ch := make(chan error)
		s.exit <- ch
		err = <-ch
	}
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

// Run is a sample way to start the grpc server with graceful stop
func (s *Server) Run() error {
	defer s.Stop()
	ch := make(chan error)
	go func() {
		log.StdPrintf("%s start grpc server on %s", s.opts.Location, s.opts.Addr)
		err := s.ListenAndServe()
		if err != nil {
			ch <- err
		}
	}()

	// Wait for interrupt signal to gracefully runAndClose the server with
	// a timeout of 5 seconds.
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.StdPrintln("grpc server shutdown.")
		return nil
	case err := <-ch:
		return err
	}
}

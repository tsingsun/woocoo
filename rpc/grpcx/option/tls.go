package option

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"path/filepath"
)

// TLSOption tls option. it supports file or cert string
// if client `tls:` is empty, it will use insecure.NewCredentials()
type TLSOption struct {
}

// Name return the name of option
func (TLSOption) Name() string {
	return "tls"
}

func (TLSOption) getFiles(cfg *conf.Configuration) (certFile, keyFile string) {
	certFile = cfg.String("cert")
	keyFile = cfg.String("key")
	if certFile != "" && !filepath.IsAbs(certFile) {
		certFile = filepath.Join(cfg.GetBaseDir(), certFile)
	}
	if keyFile != "" && !filepath.IsAbs(keyFile) {
		keyFile = filepath.Join(cfg.GetBaseDir(), keyFile)
	}
	return
}

func (t TLSOption) ServerOption(cfg *conf.Configuration) grpc.ServerOption {
	cert, key := t.getFiles(cfg)
	if cert == "" || key == "" {
		panic("tls cert or key is empty")
	}
	tc, err := credentials.NewServerTLSFromFile(cert, key)
	if err != nil {
		panic(err)
	}
	return grpc.Creds(tc)
}

func (t TLSOption) DialOption(cfg *conf.Configuration) grpc.DialOption {
	cert, _ := t.getFiles(cfg)
	if cert == "" {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	tc, err := credentials.NewClientTLSFromFile(cert, "")
	if err != nil {
		panic(err)
	}
	return grpc.WithTransportCredentials(tc)
}

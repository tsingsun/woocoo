package option

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TLSOption tls option. it supports file or cert string
// if client `tls:` is empty, it will use insecure.NewCredentials()
type TLSOption struct {
}

// Name return the name of option
func (TLSOption) Name() string {
	return "tls"
}

func (t TLSOption) ServerOption(cfg *conf.Configuration) grpc.ServerOption {
	tls := conf.NewTLS(cfg)
	if tls.Cert == "" || tls.Key == "" {
		panic("tls cert or key is empty")
	}
	tc, err := credentials.NewServerTLSFromFile(tls.Cert, tls.Key)
	if err != nil {
		panic(err)
	}
	return grpc.Creds(tc)
}

func (t TLSOption) DialOption(cfg *conf.Configuration) grpc.DialOption {
	tls := conf.NewTLS(cfg)
	if tls.Cert == "" {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	tc, err := credentials.NewClientTLSFromFile(tls.Cert, "")
	if err != nil {
		panic(err)
	}
	return grpc.WithTransportCredentials(tc)
}

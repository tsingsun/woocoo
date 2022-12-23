// Package option provides quick support tools for gRPC server and client options .

package option

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/gzip"
)

// CompressionOption compression option
//
//	compression:
//	  name: gzip
//	  level: 1
//
// TODO add github.com/klauspost/compress to improve speed
type CompressionOption struct {
}

func (CompressionOption) Name() string {
	return "compression"
}

func (CompressionOption) ServerOption(cfg *conf.Configuration) grpc.ServerOption {
	cp := encoding.GetCompressor(cfg.String("name"))
	if cp == nil {
		panic("compression not found,do you forget to import it?")
	}
	if cp.Name() == gzip.Name {
		if cfg.IsSet("level") {
			if err := gzip.SetLevel(cfg.Int("level")); err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func (CompressionOption) DialOption(cfg *conf.Configuration) grpc.DialOption {
	return grpc.WithDefaultCallOptions(grpc.UseCompressor(cfg.String("name")))
}

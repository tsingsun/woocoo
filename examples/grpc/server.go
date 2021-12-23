package main

import (
	"github.com/tsingsun/woocoo/rpc/grpcx"
	_ "github.com/tsingsun/woocoo/rpc/grpcx/registry/etcd3"
)

func main() {
	srv := grpcx.NewBuiltIn()
	if err := srv.Run(); err != nil {
		panic(err)
	}
}

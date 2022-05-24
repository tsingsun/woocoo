package interceptor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"testing"
	"time"
)

func TestRecoveryUnaryServerInterceptorUnary(t *testing.T) {
	cnf := conf.New(conf.WithBaseDir(testdata.BaseDir())).Load()
	addr := "localhost:50053"
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"TimestampFormat": "2006-01-02 15:04:05",
	})
	zl, err := zap.NewProduction()
	assert.NoError(t, err)
	clfg := conf.NewFromParse(p)
	gloger := log.New(zl)
	gloger.AsGlobal()
	gloger.Apply(cnf.Sub("log"))
	zgl := zapgrpc.NewLogger(gloger.Operator())
	grpclog.SetLoggerV2(zgl)
	go func() {
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.ChainUnaryInterceptor(LoggerUnaryServerInterceptor(clfg), RecoveryUnaryServerInterceptor(cnf)),
		}

		s := grpc.NewServer(opts...)
		testproto.RegisterTestServiceServer(s, &testproto.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("failed to listen: %v", err)
			return
		}
		if err := s.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	time.Sleep(time.Second)

	var copts []grpc.DialOption
	copts = append(copts, grpc.WithBlock(), grpc.WithInsecure())
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	client := testproto.NewTestServiceClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	_, err = client.PingPanic(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	if err == nil {
		t.Error("must error")
	}
}

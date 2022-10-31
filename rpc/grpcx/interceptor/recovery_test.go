package interceptor

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/logtest"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"strings"
	"testing"
)

func TestRecoveryUnaryServerInterceptor(t *testing.T) {
	wctest.InitGlobalLogger(false)
	zgl := zapgrpc.NewLogger(log.Global().Logger().Logger)
	grpclog.SetLoggerV2(zgl)
	gs, addr := testproto.NewPingGrpcService(t, grpc.ChainUnaryInterceptor(
		LoggerUnaryServerInterceptor(conf.NewFromStringMap(map[string]interface{}{})),
		RecoveryUnaryServerInterceptor(conf.NewFromStringMap(map[string]interface{}{})),
	))
	require.NotNil(t, gs)
	defer gs.Stop()

	conn, client := testproto.NewPingGrpcClient(t, context.Background(), addr)
	defer conn.Close()

	t.Run("stacktrace", func(t *testing.T) {
		logdata := wctest.InitBuffWriteSyncer()
		log.Component(AccessLogComponentName).SetLogger(log.Global().Logger())
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.LastLine()
		require.Contains(t, last, "grpc panic error")
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "testproto/grpc_testing.go")
	})
	t.Run("disableStacktrace", func(t *testing.T) {
		wctest.InitGlobalLogger(true)
		logdata := wctest.InitBuffWriteSyncer()
		log.Component(AccessLogComponentName).SetLogger(log.Global().Logger())
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.LastLine()
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "testproto/grpc_testing.go")
	})
}

func TestRecoveryUnaryServerInterceptorWithoutLogger(t *testing.T) {
	logdata := wctest.InitBuffWriteSyncer()
	gs, addr := testproto.NewPingGrpcService(t, grpc.ChainUnaryInterceptor(
		RecoveryUnaryServerInterceptor(conf.NewFromStringMap(map[string]interface{}{})),
	))
	require.NotNil(t, gs)
	defer gs.Stop()

	conn, client := testproto.NewPingGrpcClient(t, context.Background(), addr)
	defer conn.Close()

	t.Run("stacktrace", func(t *testing.T) {
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.LastLine()
		require.Contains(t, last, "grpc panic error")
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "testproto/grpc_testing.go")
	})
	t.Run("disableStacktrace", func(t *testing.T) {
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		last := logdata.LastLine()
		require.Contains(t, last, "[Recovery from panic]")
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "testproto/grpc_testing.go")
	})
}

func TestHandleRecoverError(t *testing.T) {
	logdata := &logtest.Buffer{}
	log.New(logtest.NewBuffLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).AsGlobal()

	gs, addr := testproto.NewPingGrpcService(t, grpc.ChainUnaryInterceptor(
		RecoveryUnaryServerInterceptor(conf.NewFromStringMap(map[string]interface{}{})),
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			wg := new(errgroup.Group)
			wg.Go(func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = HandleRecoverError(ctx, r)
					}
				}()
				panic("test panic")
			})
			err = wg.Wait()
			return
		},
	))
	require.NotNil(t, gs)
	defer gs.Stop()

	conn, client := testproto.NewPingGrpcClient(t, context.Background(), addr)
	defer conn.Close()

	t.Run("panic", func(t *testing.T) {
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{})
		require.Error(t, err)
		last := logdata.LastLine()
		require.Contains(t, last, "[Recovery from panic]")
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "interceptor.TestHandleRecoverError")
	})
}

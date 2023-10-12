package interceptor

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/logtest"
	"github.com/tsingsun/woocoo/test/testproto"
	"github.com/tsingsun/woocoo/test/wctest"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestRecoveryUnaryServerInterceptor(t *testing.T) {
	wctest.InitGlobalLogger(false)
	gs, addr := testproto.NewPingGrpcService(t, grpc.ChainUnaryInterceptor(
		AccessLogger{}.UnaryServerInterceptor(conf.NewFromStringMap(map[string]any{})),
		Recovery{}.UnaryServerInterceptor(conf.NewFromStringMap(map[string]any{})),
	))
	require.NotNil(t, gs)
	defer gs.Stop()

	conn, client := testproto.NewPingGrpcClient(t, context.Background(), addr)
	defer conn.Close()

	t.Run("stacktrace", func(t *testing.T) {
		logdata := wctest.InitBuffWriteSyncer()
		log.Component(AccessLogComponentName).SetLogger(log.Global().Logger(log.WithOriginalLogger()))
		_, err := client.PingPanic(context.Background(), &testproto.PingRequest{
			Value: t.Name(),
		})
		require.Error(t, err)
		// only one component
		last := logdata.LastLine()
		assert.Equal(t, 1, strings.Count(last, `"component"`))
		require.Contains(t, last, "grpc panic error")
		line := strings.Split(last, "\\n\\t")[1]
		require.Contains(t, line, "testproto/grpc_testing.go")
	})
	t.Run("disableStacktrace", func(t *testing.T) {
		wctest.InitGlobalLogger(true)
		logdata := wctest.InitBuffWriteSyncer()
		log.Component(AccessLogComponentName).SetLogger(log.Global().Logger(log.WithOriginalLogger()))
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
		Recovery{}.UnaryServerInterceptor(conf.NewFromStringMap(map[string]any{})),
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
		Recovery{}.UnaryServerInterceptor(conf.NewFromStringMap(map[string]any{})),
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
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

func TestRecoveryStreamServerInterceptor(t *testing.T) {
	interceptor := Recovery{}
	assert.Equal(t, "recovery", interceptor.Name())
	logdata := &logtest.Buffer{}
	log.New(logtest.NewBuffLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).AsGlobal()

	t.Run("no panic", func(t *testing.T) {
		handlerCalled := false
		stream := &mockStream{}
		handler := func(srv any, stream grpc.ServerStream) error {
			handlerCalled = true
			return nil
		}
		err := interceptor.StreamServerInterceptor(conf.New())(nil, stream, nil, handler)
		require.NoError(t, err)
		assert.True(t, handlerCalled)
	})

	t.Run("panic", func(t *testing.T) {
		panicErr := errors.New("panic error")
		handler := func(srv any, stream grpc.ServerStream) error {
			panic(panicErr)
		}
		err := interceptor.StreamServerInterceptor(conf.New())(nil, &mockStream{}, nil, handler)
		assert.Equal(t, codes.Unknown, status.Code(err))
		assert.Equal(t, "panic error", err.Error())
	})
}

type mockStream struct {
	md metadata.MD
}

func (s *mockStream) SetHeader(metadata.MD) error  { return nil }
func (s *mockStream) SendHeader(metadata.MD) error { return nil }
func (s *mockStream) SetTrailer(metadata.MD)       {}
func (s *mockStream) Context() context.Context {
	if s.md == nil {
		return context.Background()
	}
	return metadata.NewIncomingContext(context.Background(), s.md)
}
func (s *mockStream) SendMsg(m any) error { return nil }
func (s *mockStream) RecvMsg(m any) error { return nil }

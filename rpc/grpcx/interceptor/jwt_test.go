package interceptor

import (
	"context"
	"crypto/tls"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

var (
	hs256Token    = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o"
	Hs256TokenCnf = conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
		"signingKey":  "secret",
		"tokenLookup": "authorization",
	}))
)

type mockCredential struct{}

// server side can get by metadata.FromIncomingContext .
func (c mockCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + hs256Token,
	}, nil
}

func (c mockCredential) RequireTransportSecurity() bool {
	return false
}

func TestJWTUnaryServerInterceptor(t *testing.T) {
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	zgl := zapgrpc.NewLogger(logger)
	grpclog.SetLoggerV2(zgl)

	cert, err := tls.LoadX509KeyPair(testdata.Path("x509/server.crt"), testdata.Path("x509/server.key"))
	require.NoError(t, err)
	gs, addr := testproto.NewPingGrpcService(t,
		grpc.UnaryInterceptor(JWT{}.UnaryServerInterceptor(Hs256TokenCnf)),
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)))
	assert.NotNil(t, gs)
	defer gs.Stop()

	ccreds, err := credentials.NewClientTLSFromFile(testdata.Path("x509/server.crt"), "localhost")
	assert.NoError(t, err)
	// Set up the credentials for the connection.
	copts := []grpc.DialOption{
		// In addition to the following grpc.DialOption, callers may also use
		// the grpc.CallOption grpc.PerRPCCredentials with the RPC invocation
		// itself.
		// See: https://godoc.org/google.golang.org/grpc#PerRPCCredentials
		grpc.WithPerRPCCredentials(mockCredential{}),
		// oauth.NewOauthAccess requires the configuration of transport
		// credentials.
		grpc.WithTransportCredentials(ccreds),
		grpc.WithBlock(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	conn, client := testproto.NewPingGrpcClient(t, ctx, addr, copts...)
	defer cancel()
	defer conn.Close()

	resp, err := client.Ping(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	require.NoError(t, err)
	assert.EqualValues(t, resp.Counter, 42)
}

func TestJWT_SteamServerInterceptor(t *testing.T) {
	jwtInterceptor := JWT{}
	assert.Equal(t, "jwt", jwtInterceptor.Name())

	t.Run("no token", func(t *testing.T) {
		stream := &mockStream{md: metadata.New(map[string]string{})}
		err := jwtInterceptor.SteamServerInterceptor(Hs256TokenCnf)(nil, stream, nil, nil)
		require.Equal(t, auth.ErrJWTMissing, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		stream := &mockStream{md: metadata.New(map[string]string{"authorization": "Bearer invalid_token"})}
		err := jwtInterceptor.SteamServerInterceptor(Hs256TokenCnf)(nil, stream, nil, nil)
		assert.ErrorIs(t, err, jwt.ErrTokenMalformed)
	})

	t.Run("valid token", func(t *testing.T) {
		stream := &mockStream{md: metadata.New(map[string]string{"authorization": "Bearer " + hs256Token})}
		handlerCalled := false
		handler := func(srv any, stream grpc.ServerStream) error {
			_, ok := JWTFromIncomingContext(stream.Context())
			require.True(t, ok)
			handlerCalled = true
			return nil
		}
		err := jwtInterceptor.SteamServerInterceptor(Hs256TokenCnf)(nil, stream, nil, handler)
		require.NoError(t, err)
		assert.True(t, handlerCalled)
	})
}

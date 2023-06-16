package interceptor

import (
	"context"
	"crypto/tls"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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
	fetchToken := &oauth2.Token{
		AccessToken: hs256Token,
	}
	// Set up the credentials for the connection.
	perRPC := oauth.NewOauthAccess(fetchToken)
	copts := []grpc.DialOption{
		// In addition to the following grpc.DialOption, callers may also use
		// the grpc.CallOption grpc.PerRPCCredentials with the RPC invocation
		// itself.
		// See: https://godoc.org/google.golang.org/grpc#PerRPCCredentials
		grpc.WithPerRPCCredentials(perRPC),
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
	assert.NoError(t, err)
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
		_, ok := err.(*jwt.ValidationError)
		assert.True(t, ok)
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

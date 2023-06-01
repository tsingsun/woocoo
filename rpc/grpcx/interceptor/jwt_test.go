package interceptor

import (
	"context"
	"crypto/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"testing"
	"time"
)

func TestJWTUnaryServerInterceptor(t *testing.T) {
	var (
		hs256Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o"
	)
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	zgl := zapgrpc.NewLogger(logger)
	grpclog.SetLoggerV2(zgl)

	acfg := conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
		"signingKey":  "secret",
		"tokenLookup": "authorization",
	}))

	cert, err := tls.LoadX509KeyPair(testdata.Path("x509/server.crt"), testdata.Path("x509/server.key"))
	require.NoError(t, err)
	gs, addr := testproto.NewPingGrpcService(t,
		grpc.UnaryInterceptor(JWT{}.UnaryServerInterceptor(acfg)),
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

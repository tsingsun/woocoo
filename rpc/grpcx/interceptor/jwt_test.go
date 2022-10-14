package interceptor

import (
	"context"
	"crypto/tls"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"net"
	"testing"
	"time"
)

var (
	addr       = "127.0.0.1:50051"
	hs256Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o"
)

func TestJWTUnaryServerInterceptor(t *testing.T) {
	acfg := conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
		"signingKey":  "secret",
		"tokenLookup": "authorization",
	}))

	go func() {
		cert, err := tls.LoadX509KeyPair(testdata.Path("x509/test.pem"), testdata.Path("x509/test.key"))
		assert.NoError(t, err)
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.UnaryInterceptor(JWTUnaryServerInterceptor(acfg)),
			// Enable TLS for all incoming connections.
			grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		}
		s := grpc.NewServer(opts...)
		testproto.RegisterTestServiceServer(s, &testproto.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		assert.NoError(t, err)
		if err := s.Serve(lis); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(time.Second)
	ccreds, err := credentials.NewClientTLSFromFile(testdata.Path("x509/test.pem"), "*.woocoo.com")
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
	}
	copts = append(copts, grpc.WithBlock())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, copts...)
	assert.NoError(t, err)
	client := testproto.NewTestServiceClient(conn)
	resp, err := client.Ping(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	assert.NoError(t, err)
	assert.EqualValues(t, resp.Counter, 42)
}

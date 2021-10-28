package auth_test

import (
	"context"
	"crypto/tls"
	"github.com/golang-jwt/jwt/v4"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/user"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/auth"
	"github.com/tsingsun/woocoo/test"
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
	cnf            = testdata.Config
	addr           = "localhost:50051"
	hs256BadHeader = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE1MTYyMzkxMjJ9.JcRoPW5fA44i7vuGyXGXKHuAfZYly_uFGs5FznyPJBc"
	hs256OkToken   = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE3MTYyMzkwMjJ9.kiW0BWa5S93F401V0N5wPZkuJS5L2cxzGZDTeDnne2I"
)

func TestAuth_UnaryServerInterceptor(t *testing.T) {
	auth.IdentityHandler = func(ctx context.Context, claims jwt.MapClaims) user.Identity {
		return &user.User{
			user.IDKey:    claims["sub"].(string),
			user.OrgIDKey: claims["X-Org-Id"].(string),
		}
	}
	ints, err := auth.New()
	p := conf.NewParserFromStringMap(map[string]interface{}{
		"signingAlgorithm": "HS256",
		"realm":            "auth",
		"secret":           "123456",
		"TenantHeader":     "wc",
	})
	acfg := cnf.CutFromParser(p)
	ints.Apply(acfg)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		cert, err := tls.LoadX509KeyPair(testdata.Path("x509/server_cert.pem"), testdata.Path("x509/server_key.pem"))
		if err != nil {
			t.Errorf("failed to load key pair: %s", err)
		}
		opts := []grpc.ServerOption{
			// The following grpc.ServerOption adds an interceptor for all unary
			// RPCs. To configure an interceptor for streaming RPCs, see:
			// https://godoc.org/google.golang.org/grpc#StreamInterceptor
			grpc.UnaryInterceptor(ints.UnaryServerInterceptor()),
			// Enable TLS for all incoming connections.
			grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		}
		s := grpc.NewServer(opts...)
		testproto.RegisterTestServiceServer(s, &test.TestPingService{})
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	time.Sleep(2000)
	ccreds, err := credentials.NewClientTLSFromFile(testdata.Path("x509/ca_cert.pem"), "x.test.example.com")
	if err != nil {
		t.Fatalf("failed to load credentials: %v", err)
	}
	fetchToken := &oauth2.Token{
		AccessToken: hs256OkToken,
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
	conn, err := grpc.Dial(addr, copts...)
	if err != nil {
		t.Fatal(err)
	}
	client := testproto.NewTestServiceClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	resp, err := client.Ping(context.Background(), &testproto.PingRequest{
		Value: t.Name(),
	})
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}

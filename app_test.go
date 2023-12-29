package woocoo

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/wctest"
	"github.com/tsingsun/woocoo/web"
	"log"
	"testing"
	"time"

	mock "github.com/tsingsun/woocoo/test/mock/registry"
)

func TestApp(t *testing.T) {
	mock.RegisterDriver(map[string]*registry.ServiceInfo{
		"helloworld.Greeter": {
			Name:    "helloworld.Greeter",
			Version: "1.0",
			Host:    "127.0.0.1",
			Port:    20000,
		},
	})
	cnf := wctest.Configuration()
	app := New(WithAppConfiguration(cnf))
	websrv := web.New(web.WithConfiguration(app.AppConfiguration().Sub("web")))
	grpcsrv := grpcx.New(grpcx.WithConfiguration(app.AppConfiguration().Sub("grpc")))
	time.AfterFunc(time.Second*2, func() {
		t.Log("stop")
		app.Stop()
	})
	app.RegisterServer(websrv, grpcsrv)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}

type (
	server1 struct {
	}
	server2 struct {
		timeout bool
	}
)

func (s *server1) Start(context.Context) error {
	return nil
}

func (s *server1) Stop(context.Context) error {
	return nil
}

func (s *server2) Start(context.Context) error {
	return nil
}

func (s *server2) Stop(context.Context) error {
	if s.timeout {
		log.Print("server2 stop timeout")
	}
	return nil
}

func TestMiniApp(t *testing.T) {
	type args struct {
		ctx     context.Context
		timeout time.Duration
		servers []Server
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{"background ctx", args{ctx: context.Background(), timeout: 0, servers: []Server{&server1{}, &server2{}}}, assert.NoError},
		{"timeout ctx", args{ctx: context.Background(), timeout: time.Millisecond * 500, servers: []Server{&server1{}, &server2{timeout: true}}},
			func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, context.DeadlineExceeded)
			}},
		{"parent timeout ctx", args{ctx: func() context.Context {
			ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*500) //nolint:govet
			return ctx
		}(), timeout: 0, servers: []Server{&server1{}, &server2{timeout: true}}}, func(t assert.TestingT, err error, i ...any) bool {
			return assert.ErrorIs(t, err, context.DeadlineExceeded)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run, stop := MiniApp(tt.args.ctx, tt.args.timeout, tt.args.servers...)
			time.AfterFunc(time.Second, func() {
				t.Log("force stop")
				stop()
			})
			if err := run(); tt.wantErr != nil {
				tt.wantErr(t, err)
			}
		})
	}
}

func TestGroupRun(t *testing.T) {
	type args struct {
		ctx     context.Context
		timeout time.Duration
		start   func(ctx context.Context) error
		stop    func(ctx context.Context) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "background ctx",
			args: func() args {
				ctx, cancel := context.WithCancel(context.Background())
				return args{
					ctx:     ctx,
					timeout: 0,
					start: func(_ context.Context) error {
						for {
							select {
							case <-ctx.Done():
								return nil
							default:
								time.Sleep(time.Millisecond * 100)
							}
						}
					},
					stop: func(ctx context.Context) error {
						cancel()
						return nil
					},
				}
			}(), wantErr: assert.NoError,
		},
		{
			name: "run done",
			args: func() args {
				return args{
					ctx:     context.Background(),
					timeout: 0,
					start: func(_ context.Context) error {
						return nil
					},
					stop: func(ctx context.Context) error {
						return nil
					},
				}
			}(), wantErr: assert.NoError,
		},
		{
			name: "run timeout",
			args: func() args {
				return args{
					ctx:     context.Background(),
					timeout: time.Millisecond * 500,
					start: func(_ context.Context) error {
						return nil
					},
					stop: func(ctx context.Context) error {
						return nil
					},
				}
			}(), wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run, stop := GroupRun(tt.args.ctx, tt.args.timeout, tt.args.start, tt.args.stop)
			time.AfterFunc(time.Second*2, func() {
				t.Log("force stop")
				stop()
			})
			err := run()
			if tt.wantErr != nil {
				tt.wantErr(t, err)
			}
		})
	}
}

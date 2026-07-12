package woocoo

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/wctest"
	"github.com/tsingsun/woocoo/web"

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
	app := New(WithAppConfiguration(cnf), WithInterval(time.Millisecond*100))
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

func TestMiniServer(t *testing.T) {
	t.Run("with start and stop functions", func(t *testing.T) {
		started, stopped := false, false
		s := &miniServer{
			start: func(ctx context.Context) error {
				started = true
				return nil
			},
			stop: func(ctx context.Context) error {
				stopped = true
				return nil
			},
		}
		assert.NoError(t, s.Start(context.Background()))
		assert.True(t, started)
		assert.NoError(t, s.Stop(context.Background()))
		assert.True(t, stopped)
	})

	t.Run("start returns error", func(t *testing.T) {
		s := &miniServer{
			start: func(ctx context.Context) error {
				return errors.New("start error")
			},
		}
		assert.Error(t, s.Start(context.Background()))
	})
}

func TestGroup(t *testing.T) {
	t.Run("empty group returns nil", func(t *testing.T) {
		g := &Group{}
		assert.NoError(t, g.Run())
	})

	t.Run("run exits when any actor returns", func(t *testing.T) {
		var g Group
		actor1Interrupted := false
		actor2Interrupted := false
		done := make(chan struct{})

		g.Add(func() error {
			return nil
		}, func(err error) {
			actor1Interrupted = true
		}).Add(func() error {
			<-done
			return nil
		}, func(err error) {
			actor2Interrupted = true
			close(done)
		})

		err := g.Run()
		assert.NoError(t, err)
		assert.True(t, actor1Interrupted)
		assert.True(t, actor2Interrupted)
	})

	t.Run("run returns first error and passes to interrupt", func(t *testing.T) {
		var g Group
		expectedErr := errors.New("actor error")
		var receivedErr error
		done := make(chan struct{})

		g.Add(func() error {
			return expectedErr
		}, func(err error) {
		}).Add(func() error {
			<-done
			return nil
		}, func(err error) {
			receivedErr = err
			close(done)
		})

		err := g.Run()
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, expectedErr, receivedErr)
	})
}

func TestSync(t *testing.T) {
	app := New()
	err := app.Sync()
	assert.NoError(t, err)
}

package handler

import (
	"bufio"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"github.com/tsingsun/woocoo/test/testlog"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type ResponseWrite struct {
	httptest.ResponseRecorder
}

func (r ResponseWrite) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	panic("implement me")
}

func (r ResponseWrite) CloseNotify() <-chan bool {
	panic("implement me")
}

func (r ResponseWrite) Status() int {
	return r.Code
}

func (r ResponseWrite) Size() int {
	return 0
}

func (r ResponseWrite) Written() bool {
	return true
}

func (r ResponseWrite) WriteHeaderNow() {
	r.WriteHeader(r.Status())
}

func (r ResponseWrite) Pusher() http.Pusher {
	panic("implement me")
}

func TestHandleRecoverError(t *testing.T) {
	type args struct {
		c   *gin.Context
		err any
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "with logger error",
			args: args{
				c: &gin.Context{
					Request: httptest.NewRequest("GET", "/", nil),
					Writer:  &ResponseWrite{},
					Keys: map[string]any{
						AccessLogComponentName: log.NewCarrier(),
					},
				},
				err: errors.New("public error"),
			},
			want: func() any {
				logdata := testlog.InitStringWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				assert.Len(t, ss.Entry, 0)
				fc := GetLogCarrierFromGinContext(i[1].(*gin.Context))
				assert.NotNil(t, fc)
				assert.Len(t, fc.Fields, 3)
				return true
			},
		},
		{
			name: "without logger",
			args: args{
				c: &gin.Context{
					Request: httptest.NewRequest("GET", "/", nil),
					Writer:  &ResponseWrite{},
				},
				err: "panic",
			},
			want: func() any {
				logdata := testlog.InitStringWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "[Recovery from panic]")
				assert.Contains(t, all, "\"component\":\"web\"")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want()
			HandleRecoverError(tt.args.c, tt.args.err, 3)
			if !tt.wantErr(t, nil, want, tt.args.c) {
				return
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	testlog.ApplyGlobal(true)
	type args struct {
		cfg     *conf.Configuration
		handler gin.HandlerFunc
	}
	rargs := func(p any) args {
		return args{
			cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
				"format": "error",
			})),
			handler: func(c *gin.Context) {
				panic(p)
			},
		}
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "panic any",
			args: rargs("panicx"),
			want: func() any {
				testlog.ApplyGlobal(true)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				if !i[1].(bool) {
					assert.Contains(t, all, "internal server error")
				}
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error",
			args: rargs(errors.New("panicx")),
			want: func() any {
				testlog.ApplyGlobal(true)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic any-false",
			args: rargs("panicx"),
			want: func() any {
				testlog.ApplyGlobal(false)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				if !i[1].(bool) {
					assert.Contains(t, all, "internal server error")
				}
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error-false",
			args: rargs(errors.New("panicx")),
			want: func() any {
				testlog.ApplyGlobal(false)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
	}
	withoutLogger := true
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			want := tt.want()
			gin.SetMode(gin.ReleaseMode)
			srv := gin.New()
			srv.Use(Recovery().ApplyFunc(tt.args.cfg))
			srv.GET("/", func(c *gin.Context) {
				tt.args.handler(c)
			})
			srv.ServeHTTP(w, r)

			if !tt.wantErr(t, nil, want, withoutLogger) {
				return
			}
		})
	}
}

func TestRecoveryMiddleware_WithLogger(t *testing.T) {
	type args struct {
		cfg     *conf.Configuration
		handler gin.HandlerFunc
	}
	rargs := args{
		cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
			"format": "error",
		})),
		handler: func(c *gin.Context) {
			panic("panicx")
		},
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "panic any",
			args: rargs,
			want: func() any {
				testlog.ApplyGlobal(true)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error",
			args: rargs,
			want: func() any {
				testlog.ApplyGlobal(true)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic any-false",
			args: rargs,
			want: func() any {
				testlog.ApplyGlobal(false)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error-false",
			args: rargs,
			want: func() any {
				testlog.ApplyGlobal(false)
				return testlog.InitStringWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				line := strings.Split(ss.Entry[0], "\\n\\t")[0]
				assert.Contains(t, line, "handler.TestRecoveryMiddleware")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"-with-access-logger", func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			want := tt.want()
			gin.SetMode(gin.ReleaseMode)
			srv := gin.New()
			middleware := AccessLog().ApplyFunc(tt.args.cfg)
			srv.Use(middleware, Recovery().ApplyFunc(tt.args.cfg))
			srv.GET("/", func(c *gin.Context) {
				tt.args.handler(c)
			})
			srv.ServeHTTP(w, r)
			if !tt.wantErr(t, nil, want) {
				return
			}
		})
	}
}

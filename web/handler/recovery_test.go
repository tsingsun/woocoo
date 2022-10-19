package handler

import (
	"bufio"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ResponseWrite struct {
	httptest.ResponseRecorder
}

func (r ResponseWrite) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (r ResponseWrite) CloseNotify() <-chan bool {
	//TODO implement me
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
	//TODO implement me
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
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
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
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
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
			HandleRecoverError(tt.args.c, tt.args.err)
			if !tt.wantErr(t, nil, want, tt.args.c) {
				return
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	log.Global().Apply(conf.NewFromBytes([]byte(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: false`)))
	type args struct {
		cfg     *conf.Configuration
		handler gin.HandlerFunc
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "panic any",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic("panicx")
				},
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).
					AsGlobal().DisableStacktrace = true
				return logdata
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
				assert.Contains(t, ss.Entry[0], "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic(errors.New("public error"))
				},
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).
					AsGlobal().DisableStacktrace = false
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "public error")
				assert.Contains(t, ss.Entry[0], "handler.TestRecoveryMiddleware")
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
	log.Global().Apply(conf.NewFromBytes([]byte(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: false`)))
	type args struct {
		cfg     *conf.Configuration
		handler gin.HandlerFunc
	}
	tests := []struct {
		name    string
		args    args
		want    func() any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "panic any",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic("panicx")
				},
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).
					AsGlobal().DisableStacktrace = false
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, ss.Entry[0], "handler.TestRecoveryMiddleware")
				return true
			},
		},
		{
			name: "panic error",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic(errors.New("public error"))
				},
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).
					AsGlobal().DisableStacktrace = false
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "public error")
				assert.Contains(t, ss.Entry[0], "handler.TestRecoveryMiddleware")
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

package handler_test

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"github.com/tsingsun/woocoo/web/handler"
	"go.uber.org/zap"
	"net/http/httptest"
	"testing"
)

func TestLoggerMiddleware(t *testing.T) {
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
			name: "allType but fatal",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					l := log.Component(handler.AccessLogComponentName)
					l.Debug("error", zap.Error(errors.New("debugx")))
					l.Info("error", zap.Error(errors.New("infox")))
					l.Warn("error", zap.Error(errors.New("warnx")))
					l.Error("error", zap.Error(errors.New("errorx")))
					l.DPanic("error", zap.Error(errors.New("dpanicx")))
					l.Panic("error", zap.Error(errors.New("panicx")))
					l.Fatal("error", zap.Error(errors.New("fatalx")))
					e := c.Error(errors.New("error"))
					e.Type = gin.ErrorTypePublic
				},
			},
			want: func() any {
				logdata := &test.StringWriteSyncer{}
				log.New(test.NewStringLogger(logdata)).AsGlobal()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				assert.Contains(t, all, "debugx")
				assert.Contains(t, all, "infox")
				assert.Contains(t, all, "warnx")
				assert.Contains(t, all, "errorx")
				assert.Contains(t, all, "dpanicx")
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "stacktrace")
				// panic
				assert.Contains(t, all, handler.AccessLogComponentName)
				return true
			},
		},
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
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*test.StringWriteSyncer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panicx")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "internal server error")
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
				log.New(test.NewStringLogger(logdata)).AsGlobal().DisableStacktrace = true
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
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			want := tt.want()
			gin.SetMode(gin.ReleaseMode)
			srv := gin.New()
			middleware := handler.AccessLog().ApplyFunc(tt.args.cfg)
			srv.Use(middleware, handler.Recovery().ApplyFunc(tt.args.cfg))
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

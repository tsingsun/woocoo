package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/testco/logtest"
	"github.com/tsingsun/woocoo/testco/wctest"
	"go.uber.org/zap"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
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
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "id,remoteIp,host,method,uri,userAgent,status,error,latency,bytesIn,bytesOut," +
						"path,protocol,referer,latencyHuman,header:Accept,query:q1,form:username," +
						"cookie:c1,context:ctx1",
				}),
				handler: func(c *gin.Context) {
					l := log.Component(log.WebComponentName)
					l.Debug("error", zap.Error(errors.New("debugx")))
					l.Info("error", zap.Error(errors.New("infox")))
					l.Warn("error", zap.Error(errors.New("warnx")))
					l.Error("error", zap.Error(errors.New("errorx")))
					l.DPanic("error", zap.Error(errors.New("dpanicx")))
					c.Set("ctx1", "from context")
					l.Panic("error", zap.Error(errors.New("panicx")))
					e := c.Error(errors.New("errorx"))
					e.Type = gin.ErrorTypePublic
				},
			},
			want: func() any {
				wctest.InitGlobalLogger(false)
				return wctest.InitBuffWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				ss := i[0].(*logtest.Buffer)
				lines := ss.Lines()
				assert.Contains(t, lines[0], "debugx")
				assert.Contains(t, lines[1], "infox")
				assert.Contains(t, lines[2], "warnx")
				assert.Contains(t, lines[3], "errorx")
				assert.Contains(t, strings.Split(lines[3], "\\n\\t")[1], "handler/logger_test.go")
				assert.Contains(t, lines[4], "dpanicx")
				assert.Contains(t, lines[5], "panicx")
				assert.Contains(t, strings.Split(lines[5], "\\n\\t")[1], "handler/logger_test.go")
				// panic trigger in zap,so check the zap file
				assert.Contains(t, lines[6], AccessLogComponentName)
				assert.Contains(t, lines[6], "internal server error")
				assert.Contains(t, strings.Split(lines[6], "\\n\\t")[1], "zapcore/entry.go")
				assert.Contains(t, lines[6], "from context")
				return true
			},
		},
		{
			name: "private error",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "error",
				}),
				handler: func(c *gin.Context) {
					ce := c.Error(errors.New("private error"))
					ce.Type = gin.ErrorTypePrivate
				},
			},
			want: func() any {
				logdata := wctest.InitBuffWriteSyncer(zap.AddStacktrace(zap.ErrorLevel))
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "private error")
				assert.NotContains(t, all, "stacktrace")
				return true
			},
		},
		{
			name: "panic error",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic(errors.New("public error"))
				},
			},

			want: func() any {
				logdata := wctest.InitBuffWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "public error")
				return true
			},
		},
		{
			name: "skip path",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"exclude": []string{"/"},
				}),
				handler: func(c *gin.Context) {
				},
			},
			want: func() any {
				logdata := wctest.InitBuffWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				assert.Emptyf(t, all, "skip path must not log")
				return true
			},
		},
		{
			name: "skip path must not ignore error",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"exclude": []string{"/"},
				}),
				handler: func(c *gin.Context) {
					c.Error(errors.New("errorx"))
				},
			},
			want: func() any {
				logdata := wctest.InitBuffWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				assert.Contains(t, all, "errorx")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			want := tt.want()
			accessLog := NewAccessLog()
			middleware := accessLog.ApplyFunc(tt.args.cfg)
			srv := gin.New()
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

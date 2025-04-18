package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/logtest"
	"github.com/tsingsun/woocoo/test/wctest"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	type args struct {
		cfg     *conf.Configuration
		request *http.Request
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
						"path,protocol,referer,latencyHuman,header:accept,query:q1,form:username," +
						"cookie:c1,cookie:noexist,,context:ctx1",
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
			name: "level should increase",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "error,header:accept,",
					"level":  "info",
				}),
				handler: func(c *gin.Context) {
				},
			},
			want: func() any {
				logdata := wctest.InitBuffWriteSyncer(zap.IncreaseLevel(zap.WarnLevel))
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				assert.Empty(t, all)
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
		{
			name: "log body in",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "bodyIn",
				}),
				request: httptest.NewRequest("POST", "/?query=1", strings.NewReader(`
{
	"username": "testuser",
	"password": "testpass",
	"age": 28,
	"hobbies": ["reading", "swimming", "coding"]
}`)),
				handler: func(c *gin.Context) {
					// get body correct
					var user struct {
						Username string   `json:"username"`
						Password string   `json:"password"`
						Age      int      `json:"age"`
						Hobbies  []string `json:"hobbies"`
					}
					if err := c.ShouldBindJSON(&user); err != nil {
						_ = c.AbortWithError(http.StatusInternalServerError, err)
					}
					assert.Equal(t, "testuser", user.Username)
				},
			},
			want: func() any {
				logdata := wctest.InitBuffWriteSyncer()
				return logdata
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				r := i[1].(*httptest.ResponseRecorder)
				assert.Equal(t, http.StatusOK, r.Code)
				ss := i[0].(*logtest.Buffer)
				all := ss.String()
				assert.Contains(t, all, `testuser`)
				return true
			},
		},
		{
			name: "log empty body in",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "bodyIn",
				}),
				request: httptest.NewRequest("POST", "/", strings.NewReader("")),
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
				assert.NotContains(t, all, `bodyIn: ""`)
				return true
			},
		},
		{
			name: "log nil body in",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"format": "bodyIn",
				}),
				request: httptest.NewRequest("POST", "/", nil),
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
				assert.NotContains(t, all, `bodyIn: ""`)
				return true
			},
		},
		{
			name: "append format",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"appendFormat": "header:accept,",
				}),
				request: httptest.NewRequest("GET", "/?query=1", nil),
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
				assert.Contains(t, all, `header:accept`)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.args.request
			if r == nil {
				r = httptest.NewRequest("GET", "/?query=1", nil)
				r.Header.Set("accept", "*")
				r.AddCookie(&http.Cookie{Name: "c1", Domain: "localhost", Value: "cookievalue"})
			}
			w := httptest.NewRecorder()
			want := tt.want()
			accessLog := AccessLog()
			assert.Equal(t, AccessLogName, accessLog.Name())
			middleware := accessLog.ApplyFunc(tt.args.cfg)
			srv := gin.New()
			srv.Use(middleware, Recovery().ApplyFunc(tt.args.cfg))
			srv.GET("/", func(c *gin.Context) {
				tt.args.handler(c)
			})
			srv.POST("/", func(c *gin.Context) {
				tt.args.handler(c)
			})
			srv.ServeHTTP(w, r)
			if !tt.wantErr(t, nil, want, w) {
				return
			}
		})
	}
}

package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"net/http"
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
		name string
		args args
		want func() any
	}{
		{
			name: "error",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					log.Global().Error("test", zap.Error(errors.New("test")))
					e := c.Error(errors.New("error"))
					e.Type = gin.ErrorTypePublic
				},
			},

			want: func() any {
				return http.StatusOK
			},
		},
		{
			name: "info",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {},
			},

			want: func() any {
				return http.StatusOK
			},
		},
		{
			name: "panic",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic("test")
				},
			},

			want: func() any {
				// reset the logger
				log.Global().Apply(conf.NewFromBytes([]byte(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: false`)))
				return http.StatusInternalServerError
			},
		},
		{
			name: "panic-DisableStacktrace",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"format": "error",
				})),
				handler: func(c *gin.Context) {
					panic("test")
				},
			},

			want: func() any {
				// reset the logger
				log.Global().Apply(conf.NewFromBytes([]byte(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: true`)))
				return http.StatusInternalServerError
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
			middleware := AccessLog().ApplyFunc(tt.args.cfg)
			srv.Use(middleware, Recovery().ApplyFunc(tt.args.cfg))
			srv.GET("/", func(c *gin.Context) {
				tt.args.handler(c)
			})
			srv.ServeHTTP(w, r)
			assert.EqualValues(t, want, w.Code)
		})
	}
}

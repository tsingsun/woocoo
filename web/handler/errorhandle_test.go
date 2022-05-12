package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorHandleMiddleware_ApplyFunc(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name   string
		handle gin.HandlerFunc
		args   args
		check  func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name: "standError", handle: func(c *gin.Context) {
				c.Error(fmt.Errorf("standError")) //nolint:errcheck
			}, args: args{cfg: conf.NewFromParse(conf.NewParserFromStringMap(nil))},
			check: func(t *testing.T, r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.Contains(t, r.Body.String(), "standError")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorHandleMiddleware{}
			gin.SetMode(gin.ReleaseMode)
			web := gin.New()
			got := e.ApplyFunc(tt.args.cfg)
			web.Use(got)
			web.GET("/", tt.handle)
			req := httptest.NewRequest("GET", "/", nil)
			res := httptest.NewRecorder()
			web.ServeHTTP(res, req)
			tt.check(t, res)
		})
	}
}

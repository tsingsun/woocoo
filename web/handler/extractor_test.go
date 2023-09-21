package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateExtractors(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	tests := []struct {
		name       string
		lookups    string
		authScheme string
		ctx        *gin.Context
		want       []string
		wantErr    bool
	}{
		{
			name:    "empty lookups",
			lookups: "",
		},
		{
			name:    "invalid lookup source",
			lookups: "invalid:param",
			wantErr: true,
		},
		{
			name:    "query extractor",
			lookups: "query:access_token",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("GET", "/?access_token=abc", nil)
				return c
			}(),
			want: []string{"abc"},
		},
		{
			name:    "param extractor",
			lookups: "param:id",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Params = gin.Params{{Key: "id", Value: "123"}}
				return c
			}(),
			want: []string{"123"},
		},
		{
			name:    "cookie extractor",
			lookups: "cookie:token",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("GET", "/", nil)
				c.Request.AddCookie(&http.Cookie{Name: "token", Value: "abc"})
				return c
			}(),
			want: []string{"abc"},
		},
		{
			name:    "form extractor",
			lookups: "form:code",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("POST", "/", nil)
				c.Request.PostForm = map[string][]string{"code": {"123"}}
				return c
			}(),
			want: []string{"123"},
		},
		{
			name:    "header extractor",
			lookups: "header:Authorization",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("POST", "/", nil)
				c.Request.Header.Set("Authorization", "Bearer abc")
				return c
			}(),
			want: []string{"Bearer abc"},
		},
		{
			name:       "header extractor with scheme",
			lookups:    "header:Authorization",
			authScheme: "Bearer",
			ctx: func() *gin.Context {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("POST", "/", nil)
				c.Request.Header.Set("Authorization", "Bearer abc")
				return c
			}(),
			want: []string{"abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gots, err := CreateExtractors(tt.lookups, tt.authScheme)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			for i, extractor := range gots {
				got, err := extractor(tt.ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.want[i], got[i])
			}
		})
	}
}

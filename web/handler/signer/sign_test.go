package signer

import (
	"context"
	"errors"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/cache/lfu"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web/handler"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewMiddleware(t *testing.T) {
	type args struct {
		name string
		opts []handler.MiddlewareOption
	}
	tests := []struct {
		name  string
		args  args
		check func(*Middleware)
		panic bool
	}{
		{
			name: "options",
			args: args{
				name: "",
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func(config any) {
						c := config.(*Config)
						c.Interval = 10 * time.Minute
					}),
				},
			},
			check: func(middleware *Middleware) {
				assert.NotNil(t, middleware.config)
				assert.Equal(t, 10*time.Minute, middleware.config.Interval)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() {
					NewMiddleware("", tt.args.opts...)
				})
				return
			}
			middleware := NewMiddleware("", tt.args.opts...)
			tt.check(middleware)
		})
	}
}

func TestMiddleware_ApplyFunc(t *testing.T) {
	cnfstr := `
signerConfig:
  authLookup: "header:Authorization"
  authScheme: "TEST-HMAC-SHA1"
  authHeaders: ["jsapi_ticket","timestamp","noncestr"]
  authHeaderDelimiter: ";"
  signedLookups: 
  - jsapi_ticket: header:Authorization>Bearer
  - timestamp:
  - noncestr:
  - url: CanonicalUri
  delimiter: "&"
  nonceKey: "noncestr"
  unsignedPayload: true
interval: 5s
exclude: ["/skip"]
`
	t.Run("empty authLookup", func(t *testing.T) {
		mid := TokenSigner().(*Middleware)
		cnf := conf.NewFromBytes([]byte(cnfstr))
		cnf.Parser().Set("signerConfig.authLookup", "")
		assert.Panics(t, func() {
			mid.ApplyFunc(cnf)
		})
	})
	t.Run("miss authLookup", func(t *testing.T) {
		mid := TokenSigner().(*Middleware)
		cnf := conf.NewFromBytes([]byte(cnfstr))
		cnf.Parser().Set("signerConfig.authLookup", "XX")
		assert.Panics(t, func() {
			mid.ApplyFunc(cnf)
		})
	})
}

func TestToken_Wechat_OK(t *testing.T) {
	act := "sM4AOVdWfPE4DxkXGEs8VMCPGGVi4C3VM0P37wVUCFvkVAy_90u5h9nbSlYy3-Sl-HhTdfl2fzFy1AOcHKP7qg"
	_, engine := gin.CreateTestContext(httptest.NewRecorder())

	cnf := `
signerConfig:
  authLookup: "header:Authorization"
  authScheme: "TEST-HMAC-SHA1"
  authHeaders: ["jsapi_ticket","timestamp","noncestr"]
  authHeaderDelimiter: ";"
  signedLookups: 
  - jsapi_ticket: header:Authorization>Bearer
  - timestamp:
  - noncestr:
  - url: CanonicalUri
  delimiter: "&"
  nonceKey: "noncestr"
  unsignedPayload: true
interval: 5s
exclude: ["/skip"]
`
	mid := TokenSigner().(*Middleware)
	assert.Equal(t, TokenSignerName, mid.Name())
	mid.config.NowFunc = func() time.Time {
		return time.Unix(1414587457, 0)
	}
	engine.RedirectTrailingSlash = false
	engine.Use(mid.ApplyFunc(conf.NewFromBytes([]byte(cnf))))

	engine.POST("/", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	engine.POST("/skip", func(context *gin.Context) {})
	sig := "0f9de62fce790f9a083d5c99e95740ceb90c27ed"
	t.Run("in header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		authsig := fmt.Sprintf("%s %s=%s;timestamp=%s;noncestr=%s;Signature=%s",
			"TEST-HMAC-SHA1", "jsapi_ticket", act,
			"1414587457", "Wm3WZYTPz0wzccnW", sig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", authsig)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 404, w.Code)
		mid.cache.Del(context.Background(), sig)
	})
	t.Run("token out of header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		authsig := fmt.Sprintf("%s timestamp=%s;noncestr=%s;Signature=%s",
			"TEST-HMAC-SHA1",
			"1414587457", "Wm3WZYTPz0wzccnW", sig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", authsig)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 404, w.Code)
		mid.cache.Del(context.Background(), sig)
	})
	t.Run("skip", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/skip", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
		mid.cache.Del(context.Background(), sig)
	})
	t.Run("wrong ts", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		authsig := fmt.Sprintf("%s timestamp=%s;noncestr=%s;Signature=%s",
			"TEST-HMAC-SHA1",
			"wrong", "Wm3WZYTPz0wzccnW", sig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", authsig)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)

		mid.timestampExtractor = func(c *gin.Context) ([]string, error) {
			return nil, errors.New("error")
		}
		authsig = fmt.Sprintf("%s timestampx=%s;noncestr=%s;Signature=%s",
			"TEST-HMAC-SHA1",
			"wrong", "Wm3WZYTPz0wzccnW", sig,
		)
		req.Header.Add("Authorization", authsig)
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
	t.Run("wrong nonce", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		authsig := fmt.Sprintf("%s timestamp=%s;nonce=%s;Signature=%s",
			"TEST-HMAC-SHA1",
			"1414587457", "Wm3WZYTPz0wzccnW", sig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", authsig)
		w := httptest.NewRecorder()
		mid.nonceExtractor = func(c *gin.Context) ([]string, error) {
			return nil, errors.New("error")
		}
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
}

func TestToken_Wechat_Fail(t *testing.T) {
	cnf := `
signerConfig:
  authLookup: "header:Authorization"
  authScheme: "TEST-HMAC-SHA1"
  authHeaders: ["jsapi_ticket","timestamp"]
  authHeaderDelimiter: ";"
  signedLookups: 
  - jsapi_ticket: header:Authorization>Bearer
  - timestamp:
  - noncestr:
  - url: CanonicalUri
  delimiter: "&"
  nonceKey: "noncestr"
  unsignedPayload: true
interval: 5s
`
	act := "sM4AOVdWfPE4DxkXGEs8VMCPGGVi4C3VM0P37wVUCFvkVAy_90u5h9nbSlYy3-Sl-HhTdfl2fzFy1AOcHKP7qg"
	nocestr := "Wm3WZYTPz0wzccnW"
	rightSig := "0f9de62fce790f9a083d5c99e95740ceb90c27ed"
	rightScheme := "TEST-HMAC-SHA1"
	mid := TokenSigner().(*Middleware)
	mid.config.NowFunc = func() time.Time {
		return time.Unix(1414587457, 0)
	}
	_, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine.RedirectTrailingSlash = false
	engine.Use(mid.ApplyFunc(conf.NewFromBytes([]byte(cnf))))

	engine.POST("/", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	t.Run("miss scheme", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		sig := fmt.Sprintf("%s %s=%s;timestamp=%s;noncestr=%s;Signature=%s",
			"miss", "jsapi_ticket", act,
			"1414587457", nocestr, rightSig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", sig)
		req.Header.Add("noncestr", nocestr)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
	t.Run("miss signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		sig := fmt.Sprintf("%s %s=%s;timestamp=%s;noncestr=%s;Signature=%s",
			rightScheme, "jsapi_ticket", act,
			"1414587457", "Wm3WZYTPz0wzccnW", "wrong-signature",
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", sig)
		req.Header.Add("noncestr", nocestr)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
	t.Run("miss signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		sig := fmt.Sprintf("%s %s=%s;timestamp=%s;noncestr=%s;Signature=%s",
			rightScheme, "jsapi_ticket", act,
			"1414587457", "Wm3WZYTPz0wzccnW", "",
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", sig)
		req.Header.Add("noncestr", nocestr)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("miss timestamp", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		sig := fmt.Sprintf("%s %s=%s;timestamp=;noncestr=%s;Signature=%s",
			rightScheme, "jsapi_ticket", act,
			"Wm3WZYTPz0wzccnW", "",
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", sig)
		req.Header.Add("noncestr", nocestr)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
	t.Run("wrong timestamp", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://mp.weixin.qq.com?params=value", nil)
		sig := fmt.Sprintf("%s %s=%s;timestamp=%s;noncestr=%s;Signature=%s",
			rightScheme, "jsapi_ticket", act,
			"1", "Wm3WZYTPz0wzccnW", rightSig,
		)
		req.Header.Add("Authorization", "Bearer "+act)
		req.Header.Add("Authorization", sig)
		req.Header.Add("noncestr", nocestr)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
}

func TestDefaultSignature_Cache(t *testing.T) {
	p, err := conf.NewParserFromFile(testdata.Path("token/jwt.yaml"))
	require.NoError(t, err)
	tokens := conf.NewFromParse(p)

	act := tokens.String("secretToken")
	mredis := miniredis.RunT(t)
	err = cache.RegisterCache("signature", func() cache.Cache {
		rd, err := redisc.New(conf.NewFromStringMap(map[string]any{
			"type":  "standalone",
			"addrs": []string{mredis.Addr()},
		}))
		require.NoError(t, err)
		return rd
	}())
	require.NoError(t, err)

	cnfstr := `
signerConfig:
  authLookup: "header:Authorization"
  authScheme: "TEST-HMAC-SHA1" 
  authHeaderDelimiter: ";"
  signedLookups:
  - x-timestamp: "header"
  - content-type: "header"
  - content-length: ""
  - x-tenant-id: "header" 
  timestampKey: x-timestamp 
interval: 10s
ttl: 20s
storeKey: signature
`
	cnf := conf.NewFromBytes([]byte(cnfstr))
	sig := "c273dc538230b15894bbc5dade25c88f65ec6df4"
	var tests = []struct {
		name  string
		cnf   *conf.Configuration
		check func(mw *Middleware, sig string)
	}{
		{
			name: "redis with body",
			cnf:  cnf,
			check: func(mw *Middleware, sig string) {
				assert.True(t, mredis.Exists(sig))
				mredis.FastForward(30 * time.Second)
				assert.False(t, mredis.Exists(sig))
			},
		},
		{
			name: "default cache without body",
			cnf: func() *conf.Configuration {
				c := conf.NewFromBytes([]byte(cnfstr))
				c.Parser().Set("storeKey", "")
				return c
			}(),
			check: func(mw *Middleware, sig string) {
				assert.IsType(t, &lfu.TinyLFU{}, mw.cache)
				mw.cache.Has(context.Background(), sig)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mid := Signature().(*Middleware)
			mid.config.NowFunc = func() time.Time {
				return time.Unix(1695298519, 0)
			}
			assert.Equal(t, SignerName, mid.Name())
			_, engine := gin.CreateTestContext(httptest.NewRecorder())
			//engine.RedirectTrailingSlash = false
			engine.Use(mid.ApplyFunc(tt.cnf))

			engine.POST("/", func(c *gin.Context) {
				var hl []string
				assert.NoError(t, c.ShouldBind(&hl))
				c.JSON(http.StatusOK, gin.H{
					"message": "pong",
				})
			})
			body := strings.NewReader(`["hello","world"]`)
			req := httptest.NewRequest("POST", "/", body)
			req.Header.Add("X-Tenant-Id", "123")
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("x-timestamp", "1695298510")
			req.Header.Add("Authorization", "Bearer "+act)
			au := fmt.Sprintf("%s SignedHeaders=%s;Signature=%s", "TEST-HMAC-SHA1",
				"content-length;content-type;x-tenant-id;x-timestamp", sig)
			req.Header.Add("Authorization", au)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			w = httptest.NewRecorder()
			req = req.Clone(context.Background())
			req.Body = io.NopCloser(strings.NewReader(`["hello","world"]`))
			engine.ServeHTTP(w, req)
			assert.Equal(t, 401, w.Code)
			tt.check(mid, sig)
			mid.cache.Del(context.Background(), sig)
		})
	}
}

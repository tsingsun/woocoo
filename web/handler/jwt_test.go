package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
)

func TestJWTMiddleware_ApplyFunc(t *testing.T) {
	p, err := conf.NewParserFromFile(testdata.Path("token/jwt.yaml"))
	require.NoError(t, err)
	tokens := conf.NewFromParse(p)

	mredis := miniredis.RunT(t)
	err = cache.RegisterCache("tokenStore", func() cache.Cache {
		rd, err := redisc.New(conf.NewFromStringMap(map[string]any{
			"type":  "standalone",
			"addrs": []string{mredis.Addr()},
		}))
		require.NoError(t, err)
		return rd
	}())
	require.NoError(t, err)
	type args struct {
		cfg  *conf.Configuration
		opts []handler.MiddlewareOption
	}
	tests := []struct {
		name         string
		args         args
		token        func() string
		errorHandler gin.HandlerFunc
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "default", args: args{cfg: conf.NewFromStringMap(map[string]any{
				"signingKey": "secret",
			})},
			token: func() string {
				return tokens.String("secretToken")
			},
			wantErr: assert.NoError,
		},
		{
			name: "default-opts", args: args{cfg: conf.NewFromStringMap(map[string]any{
				"signingKey": "secret",
			}),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() any {
						nc := &handler.JWTConfig{
							JWTOptions: *auth.NewJWTOptions(),
						}
						nc.JWTOptions.SigningKey = "wrong"
						return nc
					}),
				},
			},
			token: func() string {
				return tokens.String("secretToken")
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				r := i[0].(*httptest.ResponseRecorder)
				return assert.Equal(t, http.StatusOK, r.Code)
			},
		},

		{
			name: "rs256", args: args{cfg: conf.NewFromStringMap(map[string]any{
				"signingMethod": "RS256",
				"signingKey": `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnou03fsVPvv0cYdB61jO
PF0kCP6pawD6Q6DCKvmymP2VGS/RmA1Qf3S8PhLl8AgIwZUNWJeqs9vMiR2wnHiW
2VIUKk4vQ1zsyqhGZ4y1JlDg7yeVzhFoMFen7AfqBnguaNhdzsuNI+HOSyMfSjQz
2p5CG/YI6rPaLEImvTnLPbfsW3XRix0OSLvXZ97FG4gQhnys1pLkwkzy4EQ/L+fc
xt3yh6529bjEJA4uILrdkO/36wBUEDOcfg4j8ldpEkIlLxRnKV/0FrRqrAaetAQJ
3Cv+UWJLwnG59DeVz6wNrOjZ/6urfEW9QVgejPnXD85o9hM89Ys3HexFo/NkVuir
ZwIDAQAB
-----END PUBLIC KEY-----
`,
			})},
			token: func() string {
				return tokens.String("rs256Token")
			},
			wantErr: assert.NoError,
		},
		{
			name: "rs256-file", args: args{cfg: conf.NewFromStringMap(map[string]any{
				"signingMethod": "RS256",
				"signingKey":    "file:///" + testdata.Path(filepath.Join("etc", "jwt_public_key.pem")),
			})},
			token: func() string {
				return tokens.String("rs256Token")
			},
			wantErr: assert.NoError,
		},
		{
			name: "tokenStore Token no exist",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"signingKey":    "secret",
					"tokenStoreKey": "tokenStore",
				}),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() any {
						nc := &handler.JWTConfig{
							JWTOptions: *auth.NewJWTOptions(),
							ErrorHandler: func(c *gin.Context, err error) error {
								c.AbortWithStatusJSON(http.StatusNotAcceptable, 1)
								return errors.New("error")
							},
						}
						return nc
					}),
				},
			},
			token: func() string {
				return tokens.String("secretToken")
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				r := i[0].(*httptest.ResponseRecorder)
				assert.Equal(t, http.StatusNotAcceptable, r.Code)
				assert.Equal(t, "1", r.Body.String())
				return false
			},
		},
		{
			name: "tokenStoreExist", args: args{cfg: conf.NewFromStringMap(map[string]any{
				"signingKey":    "secret",
				"tokenStoreKey": "tokenStore",
			})},
			token: func() string {
				tstr := tokens.String("secretToken")
				token, err := jwt.Parse(tstr, func(token *jwt.Token) (any, error) {
					return []byte("secret"), nil
				})
				require.NoError(t, err)
				require.NoError(t, mredis.Set(token.Claims.(jwt.MapClaims)["jti"].(string), tstr))
				return tstr
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				r := i[0].(*httptest.ResponseRecorder)
				return assert.Equal(t, http.StatusOK, r.Code)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := handler.NewJWT(tt.args.opts...)
			srv := web.New()
			srv.Router().Engine.Use(mw.ApplyFunc(tt.args.cfg))
			srv.Router().Engine.NoRoute(func(c *gin.Context) {
				c.String(http.StatusOK, "")
			})
			var r *http.Request
			var w = httptest.NewRecorder()
			token := tt.token()
			switch tt.args.cfg.String("lookupToken") {
			case "query:token":
				r = httptest.NewRequest("GET", "http://127.0.0.1?token="+token, nil)
			default:
				r = httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "Bearer "+token)
			}
			srv.Router().ServeHTTP(w, r)
			if !tt.wantErr(t, nil, w) {
				return
			}
			if tt.name == "tokenStoreExist" {
				srv.Router().Engine.ContextWithFallback = true
				srv.Router().Engine.POST("/logout", mw.Config.LogoutHandler)
				r = httptest.NewRequest("POST", "/logout", nil)
				r.Header.Set("Authorization", "Bearer "+token)
				w = httptest.NewRecorder()
				srv.Router().ServeHTTP(w, r)
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

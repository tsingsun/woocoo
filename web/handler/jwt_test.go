package handler_test

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestJWTMiddleware_ApplyFunc(t *testing.T) {
	type args struct {
		cfg  *conf.Configuration
		opts []handler.MiddlewareOption
	}
	tests := []struct {
		name    string
		args    args
		token   string
		wantErr bool
	}{
		{
			name: "default", args: args{cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
				"signingKey": "secret",
			}))},
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o",
		},
		{
			name: "default-opts", args: args{cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
				"signingKey": "secret",
			})),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() interface{} {
						nc := &handler.JWTConfig{
							JWTOptions: *auth.NewJWT(),
						}
						nc.JWTOptions.SigningKey = "wrong"
						return nc
					}),
				},
			},
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.XbPfbIHMI6arZ3Y922BhjWgQzWXcXNrz0ogtVhfEd2o",
			wantErr: true,
		},

		{
			name: "rs256", args: args{cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
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
			}))},
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.f5Wq6xqAZCTmtZSnCvp8-1uLMegHwBpAydy779wfZ9FtLxJgBQ1S6QsJuo-W-myLybeBHXBlFRQcc4OWoKGSFomP0oVAdUNPnrm_lm3yRup6mVYJ-QuvT_R0SLtv8ruOAmvu5fcszB06TjrQMHfLQHIikhgqbnMxBARtnlaGlEi4uxmce298QOI4TRtxm7-aR4RlfM2lGcltqrzjhT2sa8TibcdpJ8XPrCm4VBKF6qIX6CuVxZbzX6OT8UJKv_eEWnyL0es7-HIsvj7yHA_l6UgSFV9sXjsOxf-m4PI9iZqJHmOruedOYSgECvo4oiHE0wUc7rU9XQxv0b9HjEW7-Q",
		},
		{
			name: "rs256-file", args: args{cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
				"signingMethod": "RS256",
				"signingKey":    "file:///" + testdata.Path(filepath.Join("etc", "jwt_public_key.pem")),
			}))},
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.f5Wq6xqAZCTmtZSnCvp8-1uLMegHwBpAydy779wfZ9FtLxJgBQ1S6QsJuo-W-myLybeBHXBlFRQcc4OWoKGSFomP0oVAdUNPnrm_lm3yRup6mVYJ-QuvT_R0SLtv8ruOAmvu5fcszB06TjrQMHfLQHIikhgqbnMxBARtnlaGlEi4uxmce298QOI4TRtxm7-aR4RlfM2lGcltqrzjhT2sa8TibcdpJ8XPrCm4VBKF6qIX6CuVxZbzX6OT8UJKv_eEWnyL0es7-HIsvj7yHA_l6UgSFV9sXjsOxf-m4PI9iZqJHmOruedOYSgECvo4oiHE0wUc7rU9XQxv0b9HjEW7-Q",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := handler.JWT(tt.args.opts...)
			srv := web.New()
			srv.Router().Engine.Use(mw.ApplyFunc(tt.args.cfg))
			srv.Router().Engine.NoRoute(func(c *gin.Context) {
				c.String(http.StatusOK, "")
			})
			var r *http.Request
			var w = httptest.NewRecorder()
			switch tt.args.cfg.String("lookupToken") {
			case "query:token":
				r = httptest.NewRequest("GET", "http://localhost?token="+tt.token, nil)
			default:
				r = httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "Bearer "+tt.token)
			}
			srv.Router().ServeHTTP(w, r)

			if w.Code != http.StatusOK && !tt.wantErr {
				t.Errorf("JWTMiddleware.ApplyFunc() error = %v, wantErr %v", w.Code, http.StatusOK)
			}
		})
	}
}

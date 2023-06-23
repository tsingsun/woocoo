package auth

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"path/filepath"
	"testing"
)

var (
	rsakey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnou03fsVPvv0cYdB61jO
PF0kCP6pawD6Q6DCKvmymP2VGS/RmA1Qf3S8PhLl8AgIwZUNWJeqs9vMiR2wnHiW
2VIUKk4vQ1zsyqhGZ4y1JlDg7yeVzhFoMFen7AfqBnguaNhdzsuNI+HOSyMfSjQz
2p5CG/YI6rPaLEImvTnLPbfsW3XRix0OSLvXZ97FG4gQhnys1pLkwkzy4EQ/L+fc
xt3yh6529bjEJA4uILrdkO/36wBUEDOcfg4j8ldpEkIlLxRnKV/0FrRqrAaetAQJ
3Cv+UWJLwnG59DeVz6wNrOjZ/6urfEW9QVgejPnXD85o9hM89Ys3HexFo/NkVuir
ZwIDAQAB
-----END PUBLIC KEY-----
`
)

func TestJwtOptions(t *testing.T) {
	p, err := conf.NewParserFromFile(testdata.Path("token/jwt.yaml"))
	require.NoError(t, err)
	tokens := conf.NewFromParse(p)

	t.Run("default", func(t *testing.T) {
		opt := NewJWTOptions()
		assert.Error(t, opt.Init())
		opt.SigningKey = "secret"
		assert.NoError(t, opt.Init())
		assert.Equal(t, opt.SigningMethod, AlgorithmHS256)
		assert.Equal(t, opt.TokenLookup, "header:Authorization")
		assert.Equal(t, opt.AuthScheme, "Bearer")
		assert.Equal(t, opt.Claims, jwt.MapClaims{})
		assert.NotEqual(t, opt.SigningKey, defaultJWTOptions.SigningKey, "not change to default options")

		_, err := opt.ParseTokenFunc(nil, "")
		assert.Error(t, err)
		tk, err := opt.ParseTokenFunc(nil, tokens.String("secretToken"))
		assert.NoError(t, err)
		id, ok := opt.GetTokenIDFunc(tk)
		assert.True(t, ok)
		assert.Equal(t, id, "67a87482e91f4f2e9220f51376185b7e")
	})
	t.Run("keys", func(t *testing.T) {
		opt := NewJWTOptions()
		opt.SigningKeys = map[string]any{
			"secret": "secret",
		}
		assert.NoError(t, opt.Init())
		_, err := opt.ParseTokenFunc(nil, tokens.String("secretKidToken"))
		assert.Error(t, err)

		opt.SigningKeys = map[string]any{
			"nokid": []byte("secret"),
		}
		_, err = opt.ParseTokenFunc(nil, tokens.String("secretKidToken"))
		assert.Error(t, err)

		opt.SigningKeys = map[string]any{
			"secret": []byte("secret"),
		}
		_, err = opt.ParseTokenFunc(nil, tokens.String("secretKidToken"))
		assert.NoError(t, err)
	})
	t.Run("rsa", func(t *testing.T) {
		opt := NewJWTOptions()
		opt.SigningMethod = "RS256"
		opt.SigningKey = rsakey
		require.NoError(t, opt.Init())
		tk, err := opt.ParseTokenFunc(nil, tokens.String("rs256Token"))
		assert.NoError(t, err)
		assert.Equal(t, tk.Claims.(jwt.MapClaims)["sub"], "1234567890")
	})
	t.Run("rs256-file", func(t *testing.T) {
		opt := NewJWTOptions()
		opt.SigningMethod = "RS256"
		opt.SigningKey = "file:///" + testdata.Path(filepath.Join("etc", "jwt_public_key.pem"))
		require.NoError(t, opt.Init())
		tk, err := opt.ParseTokenFunc(nil, tokens.String("rs256Token"))
		assert.NoError(t, err)
		assert.Equal(t, tk.Claims.(jwt.MapClaims)["sub"], "1234567890")
	})
	t.Run("rs256-wrong-method", func(t *testing.T) {
		opt := NewJWTOptions()
		opt.SigningMethod = "ES256"
		opt.SigningKey = "file:///" + testdata.Path(filepath.Join("etc", "jwt_public_key.pem"))
		require.NoError(t, opt.Init())
		_, err := opt.ParseTokenFunc(nil, tokens.String("rs256Token"))
		assert.Error(t, err)
	})
	t.Run("ParseSigningKeyFromString", func(t *testing.T) {
		key, err := ParseSigningKeyFromString("file:///"+testdata.Path(filepath.Join("etc", "jwt_private.pem")),
			"RS256", true)
		assert.NoError(t, err)
		assert.NotNil(t, key)

		_, err = ParseSigningKeyFromString("", "XXXX", false)
		assert.ErrorContains(t, err, "requires signing method")

		_, err = ParseSigningKeyFromString("file:////", "RS256", false)
		assert.Error(t, err)

	})
}

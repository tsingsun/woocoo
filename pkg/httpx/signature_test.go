package httpx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func buildAwsReq(serviceName, region string, body string) (*http.Request, io.ReadSeeker) {
	reader := strings.NewReader(body)
	endpoint := "https://" + serviceName + "." + region + ".amazonaws.com"
	req, _ := http.NewRequest("POST", endpoint, reader)
	req.URL.Opaque = "//example.org/bucket/key-._~,!@#$%^&*()"
	req.Header.Set("X-Amz-Target", "prefix.Operation")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	if len(body) > 0 {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}

	req.Header.Set("X-Amz-Meta-Other-Header", "some-value=!@#$%^&* (+)")
	req.Header.Add("X-Amz-Meta-Other-Header_With_Underscore", "some-value=!@#$%^&* (+)")
	req.Header.Add("X-amz-Meta-Other-Header_With_Underscore", "some-value=!@#$%^&* (+)")

	return req, reader
}

type awsSigner struct {
	*DefaultSigner
}

func (s *awsSigner) AttachData(ctx *SigningCtx) error {
	ctx.CredentialString = "20230918/us-east-1/dynamodb/aws4_request"
	ctx.SignedVals["Credential"] = s.GetAccessKeyID() + "/" + ctx.CredentialString
	return nil
}

func (s *awsSigner) CalculateSignature(ctx *SigningCtx) error {
	if err := s.StringToSign(ctx); err != nil {
		return err
	}
	creds := deriveSigningKey("us-east-1", "dynamodb", s.SignerConfig.GetAccessKeySecret(), ctx.SignTime)
	signature := hmacSHA256(creds, []byte(ctx.StringToSign))
	ctx.Signature = hex.EncodeToString(signature)
	return nil
}

func deriveSigningKey(region, service, secretKey string, dt time.Time) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(formatShortTime(dt)))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	signingKey := hmacSHA256(kService, []byte("aws4_request"))
	return signingKey
}

func formatShortTime(dt time.Time) string {
	return dt.UTC().Format("20060102")
}

func hmacSHA256(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func TestSignerConfig(t *testing.T) {
	t.Run("BuildSigner", func(t *testing.T) {
		cfg := DefaultSignerConfig
		cfg.SignedLookups = map[string]string{
			"content-length": "",
		}
		cfg.AuthLookup = "header:Authorization"
		cfg.AuthScheme = "TEST-HMAC-SHA1"
		signer, err := cfg.BuildSigner(WithSigner(NewDefaultSigner))
		require.NoError(t, err)
		assert.Equal(t, "TEST-HMAC-SHA1 ", signer.config.AuthScheme, "will add space")
	})
	t.Run("BuildSignerErr", func(t *testing.T) {
		cfg := DefaultSignerConfig
		cfg.SignedLookups = map[string]string{
			"content-length": "",
		}
		cfg.AuthLookup = ""
		_, err := cfg.BuildSigner(WithSigner(NewDefaultSigner))
		require.ErrorContains(t, err, "authLookup must not empty")
	})
}

func TestSignRequest_aws(t *testing.T) {
	cnf := conf.NewFromStringMap(map[string]any{
		"signedLookups": map[string]string{
			"x-amz-date":              "",
			"content-length":          "",
			"content-type":            "",
			"host":                    "header",
			"x-amz-meta-other-header": "",
			"x-amz-meta-other-header_with_underscore": "",
			"x-amz-security-token":                    "",
			"x-amz-target":                            "",
		},
		"authHeaders":         []string{"Credential"},
		"authHeaderDelimiter": ", ",
		"timestampKey":        "x-amz-date",
		"algorithm":           "sha256",
		"authScheme":          "AWS4-HMAC-SHA256",
		"authLookup":          "header:Authorization",
		"dateFormat":          "20060102T150405Z",
		"credentials": map[string]string{
			"id":      "AKID",
			"secret":  "SECRET",
			"SESSION": "SESSION",
		},
	})
	cfg, err := NewSignerConfig(WithConfiguration(cnf))
	require.NoError(t, err)
	ds, err := NewDefaultSigner(cfg)
	require.NoError(t, err)
	signer := Signature{
		config: cfg,
		singer: &awsSigner{
			DefaultSigner: ds.(*DefaultSigner),
		},
	}
	assert.Equal(t, signer.config.Algorithm.name, AlgorithmSha256.name)

	req, _ := buildAwsReq("dynamodb", "us-east-1", "{}")
	req.Header.Add("X-Amz-Security-Token", "SESSION")
	st, err := time.Parse(signer.config.DateFormat, "20230918T122143Z")
	require.NoError(t, err)
	err = signer.Sign(req, "", st)
	require.NoError(t, err)

	expectedSig := "AWS4-HMAC-SHA256 Credential=AKID/20230918/us-east-1/dynamodb/aws4_request, SignedHeaders=content-length;content-type;host;x-amz-date;x-amz-meta-other-header;x-amz-meta-other-header_with_underscore;x-amz-security-token;x-amz-target, Signature=7cbac190fcb780f06f5b68345cebaeb30936bc16d79db454ea7a3111162fe497"
	q := req.Header
	assert.Equal(t, expectedSig, q.Get("Authorization"))

	req = req.Clone(context.Background())
	req.Body = io.NopCloser(strings.NewReader("{}"))
	err = signer.Verify(req, "", st)
	assert.NoError(t, err)
}

func TestSignSimple(t *testing.T) {
	cnf := conf.NewFromStringMap(map[string]any{
		"signedLookups": map[string]any{
			"content-length": "",
			"content-type":   "",
			"x-tenant-id":    "",
		},
		"algorithm":  "sha1",
		"authScheme": "TEST-HMAC-SHA1",
		"authLookup": "header:Authorization",
	})

	signer, err := NewSignature(WithConfiguration(cnf))
	require.NoError(t, err)
	st, err := ParseSignTime("20060102T150405Z", "20230918T122143Z")
	require.NoError(t, err)
	t.Run("header not all", func(t *testing.T) {
		body := strings.NewReader("hello")
		req := httptest.NewRequest("POST", "http://example.com", body)
		req.Header.Add("X-Tenant-Id", "123")
		req.Header.Add("Authorization", "Bearer abc")

		assert.NoError(t, signer.Sign(req, "", st))
		sh := req.Header.Get("Authorization")
		assert.NotEmpty(t, sh)
		assert.Len(t, req.Header.Values("Authorization"), 2)
	})
	t.Run("body nil", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		assert.NoError(t, signer.Sign(req, "", st))
		sh := req.Header.Get("Authorization")
		assert.NotEmpty(t, sh)
	})
}

func TestTokenSigner_wechat(t *testing.T) {
	cnf := conf.NewFromStringMap(map[string]any{
		"signedLookups": map[string]string{
			"timestamp":    "",
			"noncestr":     "",
			"jsapi_ticket": "context",
			"url":          "CanonicalUri",
		},
		"timestampKey": "timestamp",
		"nonceKey":     "noncestr",
		"algorithm":    "sha1",
		"authScheme":   "TEST-HMAC-SHA1",
		"authLookup":   "header:Authorization",
		"authHeaders":  []string{"timestamp", "noncestr", "jsapi_ticket"},
		"delimiter":    "&",
		"dateFormat":   "",
		"nonceLen":     12,
		"data": map[string]any{
			"tokenEle": "jsapi_ticket",
			"nonceEle": "noncestr",
		},
	})

	signer, err := NewSignature(WithConfiguration(cnf), WithSigner(NewTokenSigner))

	url := "http://mp.weixin.qq.com?params=value#abc"
	dv := map[string]any{
		"AppID": "Wm3WZYTPz0wzccnW",
	}

	jsapi_ticket := "sM4AOVdWfPE4DxkXGEs8VMCPGGVi4C3VM0P37wVUCFvkVAy_90u5h9nbSlYy3-Sl-HhTdfl2fzFy1AOcHKP7qg"
	payload, err := json.Marshal(dv)
	require.NoError(t, err)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	req = req.WithContext(context.WithValue(context.Background(), "jsapi_ticket", jsapi_ticket))
	assert.NoError(t, err)

	ts, err := ParseSignTime("", "1414587457")
	require.NoError(t, err)
	t.Run("static nonce", func(t *testing.T) {
		r := req.Clone(req.Context())
		err = signer.Sign(r, "Wm3WZYTPz0wzccnW", ts)
		assert.NoError(t, err)
		want := "0f9de62fce790f9a083d5c99e95740ceb90c27ed"
		got, err := GetSignedRequestSignature(r, "Authorization", signer.config.AuthScheme, signer.config.AuthHeaderDelimiter)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("autogen nonce", func(t *testing.T) {
		r := req.Clone(req.Context())
		err = signer.Sign(r, "", ts)
		assert.NoError(t, err)
		got, err := GetSignedRequestSignature(r, "Authorization", signer.config.AuthScheme, signer.config.AuthHeaderDelimiter)
		require.NoError(t, err)
		assert.NotEmpty(t, got)
		hs := ValuesFromCanonical(r.Header.Get("Authorization"), signer.config.AuthHeaderDelimiter, "=")
		assert.Len(t, hs[signer.config.NonceKey], 12)
	})
}

func TestJWTToken(t *testing.T) {
	p, err := conf.NewParserFromFile(testdata.Path("token/jwt.yaml"))
	require.NoError(t, err)
	tokens := conf.NewFromParse(p)
	act := tokens.String("secretToken")
	cnf := conf.NewFromStringMap(map[string]any{
		"signedLookups": map[string]any{
			"accessToken": "header:authorization>bearer",
			"clientToken": "context:client_token",
			"timestamp":   "",
			"nonce":       "",
			"url":         "CanonicalUri",
		},
		"algorithm":  "sha1",
		"authScheme": "TEST-HMAC-SHA1",
		"authLookup": "header:Authorization",
		"delimiter":  "&",
		"nonceLen":   12,
	})
	signer, err := NewSignature(WithConfiguration(cnf), WithSigner(NewTokenSigner))
	require.NoError(t, err)
	url := "http://127.0.0.1/"
	body := strings.NewReader("hello world")
	req, err := http.NewRequest("POST", url, body)
	require.NoError(t, err)
	req.Header.Add("Authorization", "Bearer "+act)
	req = req.WithContext(context.WithValue(req.Context(), "client_token", "client_token_value"))
	ts := time.Unix(1414587457, 0)
	err = signer.Sign(req, "Wm3WZYTPz0wzccnW", ts)
	require.NoError(t, err)
	want := "2e51b8443d426814b2faa0fe80376f9a1054443b"
	got, err := GetSignedRequestSignature(req, "Authorization", signer.config.AuthScheme, signer.config.AuthHeaderDelimiter)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAlgorithm(t *testing.T) {
	t.Run("not support", func(t *testing.T) {
		cnf := conf.NewFromStringMap(map[string]any{
			"algorithm": "sha2",
		})
		_, err := NewSignerConfig(WithConfiguration(cnf))
		assert.Error(t, err)
	})
}

func TestDefaultSignature_WithBody(t *testing.T) {
	p, err := conf.NewParserFromFile(testdata.Path("token/jwt.yaml"))
	require.NoError(t, err)
	tokens := conf.NewFromParse(p)
	act := tokens.String("secretToken")

	cnfstr := `
authLookup: "header:Authorization"
authScheme: "TEST-HMAC-SHA1" 
authHeaderDelimiter: ";"
signedLookups:
  - x-timestamp: "header"
  - content-type: "header"
  - content-length: ""
  - x-tenant-id: "header" 
timestampKey: x-timestamp
`
	cnf := conf.NewFromBytes([]byte(cnfstr))
	signer, err := NewSignature(WithConfiguration(cnf))
	require.NoError(t, err)
	t.Run("json", func(t *testing.T) {
		body := strings.NewReader(`["hello","world"]`)
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Add("X-Tenant-Id", "123")
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+act)
		err = signer.Sign(req, "", time.Unix(1695298510, 0))
		assert.NoError(t, err)
	})
	t.Run("body nil SHA1", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+act)
		err = signer.Sign(req, "", time.Unix(1695298510, 0))
		assert.NoError(t, err)
	})
	t.Run("body nil SHA256", func(t *testing.T) {
		c1 := `
algorithm: SHA256
`
		require.NoError(t, cnf.Merge([]byte(c1)))
		signer, err := NewSignature(WithConfiguration(cnf))
		require.NoError(t, err)
		req, err := http.NewRequest("GET", "/", nil)
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+act)
		err = signer.Sign(req, "", time.Unix(1695298510, 0))
		assert.NoError(t, err)
	})
}

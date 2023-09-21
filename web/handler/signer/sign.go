package signer

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/httpx"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"time"
)

const (
	TokenSignerName = "tokenSign"
	SignerName      = "sign"
)

// Config is the configuration for the Replay Attack Protect middleware.
type Config struct {
	// Skipper defines a function to skip middleware.
	Skipper handler.Skipper `json:"-" yaml:"-"`
	// Exclude is a list of http paths to exclude from RAP
	Exclude      []string           `json:"exclude" yaml:"exclude"`
	SignerConfig httpx.SignerConfig `json:"signerConfig" yaml:"signerConfig"`
	// Interval is the interval time for request timestamp.
	Interval time.Duration `json:"interval" yaml:"interval"`
	// StoreKey is the name of the cache driver which is used to store nonce.
	// default is "redis".
	StoreKey string `json:"storeKey" yaml:"storeKey"`
	// TTL is the ttl of signature. Must be greater than Interval.
	//
	// If you Use TokenSigner should be greater than the token ttl, in token ttl, the signature is cached,
	// so that the same request will be rejected.
	// Default is 2 hours.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// NowFunc create a time.Time object for current time, useful in tests.
	NowFunc func() time.Time `json:"-" yaml:"-"`
}

// Middleware implements a Replay Attack Protect middleware.
type Middleware struct {
	name   string
	config *Config
	cache  cache.Cache
	Signer *httpx.Signature

	nonceExtractor     handler.ValuesExtractor
	timestampExtractor handler.ValuesExtractor
	signatureExtractor handler.ValuesExtractor
}

// NewMiddleware constructs a new Middleware struct with supplied options.
func NewMiddleware() *Middleware {
	mw := &Middleware{
		name: SignerName,
		config: &Config{
			SignerConfig: httpx.DefaultSignerConfig,
			Interval:     5 * time.Minute,
			NowFunc:      time.Now,
			TTL:          24 * time.Hour,
		},
	}
	mw.config.SignerConfig.Dry = true
	return mw
}

// Signature is the replay attack protect middleware apply function. See MiddlewareNewFunc
func Signature() handler.Middleware {
	return NewMiddleware()
}

func TokenSigner() handler.Middleware {
	mw := NewMiddleware()
	mw.name = TokenSignerName
	return mw
}

func (mw *Middleware) Name() string {
	return mw.name
}

func (mw *Middleware) build(cfg *conf.Configuration) (err error) {
	if err := cfg.Unmarshal(&mw.config); err != nil {
		panic(err)
	}
	if mw.config.Skipper == nil {
		mw.config.Skipper = handler.PathSkipper(mw.config.Exclude)
	}
	switch mw.name {
	case TokenSignerName:
		mw.Signer, err = mw.config.SignerConfig.BuildSigner(httpx.WithSigner(httpx.NewTokenSigner))
	default:
		mw.Signer, err = mw.config.SignerConfig.BuildSigner(httpx.WithSigner(httpx.NewDefaultSigner))
	}
	for key, loc := range mw.config.SignerConfig.SignedLookups {
		switch key {
		case mw.config.SignerConfig.TimestampKey:
			fs, _ := handler.CreateExtractors(loc+":"+key, "")
			if len(fs) > 0 {
				mw.timestampExtractor = fs[0]
			}
		case mw.config.SignerConfig.NonceKey:
			fs, _ := handler.CreateExtractors(loc+":"+key, "")
			if len(fs) > 0 {
				mw.nonceExtractor = fs[0]
			}
		}
	}
	for _, header := range mw.config.SignerConfig.AuthHeaders {
		switch header {
		case mw.config.SignerConfig.TimestampKey:
			mw.timestampExtractor = nil
		case mw.config.SignerConfig.NonceKey:
			mw.nonceExtractor = nil
		}
	}
	//
	fs, err := handler.CreateExtractors(mw.config.SignerConfig.AuthLookup, mw.config.SignerConfig.AuthScheme)
	if err != nil {
		return err
	}
	mw.signatureExtractor = fs[0]
	if mw.config.StoreKey != "" {
		mw.cache = cache.GetCache(mw.config.StoreKey)
	}
	return err
}

// ApplyFunc applies the middleware to the gin engine.
func (mw *Middleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := mw.build(cfg); err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		if mw.config.Skipper(c) {
			return
		}

		sigStrs, err := mw.signatureExtractor(c)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		sigVal := httpx.ValuesFromCanonical(sigStrs[0], mw.config.SignerConfig.AuthHeaderDelimiter, "=")
		signature, _ := sigVal[httpx.SignatureName]
		if signature == "" {
			c.AbortWithError(http.StatusBadRequest, httpx.ErrInvalidSignature)
			return
		}
		nonceStr, signtime, err := mw.extractorVerifyParam(c, sigVal)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		err = mw.Signer.Verify(c.Request, nonceStr, signtime)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if err = mw.SignatureValidate(c, signature); err != nil {
			c.Error(err)
			return
		}
	}
}

func (mw *Middleware) extractorVerifyParam(c *gin.Context, sigv map[string]string) (nonce string, sign time.Time, err error) {
	var (
		vs []string
		st string
	)
	if mw.nonceExtractor != nil {
		vs, err = mw.nonceExtractor(c)
		if err != nil {
			return
		}
		nonce = vs[0]
	} else {
		nonce, _ = sigv[mw.config.SignerConfig.NonceKey]
	}
	if mw.timestampExtractor != nil {
		vs, err = mw.timestampExtractor(c)
		if err != nil {
			return
		}
		st = vs[0]
	} else {
		st, _ = sigv[mw.config.SignerConfig.TimestampKey]
	}
	sign, err = httpx.ParseSignTime(st, mw.config.SignerConfig.DateFormat)
	if err != nil {
		return
	}
	if mw.config.NowFunc().Sub(sign) > mw.config.Interval {
		err = errors.New("timestamp is expired")
		return
	}
	return
}

func (mw *Middleware) SignatureValidate(c *gin.Context, signature string) (err error) {
	if mw.cache != nil {
		if exists := mw.cache.Has(c, signature); exists {
			c.AbortWithStatus(http.StatusBadRequest)
			return errors.New("signature is expired")
		}
		err = mw.cache.Set(c, signature, "1", cache.WithTTL(mw.config.TTL))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return err
		}
	}
	return
}

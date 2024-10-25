package signer

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/cache/lfu"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/httpx"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"time"
)

const (
	TokenSignerName = "tokenSign"
	HMACSignerName  = "hmacSign"
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
	// When default cache effect, this value is used by setting cache ttl.
	Interval time.Duration `json:"interval" yaml:"interval"`
	// StoreKey is the name of the cache driver which is used to store nonce.
	// default is "redis".
	StoreKey string `json:"storeKey" yaml:"storeKey"`
	// TTL is the ttl of signature. Must be greater than Interval.
	//
	// If you Use TokenSigner should be greater than the token ttl, in token ttl, the signature is cached,
	// so that the same request will be rejected.
	// Default is 24 hours. But when you use default cache, This value is not used.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// NowFunc create a time.Time object for current time, useful in tests.
	NowFunc func() time.Time `json:"-" yaml:"-"`
}

// Middleware verifies signed http request, use it for replay attack protection and data tampering prevention.
//
// If you don't set the cache, the middleware will use a default cache.
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
// name is the name of the middleware, default is "sign", name can be empty string meaning default.
func NewMiddleware(name string, opts ...handler.MiddlewareOption) *Middleware {
	mipts := handler.NewMiddlewareOption(opts...)
	if name == "" {
		name = SignerName
	}
	mw := &Middleware{
		name: name,
		config: &Config{
			SignerConfig: httpx.DefaultSignerConfig,
			Interval:     5 * time.Minute,
			NowFunc:      time.Now,
			TTL:          24 * time.Hour,
		},
	}
	if mipts.ConfigFunc != nil {
		mipts.ConfigFunc(mw.config)
	}
	// force set dry mode
	mw.config.SignerConfig.Dry = true
	return mw
}

// Signature is the replay attack protect middleware apply function. See MiddlewareNewFunc
func Signature() handler.Middleware {
	return NewMiddleware(SignerName)
}

func TokenSigner() handler.Middleware {
	mw := NewMiddleware(TokenSignerName)
	return mw
}

func (mw *Middleware) Name() string {
	return mw.name
}

func (mw *Middleware) build(cnf *conf.Configuration) (err error) {
	if err = cnf.Unmarshal(&mw.config); err != nil {
		return err
	}
	if mw.config.Skipper == nil {
		mw.config.Skipper = handler.PathSkipper(mw.config.Exclude)
	}
	switch mw.name {
	case TokenSignerName:
		mw.Signer, err = mw.config.SignerConfig.BuildSigner(httpx.WithSigner(httpx.NewTokenSigner))
	case HMACSignerName:
		mw.Signer, err = mw.config.SignerConfig.BuildSigner(httpx.WithSigner(httpx.NewHMACSigner))
	case SignerName:
	default:
		mw.Signer, err = mw.config.SignerConfig.BuildSigner(httpx.WithSigner(httpx.NewDefaultSigner))
	}
	if err != nil {
		return err
	}
	for key, loc := range mw.config.SignerConfig.SignedLookups {
		switch key {
		case mw.config.SignerConfig.TimestampKey:
			fs, _ := handler.CreateExtractors(loc, "")
			if len(fs) > 0 {
				mw.timestampExtractor = fs[0]
			}
		case mw.config.SignerConfig.NonceKey:
			fs, _ := handler.CreateExtractors(loc, "")
			if len(fs) > 0 {
				mw.nonceExtractor = fs[0]
			}
		}
	}
	// clear extractors if timestamp and nonce in auth headers, they load from auth headers
	for _, header := range mw.config.SignerConfig.AuthHeaders {
		switch header {
		case mw.config.SignerConfig.TimestampKey:
			mw.timestampExtractor = nil
		case mw.config.SignerConfig.NonceKey:
			mw.nonceExtractor = nil
		}
	}
	fs, err := handler.CreateExtractors(mw.config.SignerConfig.AuthLookup, mw.config.SignerConfig.AuthScheme)
	if err != nil {
		return err
	}
	if len(fs) == 0 {
		return errors.New("no signature extractor found")
	}
	mw.signatureExtractor = fs[0]
	if mw.config.StoreKey != "" {
		if mw.cache, err = cache.GetCache(mw.config.StoreKey); err != nil {
			return err
		}
	} else {
		mw.cache, err = lfu.NewTinyLFU(conf.NewFromStringMap(map[string]any{
			"size": 100000,
			"ttl":  mw.config.Interval + time.Minute,
		}))
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
			c.AbortWithError(http.StatusUnauthorized, err) //nolint:errcheck
			return
		}
		sigVal := httpx.ValuesFromCanonical(sigStrs[0], mw.config.SignerConfig.AuthHeaderDelimiter, "=")
		signature := sigVal[httpx.SignatureName]
		if signature == "" {
			c.AbortWithError(http.StatusUnauthorized, httpx.ErrInvalidSignature) //nolint:errcheck
			return
		}
		nonceStr, signtime, err := mw.extractorVerifyParam(c, sigVal)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err) //nolint:errcheck
			return
		}

		err = mw.Signer.Verify(c.Request, nonceStr, signtime)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err) //nolint:errcheck
			return
		}
		if err = mw.signatureValidate(c, signature); err != nil {
			c.Error(err) // nolint: errcheck
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
		nonce = sigv[mw.config.SignerConfig.NonceKey]
	}
	if mw.timestampExtractor != nil {
		vs, err = mw.timestampExtractor(c)
		if err != nil {
			return
		}
		st = vs[0]
	} else {
		st = sigv[mw.config.SignerConfig.TimestampKey]
	}
	sign, err = httpx.ParseSignTime(mw.config.SignerConfig.DateFormat, st)
	if err != nil {
		return
	}
	if dif := mw.config.NowFunc().Sub(sign); dif < 0 || dif > mw.config.Interval {
		err = errors.New("timestamp is expired")
		return
	}
	return
}

func (mw *Middleware) signatureValidate(c *gin.Context, signature string) (err error) {
	err = mw.cache.Set(c, signature, nil, cache.WithTTL(mw.config.TTL), cache.WithSetNX())
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return err
	}
	return
}

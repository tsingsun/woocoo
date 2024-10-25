package httpx

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/gds"
	"hash"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

var (
	AlgorithmSha256 = &Algorithm{"sha256", sha256.New}
	AlgorithmSha1   = &Algorithm{"sha1", sha1.New}

	ErrUnknownAlgorithm = errors.New("unknown algorithm")
	ErrInvalidSignature = errors.New("invalid signature")
	noEscape            [256]bool
)

const (
	HeaderXHost = "host"
	// emptyStringSHA256 is a SHA256 of an empty string
	emptyStringSHA256 = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` //nolint:gosec
	emptyStringSHA1   = `d41d8cd98f00b204e9800998ecf8427e`
	signatureElem     = "Signature="
	signedHeadersElem = "SignedHeaders="
	NonceName         = "nonce"
	TimestampName     = "timestamp"
	SignatureName     = "Signature"
)

func init() {
	for i := 0; i < len(noEscape); i++ {
		// these to be escaped
		noEscape[i] = (i >= 'A' && i <= 'Z') ||
			(i >= 'a' && i <= 'z') ||
			(i >= '0' && i <= '9') ||
			i == '-' ||
			i == '.' ||
			i == '_' ||
			i == '~'
	}
}

// SigningCtx holds info for signature
type SigningCtx struct {
	Request              *http.Request
	Nonce                string
	BodyDigest           string
	SignedHeaders        string
	CanonicalUri         string //nolint:stylecheck
	CanonicalQueryString string
	SignTime             time.Time
	Signature            string
	CredentialString     string
	StringToSign         string
	// CanonicalHeaders is built by sorted scope headers.
	CanonicalHeaders []string
	SignedVals       map[string]string
}

// Signer is the interface for signature, it supports client signer request or server validate request.
// Note that: only change the Request in AttachRequest, the server side not call this method.
type Signer interface {
	// BuildCanonicalRequest build and prepare data by canonical the request to use in sign action.
	BuildCanonicalRequest(r *http.Request, ctx *SigningCtx) error
	// AttachData attach data that need to sign.
	AttachData(ctx *SigningCtx) error
	// CalculateSignature calculate signature by ctx.
	CalculateSignature(ctx *SigningCtx) error
	// AttachRequest attach the signature to http request suck as set header, add the signature to request.
	AttachRequest(r *http.Request, ctx *SigningCtx)
}

type SignerOption func(*SignerConfig)

// SignerConfig is hold setting for Signer.
type SignerConfig struct {
	// Credentials default id="" secret=""
	Credentials map[string]string `yaml:"credentials" json:"credentials"`
	// SignedLookups indicate how to find data for signer, will be ordered.
	// e.g. "content-type":"header" : key `content-type` will be located in `header`.
	// support location: header(or location is empty), query, context.
	SignedLookups map[string]string `yaml:"signedLookups" json:"signedLookups"`
	// SignatureLookup indicate where to find the whole Signature info. Default: header:Authorization
	AuthLookup string `yaml:"authLookup" json:"authLookup"`
	// AuthScheme indicate the scheme in authLookup
	AuthScheme string `yaml:"authScheme" json:"authScheme"`
	// AuthHeaders indicate the headers appended to auth header.
	AuthHeaders []string `yaml:"authHeaders" json:"authHeaders"`
	// AuthHeaderDelimiter is the delimiter used to separate fields in the header string.
	// Default value ", "
	AuthHeaderDelimiter string `yaml:"authHeaderDelimiter" json:"authHeaderDelimiter"`
	// TimestampKey is the name of timestamp in SignedLookups.
	TimestampKey string `yaml:"timestampKey" json:"timestampKey"`
	// NonceKey is the name of nonce.
	NonceKey   string    `yaml:"nonceKey" json:"nonceKey"`
	Algorithm  Algorithm `yaml:"algorithm" json:"algorithm"`
	DateFormat string    `yaml:"dateFormat" json:"dateFormat"`
	NonceLen   uint8     `yaml:"nonceLen" json:"nonceLen"`
	// Delimiter is the delimiter used to separate fields in the signature string.
	// Default value "\n"
	Delimiter string `yaml:"delimiter" json:"delimiter"`
	// UnsignedPayload calls BuildBodyDigest if false, default false.
	UnsignedPayload bool `yaml:"unsignedPayload" json:"unsignedPayload"`
	// default false
	DisableURIPathEscaping bool `yaml:"disableURIPathEscaping" json:"disableURIPathEscaping"`
	// just calculate string to sign, not attach to request
	Dry bool `yaml:"-" json:"-"`
	// ScopeHeaders is a list of http headers to be included in signature, parsed from SignedLookups.
	// ScopeHeaders must confirm sort func.
	ScopeHeaders []string `yaml:"-" json:"-"`
	// SignedQueries is a list of http queries to be included in signature.
	ScopeQueries []string `yaml:"-" json:"-"`
	// SignatureQueryKey parse from AuthLookup
	SignatureQueryKey string `yaml:"-" json:"-"`
	// SignatureHeaderKey parse from AuthLookup
	SignatureHeaderKey string `yaml:"-" json:"-"`

	signer  Signer
	initErr error
}

// WithSigner set signer initial func to config.
func WithSigner(newSigner func(config *SignerConfig) (Signer, error)) SignerOption {
	return func(config *SignerConfig) {
		sg, err := newSigner(config)
		if err != nil {
			config.initErr = err
		}
		config.signer = sg
	}
}

// WithConfiguration set configuration to config.
func WithConfiguration(cnf *conf.Configuration) SignerOption {
	return func(config *SignerConfig) {
		err := cnf.Unmarshal(config)
		if err != nil {
			config.initErr = err
		}
	}
}

var DefaultSignerConfig = SignerConfig{
	AuthLookup:          "header:Authorization",
	Algorithm:           *AlgorithmSha1,
	AuthHeaderDelimiter: ", ",
	Delimiter:           "\n",
	DateFormat:          "", // use timestamp
	TimestampKey:        TimestampName,
	NonceKey:            NonceName,
	NonceLen:            10,
}

// NewSignerConfig create signer config by configuration and options.
func NewSignerConfig(opts ...SignerOption) (*SignerConfig, error) {
	s := DefaultSignerConfig
	s.Credentials = map[string]string{
		"id":     "",
		"secret": "",
	}

	for _, opt := range opts {
		opt(&s)
	}
	if s.initErr != nil {
		return nil, s.initErr
	}
	s.initData()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *SignerConfig) initData() {
	s.extractSignedLookups()
	s.extractAuthLookup()
}

func (s *SignerConfig) Validate() error {
	if s.SignatureQueryKey == "" && s.SignatureHeaderKey == "" {
		return errors.New("http signature need a key for adding to query or header, but both not found")
	}
	return nil
}

func (s *SignerConfig) extractSignedLookups() {
	for key, loc := range s.SignedLookups {
		switch loc {
		case "":
			fallthrough
		case "header":
			s.SignedLookups[key] = "header:" + key
			s.ScopeHeaders = append(s.ScopeHeaders, key)
		case "query":
			s.SignedLookups[key] = "query:" + key
			s.ScopeQueries = append(s.ScopeQueries, key)
		case "context":
			s.SignedLookups[key] = "context:" + key
		}
	}
	sort.Strings(s.ScopeHeaders)
	sort.Strings(s.ScopeQueries)
}

func (s *SignerConfig) extractAuthLookup() {
	sources := strings.Split(s.AuthLookup, ",")
	for _, source := range sources {
		parts := strings.Split(source, ":")
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "header":
			s.SignatureHeaderKey = strings.TrimSpace(parts[1])
		case "query":
			s.SignatureQueryKey = strings.TrimSpace(parts[1])
		}
	}
	if s.AuthScheme != "" && !strings.HasSuffix(s.AuthScheme, " ") {
		s.AuthScheme += " "
	}
}

func (s *SignerConfig) GetAccessKeyID() string {
	return s.Credentials["id"]
}

func (s *SignerConfig) GetAccessKeySecret() string {
	return s.Credentials["secret"]
}

func (s *SignerConfig) BuildSigner(opts ...SignerOption) (*Signature, error) {
	for _, opt := range opts {
		opt(s)
	}
	if s.initErr != nil {
		return nil, s.initErr
	}
	s.initData()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &Signature{
		config: s,
		singer: s.signer,
	}, nil
}

// Signature is sign executor for clients.
type Signature struct {
	config *SignerConfig
	singer Signer
}

// NewSignature create signature by configuration and options.
func NewSignature(opts ...SignerOption) (*Signature, error) {
	cfg, err := NewSignerConfig(opts...)
	if err != nil {
		return nil, err
	}
	s := &Signature{
		config: cfg,
	}
	if s.config.signer != nil {
		s.singer = s.config.signer
	}
	if s.singer == nil {
		singer, err := NewDefaultSigner(cfg)
		if err != nil {
			return nil, err
		}
		s.singer = singer
	}

	return s, nil
}

// Sign signs the http request and attach the signature to http request.
//
// nonce can be empty, if empty, will be generated by config.
// signTime if zero will use time.Now()
func (s *Signature) Sign(r *http.Request, nonce string, signTime time.Time) error {
	ctx := &SigningCtx{
		Request:    r,
		SignTime:   signTime,
		Nonce:      nonce,
		SignedVals: make(map[string]string),
	}
	if err := s.singer.BuildCanonicalRequest(r, ctx); err != nil {
		return err
	}

	if err := s.singer.AttachData(ctx); err != nil {
		return err
	}

	if err := s.singer.CalculateSignature(ctx); err != nil {
		return err
	}
	s.singer.AttachRequest(r, ctx)
	return nil
}

func (s *Signature) Verify(r *http.Request, nonce string, signTime time.Time) (err error) {
	ctx := &SigningCtx{
		Request:    r,
		SignTime:   signTime,
		Nonce:      nonce,
		SignedVals: make(map[string]string),
	}

	sig, err := GetSignedRequestSignature(r, s.config.SignatureHeaderKey, s.config.AuthScheme, s.config.AuthHeaderDelimiter)
	if err != nil {
		return
	}
	if err = s.singer.BuildCanonicalRequest(r, ctx); err != nil {
		return
	}

	if err = s.singer.AttachData(ctx); err != nil {
		return
	}

	if err = s.singer.CalculateSignature(ctx); err != nil {
		return
	}
	if subtle.ConstantTimeCompare([]byte(ctx.Signature), []byte(sig)) == 0 {
		err = ErrInvalidSignature
	}
	return
}

var _ Signer = (*DefaultSigner)(nil)

type DefaultSigner struct {
	*SignerConfig
}

// NewDefaultSigner create default signer with configuration
func NewDefaultSigner(config *SignerConfig) (Signer, error) {
	if config.AuthLookup == "" {
		return nil, errors.New("authLookup must not empty")
	}
	s := &DefaultSigner{
		SignerConfig: config,
	}

	return s, nil
}

func (s *DefaultSigner) BuildCanonicalRequest(r *http.Request, ctx *SigningCtx) (err error) {
	ctx.SignedVals[s.TimestampKey] = FormatSignTime(ctx.SignTime, s.DateFormat)
	if !s.Dry && s.TimestampKey != "" { // add first, lookup can find
		r.Header.Set(s.TimestampKey, ctx.SignedVals[s.TimestampKey])
	}
	if !s.UnsignedPayload {
		if err = s.BuildBodyDigest(r, ctx); err != nil {
			return
		}
	}

	if err = s.BuildCanonicalHeaders(r, ctx); err != nil {
		return
	}
	if err = s.BuildCanonicalUri(r, ctx); err != nil {
		return
	}
	err = s.BuildCanonicalQueryString(r, ctx)
	ctx.SignedHeaders = strings.Join(ctx.CanonicalHeaders, ";")
	return
}

func (s *DefaultSigner) StringToSign(ctx *SigningCtx) error {
	// Create a canonical request
	sb := strings.Builder{}
	sb.WriteString(ctx.Request.Method)
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.CanonicalUri)
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.CanonicalQueryString)
	sb.WriteString(s.Delimiter)
	for _, header := range ctx.CanonicalHeaders {
		sb.WriteString(header)
		sb.WriteString(":")
		sb.WriteString(ctx.SignedVals[header])
		sb.WriteString(s.Delimiter)
	}
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.SignedHeaders)
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.BodyDigest)
	// Create a string to sign
	hs := s.Algorithm.hash()
	hs.Write([]byte(sb.String()))

	sb.Reset()
	sb.WriteString(strings.TrimRight(s.AuthScheme, " "))
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.SignedVals[s.TimestampKey])
	sb.WriteString(s.Delimiter)
	if ctx.CredentialString != "" {
		sb.WriteString(ctx.CredentialString)
		sb.WriteString(s.Delimiter)
	}
	sb.WriteString(hex.EncodeToString(hs.Sum(nil)))

	ctx.StringToSign = sb.String()
	return nil
}

func (s *DefaultSigner) AttachData(_ *SigningCtx) error {
	return nil
}

func (s *DefaultSigner) CalculateSignature(ctx *SigningCtx) error {
	if err := s.StringToSign(ctx); err != nil {
		return err
	}
	var hs hash.Hash
	if s.GetAccessKeySecret() == "" {
		hs = s.Algorithm.hash()
	} else {
		hs = hmac.New(s.Algorithm.hash, []byte(s.GetAccessKeySecret()))
	}
	if _, err := hs.Write([]byte(ctx.StringToSign)); err != nil {
		return err
	}
	ctx.Signature = hex.EncodeToString(hs.Sum(nil))
	return nil
}

// AttachRequest attach the signature to http request.
func (s *DefaultSigner) AttachRequest(r *http.Request, ctx *SigningCtx) {
	if s.Dry {
		return
	}
	if s.SignatureQueryKey != "" {
		r.URL.RawQuery += "&" + s.SignatureQueryKey + "=" + ctx.Signature
	} else {
		sb := strings.Builder{}
		sb.WriteString(s.AuthScheme)
		for _, header := range s.AuthHeaders {
			sb.WriteString(header)
			sb.WriteString("=")
			sb.WriteString(ctx.SignedVals[header])
			sb.WriteString(s.AuthHeaderDelimiter)
		}
		sb.WriteString(signedHeadersElem)
		sb.WriteString(ctx.SignedHeaders)
		sb.WriteString(s.AuthHeaderDelimiter)
		sb.WriteString(signatureElem)
		sb.WriteString(ctx.Signature)
		if h := r.Header.Values(s.SignatureHeaderKey); len(h) > 0 {
			r.Header.Add(s.SignatureHeaderKey, sb.String())
		} else {
			r.Header.Set(s.SignatureHeaderKey, sb.String())
		}
	}
}

type Algorithm struct {
	name string
	hash func() hash.Hash
}

func algorithmFromString(name string) (*Algorithm, error) {
	switch strings.ToLower(name) {
	case AlgorithmSha1.name:
		return AlgorithmSha1, nil
	case AlgorithmSha256.name:
		return AlgorithmSha256, nil
	}
	return nil, ErrUnknownAlgorithm
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (a *Algorithm) UnmarshalText(text []byte) error {
	alg, err := algorithmFromString(string(text))
	if err != nil {
		return err
	}
	*a = *alg
	return nil
}

// BuildCanonicalHeaders implements Signer interface. if a scope-key in the header is empty, it will be ignored.
func (s *DefaultSigner) BuildCanonicalHeaders(r *http.Request, ctx *SigningCtx) error {
	ctx.CanonicalHeaders = make([]string, 0, len(s.ScopeHeaders))
	for _, header := range s.ScopeHeaders {
		var value string
		if strings.EqualFold(header, HeaderXHost) {
			if r.Host != "" {
				value = r.Host
			} else {
				value = r.URL.Host
			}
		} else {
			vs := r.Header.Values(header)
			if len(vs) == 0 {
				continue
			}
			headerValues := make([]string, len(vs))
			for j, v := range vs {
				headerValues[j] = strings.TrimSpace(v)
			}
			value = strings.Join(headerValues, ",")
		}
		ctx.CanonicalHeaders = append(ctx.CanonicalHeaders, header)
		ctx.SignedVals[header] = value
	}
	return nil
}

func (s *DefaultSigner) BuildBodyDigest(r *http.Request, ctx *SigningCtx) (err error) {
	if r.Body == nil {
		switch s.Algorithm.name {
		case AlgorithmSha256.name:
			ctx.BodyDigest = emptyStringSHA256
		case AlgorithmSha1.name:
			ctx.BodyDigest = emptyStringSHA1
		default:
			err = ErrUnknownAlgorithm
		}
		return
	} else {
		bb, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bb))

		h := s.Algorithm.hash()
		h.Write(bb)
		ctx.BodyDigest = hex.EncodeToString(h.Sum(nil))
	}
	return
}

// BuildCanonicalUri implements Signer interface to build canonical uri.
// nolint:stylecheck
func (s *DefaultSigner) BuildCanonicalUri(r *http.Request, ctx *SigningCtx) error {
	uri := getURIPath(r.URL)
	if !s.DisableURIPathEscaping {
		uri = EscapePath(uri, false)
	}
	ctx.CanonicalUri = uri
	return nil
}

// BuildCanonicalQueryString implements Signer interface to build canonical query string.
func (s *DefaultSigner) BuildCanonicalQueryString(r *http.Request, ctx *SigningCtx) error {
	r.URL.RawQuery = strings.ReplaceAll(r.URL.Query().Encode(), "+", "%20")
	ctx.CanonicalQueryString = r.URL.RawQuery
	return nil
}

var _ Signer = (*TokenSigner)(nil)

// TokenSigner is s simple signer used AccessToken to signature http request.
//
// sign element: access_token;timestamp;url.
type TokenSigner struct {
	*SignerConfig
	lookUpSorted []string
}

// NewTokenSigner create token signer with configuration
func NewTokenSigner(config *SignerConfig) (Signer, error) {
	s := &TokenSigner{
		SignerConfig: config,
	}
	s.lookUpSorted = make([]string, 0, len(s.SignedLookups))
	for key := range s.SignedLookups {
		s.lookUpSorted = append(s.lookUpSorted, key)
	}
	sort.Strings(s.lookUpSorted)
	return s, nil
}

func (s TokenSigner) StringToSign(ctx *SigningCtx) error {
	sb := strings.Builder{}
	sl := len(s.lookUpSorted)
	for _, key := range s.lookUpSorted[:sl-1] {
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(ctx.SignedVals[key])
		sb.WriteString(s.Delimiter)
	}
	last := s.lookUpSorted[sl-1]
	sb.WriteString(last)
	sb.WriteString("=")
	sb.WriteString(ctx.SignedVals[last])
	ctx.StringToSign = sb.String()
	return nil
}

func valueFromLookupExp(r *http.Request, exp string) string {
	hps := strings.Split(exp, ">")
	scheme := ""
	l := 0
	if len(hps) > 1 {
		scheme = hps[1] + " "
		l = len(scheme)
	}
	su, err := ValuesFromHeader(r, hps[0], scheme, l)
	if err == nil {
		return su[0]
	}
	return ""
}

func (s TokenSigner) BuildCanonicalRequest(r *http.Request, ctx *SigningCtx) error {
	for key, loc := range s.SignedLookups {
		switch key {
		case s.TimestampKey:
			ctx.SignedVals[key] = FormatSignTime(ctx.SignTime, s.DateFormat)
		case s.NonceKey:
			if ctx.Nonce == "" {
				ctx.Nonce = gds.RandomString(s.NonceLen)
			}
			ctx.SignedVals[key] = ctx.Nonce
		default:
			switch loc {
			case "CanonicalUri":
				url := *r.URL
				url.Fragment = ""
				ctx.SignedVals[key] = url.String()
			default:
				if strings.HasPrefix(loc, "header:") {
					v := valueFromLookupExp(r, loc[7:])
					if v == "" {
						continue
					}
					ctx.SignedVals[key] = v
				} else if strings.HasPrefix(loc, "context:") {
					cv, ok := r.Context().Value(loc[8:]).(string)
					if !ok {
						continue
					}
					ctx.SignedVals[key] = cv
				}
			}
		}
	}
	return nil
}

func (s TokenSigner) AttachData(_ *SigningCtx) error {
	return nil
}

func (s TokenSigner) CalculateSignature(ctx *SigningCtx) error {
	if err := s.StringToSign(ctx); err != nil {
		return err
	}
	var hs hash.Hash
	if s.GetAccessKeySecret() == "" {
		hs = s.Algorithm.hash()
	} else {
		hs = hmac.New(s.Algorithm.hash, []byte(s.GetAccessKeySecret()))
	}
	if _, err := hs.Write([]byte(ctx.StringToSign)); err != nil {
		return err
	}
	ctx.Signature = hex.EncodeToString(hs.Sum(nil))
	return nil
}

func (s TokenSigner) AttachRequest(r *http.Request, ctx *SigningCtx) {
	var sb strings.Builder
	sb.WriteString(s.AuthScheme)
	for _, h := range s.AuthHeaders {
		sb.WriteString(h)
		sb.WriteString("=")
		sb.WriteString(ctx.SignedVals[h])
		sb.WriteString(s.AuthHeaderDelimiter)
	}
	sb.WriteString(signatureElem)
	sb.WriteString(ctx.Signature)

	if h := r.Header.Values(s.SignatureHeaderKey); len(h) > 0 {
		r.Header.Add(s.SignatureHeaderKey, sb.String())
	} else {
		r.Header.Set(s.SignatureHeaderKey, sb.String())
	}
}

// HMACSigner is the signer for hmac auth.
type HMACSigner struct {
	*SignerConfig
	lookUpSorted []string
}

// NewHMACSigner create hmac signer with configuration
func NewHMACSigner(config *SignerConfig) (Signer, error) {
	s := &HMACSigner{
		SignerConfig: config,
	}
	if s.TimestampKey == "" {
		s.TimestampKey = "date"
	}
	if s.DateFormat == "" {
		// GMT format,
		s.DateFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
	}
	if s.Algorithm.name == "" {
		s.Algorithm = *AlgorithmSha256
	}
	s.lookUpSorted = make([]string, 0, len(s.SignedLookups))
	for key := range s.SignedLookups {
		s.lookUpSorted = append(s.lookUpSorted, key)
	}
	sort.Strings(s.lookUpSorted)
	return s, nil
}

func (s HMACSigner) BuildCanonicalRequest(r *http.Request, ctx *SigningCtx) error {
	ctx.SignedVals[s.TimestampKey] = FormatSignTime(ctx.SignTime.UTC(), s.DateFormat)
	for _, header := range s.ScopeHeaders {
		ctx.CanonicalHeaders = append(ctx.CanonicalHeaders, header)
		v := valueFromLookupExp(r, header)
		if v == "" {
			continue
		}
		ctx.SignedVals[header] = v
	}
	sort.Strings(ctx.CanonicalHeaders)
	ctx.CanonicalUri = r.URL.Path
	// CanonicalQueryString fetch from request.URL.RawQuery, use `&` as delimiter, key value pair, sorted by key.
	ctx.CanonicalQueryString = r.URL.Query().Encode()
	return nil
}

// AttachData attach data to request
// CanonicalQueryString fetch from request.URL.RawQuery, use `&` as delimiter, key value pair, sorted by key.
func (s HMACSigner) AttachData(_ *SigningCtx) error {
	return nil
}

func (s HMACSigner) StringToSign(ctx *SigningCtx) error {
	sb := strings.Builder{}
	sb.WriteString(ctx.Request.Method)
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.CanonicalUri)
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.CanonicalQueryString)
	sb.WriteString(s.Delimiter)
	sb.WriteString(s.GetAccessKeyID())
	sb.WriteString(s.Delimiter)
	sb.WriteString(ctx.SignedVals[s.TimestampKey])
	sb.WriteString(s.Delimiter)
	for _, header := range ctx.CanonicalHeaders {
		sb.WriteString(header)
		sb.WriteString(":")
		sb.WriteString(ctx.SignedVals[header])
		sb.WriteString(s.Delimiter)
	}
	ctx.StringToSign = sb.String()
	return nil
}

func (s HMACSigner) CalculateSignature(ctx *SigningCtx) error {
	if err := s.StringToSign(ctx); err != nil {
		return err
	}
	var hs hash.Hash
	if s.GetAccessKeySecret() == "" {
		hs = s.Algorithm.hash()
	} else {
		hs = hmac.New(s.Algorithm.hash, []byte(s.GetAccessKeySecret()))
	}
	if _, err := hs.Write([]byte(ctx.StringToSign)); err != nil {
		return err
	}
	ctx.Signature = base64.StdEncoding.EncodeToString(hs.Sum(nil))
	return nil
}

// AttachRequest attach request with signature.
// The signature can set to header authorization or headers.
func (s HMACSigner) AttachRequest(r *http.Request, ctx *SigningCtx) {
	if s.AuthScheme == "" {
		r.Header.Set(s.SignatureHeaderKey, ctx.Signature)
		switch s.Algorithm.name {
		case AlgorithmSha256.name:
			r.Header.Set("X-HMAC-ALGORITHM", "hmac-sha256")
		case AlgorithmSha1.name:
			r.Header.Set("X-HMAC-ALGORITHM", "hmac-sha1")
		}
		r.Header.Set("X-HMAC-ACCESS-KEY", s.GetAccessKeyID())
		r.Header.Set(s.TimestampKey, ctx.SignedVals[s.TimestampKey])
		r.Header.Set("X-HMAC-SIGNED-HEADERS", strings.Join(ctx.CanonicalHeaders, ";"))
	} else {
		var sb strings.Builder
		sb.WriteString(s.AuthScheme)
		sb.WriteString(s.AuthHeaderDelimiter)
		sb.WriteString(s.GetAccessKeyID())
		sb.WriteString(s.AuthHeaderDelimiter)
		sb.WriteString(ctx.Signature)
		sb.WriteString(s.AuthHeaderDelimiter)
		sb.WriteString(s.Algorithm.name)
		sb.WriteString(s.AuthHeaderDelimiter)
		sb.WriteString(ctx.SignedVals[s.TimestampKey])
		sb.WriteString(s.AuthHeaderDelimiter)
		// use ';'
		for i, h := range s.AuthHeaders {
			sb.WriteString(h)
			if i != len(s.SignedLookups)-1 {
				sb.WriteString(";")
			}
		}
		if h := r.Header.Values(s.SignatureHeaderKey); len(h) > 0 {
			r.Header.Add(s.SignatureHeaderKey, sb.String())
		} else {
			r.Header.Set(s.SignatureHeaderKey, sb.String())
		}
	}
}

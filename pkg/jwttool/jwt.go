package jwttool

import (
	"crypto/rsa"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"io/ioutil"
	"strings"
	"time"
)

var (
	// ErrNoPubKeyFile indicates that the given public key is unreadable
	ErrNoPubKeyFile = errors.New("public key file unreadable")
	// ErrNoPrivKeyFile indicates that the given private key is unreadable
	ErrNoPrivKeyFile = errors.New("private key file unreadable")
	// ErrInvalidPrivKey indicates that the given private key is invalid
	ErrInvalidPrivKey = errors.New("private key invalid")
	// ErrInvalidPubKey indicates the given public key is invalid
	ErrInvalidPubKey = errors.New("public key invalid")
	// ErrInvalidSigningAlgorithm indicates signing algorithm is invalid, needs to be HS256, HS384, HS512, RS256, RS384 or RS512
	ErrInvalidSigningAlgorithm = errors.New("invalid signing algorithm")
)

type JwtParser struct {
	// Realm name to display to the user. Required.
	Realm string

	// signing algorithm - possible values are HS256, HS384, HS512, RS256, RS384 or RS512
	// Optional, default is HS256.
	SigningAlgorithm string

	// Secret key used for signing. Required.
	Key []byte

	// Duration that a jwt token is valid. Optional, defaults to one hour.
	Timeout time.Duration

	// Callback function that will be called during login.
	// Using this function it is possible to add additional payload data to the webtoken.
	// The data is then made available during requests via c.Get("JWT_PAYLOAD").
	// Note that the payload is not encrypted.
	// The attributes mentioned on jwt.io can't be used as keys for the map.
	// Optional, by default no additional data will be set.
	PayloadFunc func(data interface{}) jwt.MapClaims

	// Set the identity key
	IdentityKey string

	// TokenLookup is a string in the form of "<source>:<name>" that is used
	// to extract token from the request.
	// Optional. Default value "header:Authorization".
	// Possible values:
	// - "header:<name>"
	// - "query:<name>"
	// - "cookie:<name>"
	TokenLookup string

	// TokenHeadName is a string in the header. Default value is "Bearer"
	TokenHeadName string

	// Private key file for asymmetric algorithms
	PrivKeyFile string

	// Private Key bytes for asymmetric algorithms
	//
	// Note: PrivKeyFile takes precedence over PrivKeyBytes if both are set
	PrivKeyBytes []byte

	// Public key file for asymmetric algorithms
	PubKeyFile string

	// Public key bytes for asymmetric algorithms.
	//
	// Note: PubKeyFile takes precedence over PubKeyBytes if both are set
	PubKeyBytes []byte

	// Private key
	privKey *rsa.PrivateKey

	// Public key
	pubKey *rsa.PublicKey

	// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
	TimeFunc func() time.Time
}

func (j *JwtParser) ReadKeys() error {
	err := j.privateKey()
	if err != nil {
		return err
	}
	err = j.publicKey()
	if err != nil {
		return err
	}
	return nil
}

func (j *JwtParser) privateKey() error {
	var keyData []byte
	if j.PrivKeyFile == "" {
		keyData = j.PrivKeyBytes
	} else {
		filecontent, err := ioutil.ReadFile(j.PrivKeyFile)
		if err != nil {
			return ErrNoPrivKeyFile
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return ErrInvalidPrivKey
	}
	j.privKey = key
	return nil
}

func (j *JwtParser) publicKey() error {
	var keyData []byte
	if j.PubKeyFile == "" {
		keyData = j.PubKeyBytes
	} else {
		filecontent, err := ioutil.ReadFile(j.PubKeyFile)
		if err != nil {
			return ErrNoPubKeyFile
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return ErrInvalidPubKey
	}
	j.pubKey = key
	return nil
}

func (j *JwtParser) UsingPublicKeyAlgo() bool {
	switch j.SigningAlgorithm {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

func (j *JwtParser) SignedString(token *jwt.Token) (string, error) {
	var tokenString string
	var err error
	if j.UsingPublicKeyAlgo() {
		tokenString, err = token.SignedString(j.privKey)
	} else {
		tokenString, err = token.SignedString(j.Key)
	}
	return tokenString, err
}

type GetToken func(fromType string, key string) (string, error)
type KeyFuncDone func(string)

// ParseToken parse jwt token from gin context
func (j *JwtParser) ParseToken(getToken GetToken, keyFuncDone KeyFuncDone) (*jwt.Token, error) {
	var token string
	var err error

	methods := strings.Split(j.TokenLookup, ",")
	for _, method := range methods {
		if len(token) > 0 {
			break
		}
		parts := strings.Split(strings.TrimSpace(method), ":")
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		token, err = getToken(k, v)
	}

	if err != nil {
		return nil, err
	}

	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(j.SigningAlgorithm) != t.Method {
			return nil, ErrInvalidSigningAlgorithm
		}
		if j.UsingPublicKeyAlgo() {
			return j.pubKey, nil
		}
		if keyFuncDone != nil {
			// save token string if vaild
			keyFuncDone(token)
		}

		return j.Key, nil
	})
}

// ParseTokenString parse jwt token string
func (j *JwtParser) ParseTokenString(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(j.SigningAlgorithm) != t.Method {
			return nil, ErrInvalidSigningAlgorithm
		}
		if j.UsingPublicKeyAlgo() {
			return j.pubKey, nil
		}

		return j.Key, nil
	})
}

// ExtractClaimsFromToken help to extract the JWT claims from token
func ExtractClaimsFromToken(token *jwt.Token) jwt.MapClaims {
	if token == nil {
		return make(jwt.MapClaims)
	}

	claims := jwt.MapClaims{}
	for key, value := range token.Claims.(jwt.MapClaims) {
		claims[key] = value
	}

	return claims
}

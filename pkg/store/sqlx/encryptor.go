package sqlx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

const (
	encryptorAES = "aes-gcm"
)

var encryptors = make(map[string]Encryptor)

func init() {
	RegisterEncryptor(encryptorAES, &aesEncryptor{})
}

// Encryptor is used to encrypt and decrypt password
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// RegisterEncryptor register a new encryptor
func RegisterEncryptor(name string, e Encryptor) {
	encryptors[name] = e
}

// getAESKey get aes key from env.
func getAESKey() []byte {
	return []byte(os.Getenv(aesEnvKey))
}

type aesEncryptor struct {
	key string
}

func (a *aesEncryptor) Encrypt(text string) (string, error) {
	key := getAESKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (a *aesEncryptor) Decrypt(text string) (string, error) {
	key := getAESKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("malformed ciphertext")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

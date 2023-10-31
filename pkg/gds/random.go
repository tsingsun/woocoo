package gds

import (
	"bufio"
	"crypto/rand"
	"io"
	"sync"
)

const randomStringCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const randomStringCharsetLen = 52 // len(randomStringCharset)
const randomStringMaxByte = 255 - (256 % randomStringCharsetLen)

// https://tip.golang.org/doc/go1.19#:~:text=Read%20no%20longer%20buffers%20random%20data%20obtained%20from%20the%20operating%20system%20between%20calls
var randomReaderPool = sync.Pool{New: func() interface{} {
	return bufio.NewReader(rand.Reader)
}}

// RandomString returns a securely generated random string.
//
// Notice: this is method copy from Echo
func RandomString(length uint8) string {
	reader := randomReaderPool.Get().(*bufio.Reader)
	defer randomReaderPool.Put(reader)

	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // perf: avoid read from rand.Reader many times
	var i uint8 = 0

	// security note:
	// we can't just simply do b[i]=randomStringCharset[rb%len(randomStringCharset)],
	// len(len(randomStringCharset)) is 52, and rb is [0, 255], 256 = 52 * 4 + 48.
	// make the first 48 characters more possibly to be generated then others.
	// So we have to skip bytes when rb > randomStringMaxByte

	for {
		_, err := io.ReadFull(reader, r)
		if err != nil {
			panic("unexpected error happened when reading from bufio.NewReader(crypto/rand.Reader)")
		}
		for _, rb := range r {
			if rb > randomStringMaxByte {
				// Skip this number to avoid bias.
				continue
			}
			b[i] = randomStringCharset[rb%randomStringCharsetLen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
}

// RandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random number generator
// fails to function correctly.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// err == nil only if len(b) == n
	if err != nil {
		return nil, err
	}

	return b, nil

}

package gds

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandomString(t *testing.T) {
	var testCases = []struct {
		name       string
		whenLength uint8
	}{
		{
			name:       "ok, 16",
			whenLength: 16,
		},
		{
			name:       "ok, 32",
			whenLength: 32,
		},
		{
			name:       "ok, 12",
			whenLength: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uid := RandomString(tc.whenLength)
			assert.Len(t, uid, int(tc.whenLength))
		})
	}
}

func Test_generateRandomBytes(t *testing.T) {
	b, err := RandomBytes(10)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(b))

	b, err = RandomBytes(0)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(b))
}

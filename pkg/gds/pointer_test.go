package gds

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPtr(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		i := 1
		p := Ptr(i)
		assert.Equal(t, &i, p)
	})
}

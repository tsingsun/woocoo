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

// TestIIF tests the IIF function with different types and conditions.
func TestIIF(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		var cv int
		cv = 1
		assert.Equal(t, 10, IIF(cv == 1, 10, 20))
		assert.Equal(t, 20, IIF(cv == 2, 10, 20))
	})
	t.Run("string", func(t *testing.T) {
		var cv string
		cv = "s"
		assert.Equal(t, "10", IIF(cv == "s", "10", "20"))
		assert.Equal(t, "20", IIF(cv == "2", "10", "20"))
	})
	t.Run("bool", func(t *testing.T) {
		var cv bool
		cv = true
		assert.Equal(t, true, IIF(cv, true, false))
		assert.Equal(t, false, IIF(cv, false, true))
	})
	t.Run("struct", func(t *testing.T) {
		type testStruct struct {
			Field int
		}
		expectedTrue := testStruct{Field: 1}
		expectedFalse := testStruct{Field: 2}
		assert.Equal(t, expectedTrue, IIF(true, expectedTrue, expectedFalse))
		assert.Equal(t, expectedFalse, IIF(false, expectedTrue, expectedFalse))
	})
}

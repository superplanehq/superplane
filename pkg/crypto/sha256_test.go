package crypto

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__SHA256ForMap(t *testing.T) {
	m := map[string]string{"b": "world", "a": "hello", "c": "!"}
	hash, err := SHA256ForMap(m)
	require.NoError(t, err)
	h := sha256.New()
	h.Write([]byte("a=hello,b=world,c=!"))
	assert.Equal(t, fmt.Sprintf("%x", h.Sum(nil)), hash)
}

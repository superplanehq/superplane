package crypto

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__Base64String(t *testing.T) {
	t.Run("returns a non-empty string of the requested entropy", func(t *testing.T) {
		value, err := Base64String(32)
		require.NoError(t, err)
		require.NotEmpty(t, value)
	})

	t.Run("returns different values on subsequent calls", func(t *testing.T) {
		first, err := Base64String(32)
		require.NoError(t, err)
		second, err := Base64String(32)
		require.NoError(t, err)
		require.NotEqual(t, first, second)
	})
}

func Test__NewRandomKey(t *testing.T) {
	t.Run("returns the plain key and its encrypted form", func(t *testing.T) {
		encryptor := NewNoOpEncryptor()

		plainKey, encrypted, err := NewRandomKey(context.Background(), encryptor, "name")
		require.NoError(t, err)
		require.NotEmpty(t, plainKey)
		require.Equal(t, []byte(plainKey), encrypted)
	})

	t.Run("propagates encryptor errors", func(t *testing.T) {
		encryptor := &failingEncryptor{}

		plainKey, encrypted, err := NewRandomKey(context.Background(), encryptor, "name")
		require.Error(t, err)
		require.Empty(t, plainKey)
		require.Nil(t, encrypted)
	})
}

type failingEncryptor struct{}

func (e *failingEncryptor) Encrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, errors.New("encrypt failed")
}

func (e *failingEncryptor) Decrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, errors.New("decrypt failed")
}

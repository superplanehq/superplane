package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__VerifySignature(t *testing.T) {
	key := []byte("secret key")
	data := []byte("data to sign")

	t.Run("if signature is valid, no error", func(t *testing.T) {
		// expected signature, generated with:
		// printf '%s' "data to sign" | openssl dgst -sha256 -hmac "secret key"
		signature := "246df9c6ede92636184fbcf4f03abe33216384885bd018e882870ee3c869967e"
		require.NoError(t, VerifySignature(key, data, signature))
	})

	t.Run("if signature is invalid, error", func(t *testing.T) {
		signature := "invalid signature"
		require.Error(t, VerifySignature(key, data, signature))
	})
}

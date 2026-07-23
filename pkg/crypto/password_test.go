package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func Test__HashPassword(t *testing.T) {
	t.Run("hashes a password into a valid bcrypt hash", func(t *testing.T) {
		password := "s3cr3t-p4ssw0rd"

		hash, err := HashPassword(password)

		require.NoError(t, err)
		require.NotEmpty(t, hash)

		// The hash must not be the plaintext password.
		require.NotEqual(t, password, hash)

		// The produced hash must be a valid bcrypt hash of the original
		// password, verifiable with the underlying bcrypt library.
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)))

		// The hash must be generated using the configured cost factor.
		cost, err := bcrypt.Cost([]byte(hash))
		require.NoError(t, err)
		require.Equal(t, bcryptCost, cost)
	})
}

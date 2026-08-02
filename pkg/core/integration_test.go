package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__NewAuthError(t *testing.T) {
	t.Run("nil error returns nil, not a wrapped nil", func(t *testing.T) {
		err := NewAuthError(nil)
		assert.Nil(t, err)
	})

	t.Run("non-nil error is wrapped and unwraps to the original", func(t *testing.T) {
		inner := errors.New("credentials are invalid or expired")
		err := NewAuthError(inner)
		require.Error(t, err)
		assert.Equal(t, inner.Error(), err.Error())
		assert.ErrorIs(t, err, inner)

		var authErr *AuthError
		require.ErrorAs(t, err, &authErr)
	})
}

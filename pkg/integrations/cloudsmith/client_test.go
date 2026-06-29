package cloudsmith

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parseRepositoryID(t *testing.T) {
	t.Run("valid owner/repository", func(t *testing.T) {
		owner, identifier, err := parseRepositoryID("acme/production")

		require.NoError(t, err)
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "production", identifier)
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		owner, identifier, err := parseRepositoryID("  acme / production  ")

		require.NoError(t, err)
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "production", identifier)
	})

	t.Run("empty value returns error", func(t *testing.T) {
		_, _, err := parseRepositoryID("")

		require.ErrorIs(t, err, errInvalidRepositoryID)
	})

	t.Run("missing slug returns error", func(t *testing.T) {
		_, _, err := parseRepositoryID("acme")

		require.ErrorIs(t, err, errInvalidRepositoryID)
	})

	t.Run("too many segments returns error", func(t *testing.T) {
		_, _, err := parseRepositoryID("acme/team/production")

		require.ErrorIs(t, err, errInvalidRepositoryID)
	})

	t.Run("blank segment returns error", func(t *testing.T) {
		_, _, err := parseRepositoryID("acme/")

		require.ErrorIs(t, err, errInvalidRepositoryID)
	})
}

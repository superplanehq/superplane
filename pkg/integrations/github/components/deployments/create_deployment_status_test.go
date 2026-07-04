package deployments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parseDeploymentID(t *testing.T) {
	t.Parallel()

	t.Run("accepts positive integers", func(t *testing.T) {
		t.Parallel()
		id, err := parseDeploymentID("1")
		require.NoError(t, err)
		assert.Equal(t, int64(1), id)

		id, err = parseDeploymentID("  42  ")
		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("accepts scientific notation from float coercion", func(t *testing.T) {
		t.Parallel()
		id, err := parseDeploymentID("4.671220334e+09")
		require.NoError(t, err)
		assert.Equal(t, int64(4671220334), id)
	})

	t.Run("rejects invalid", func(t *testing.T) {
		t.Parallel()
		for _, raw := range []string{"", "0", "-3", "12.5", "abc"} {
			_, err := parseDeploymentID(raw)
			require.Error(t, err, "raw=%q", raw)
		}
	})
}

func Test__normalizeGitHubDeploymentsAPIURL(t *testing.T) {
	t.Parallel()

	t.Run("prepends https when scheme missing", func(t *testing.T) {
		t.Parallel()
		got, err := normalizeGitHubDeploymentsAPIURL("preview.example.com/path")
		require.NoError(t, err)
		assert.Equal(t, "https://preview.example.com/path", got)
	})

	t.Run("preserves explicit http and https", func(t *testing.T) {
		t.Parallel()
		got, err := normalizeGitHubDeploymentsAPIURL("https://a.example/foo")
		require.NoError(t, err)
		assert.Equal(t, "https://a.example/foo", got)

		got, err = normalizeGitHubDeploymentsAPIURL("http://127.0.0.1:8080/")
		require.NoError(t, err)
		assert.Equal(t, "http://127.0.0.1:8080/", got)
	})

	t.Run("rejects non-http schemes and empty host", func(t *testing.T) {
		t.Parallel()
		for _, raw := range []string{"", "  ", "ftp://x.com", "https://", "javascript:alert(1)"} {
			_, err := normalizeGitHubDeploymentsAPIURL(raw)
			require.Error(t, err, "raw=%q", raw)
		}
	})
}

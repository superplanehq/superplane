package gcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeWorkloadIdentityProviderResourceName(t *testing.T) {
	canonical := "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane"

	t.Run("canonical unchanged", func(t *testing.T) {
		got, err := NormalizeWorkloadIdentityProviderResourceName(canonical)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("IAM REST v1 URL", func(t *testing.T) {
		raw := "https://iam.googleapis.com/v1/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane"
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("embedded in longer paste", func(t *testing.T) {
		raw := "some noise https://iam.googleapis.com/v1/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane trailing"
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("percent-encoded path", func(t *testing.T) {
		raw := "https://iam.googleapis.com/v1/projects%2F123456789%2Flocations%2Fglobal%2FworkloadIdentityPools%2Fmy-pool%2Fproviders%2Fsuperplane"
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("projects-only path", func(t *testing.T) {
		raw := "projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane"
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("case-insensitive projects segment", func(t *testing.T) {
		raw := "//IAM.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane"
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("empty errors", func(t *testing.T) {
		_, err := NormalizeWorkloadIdentityProviderResourceName("   ")
		require.Error(t, err)
	})

	t.Run("no matching path errors", func(t *testing.T) {
		_, err := NormalizeWorkloadIdentityProviderResourceName("https://example.com/nothing/here")
		require.Error(t, err)
	})

	t.Run("wrong shape errors", func(t *testing.T) {
		_, err := NormalizeWorkloadIdentityProviderResourceName("//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers")
		require.Error(t, err)
	})

	t.Run("strip quotes and trim outer whitespace", func(t *testing.T) {
		raw := `  "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/superplane"  `
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})

	t.Run("strip inner newlines", func(t *testing.T) {
		raw := strings.ReplaceAll(canonical, "/", "/\n")
		got, err := NormalizeWorkloadIdentityProviderResourceName(raw)
		require.NoError(t, err)
		assert.Equal(t, canonical, got)
	})
}

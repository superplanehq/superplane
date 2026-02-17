package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateAndParseServiceAccountKey(t *testing.T) {
	t.Run("valid key returns metadata", func(t *testing.T) {
		key := []byte(`{
			"type": "service_account",
			"project_id": "my-project",
			"private_key_id": "abc",
			"private_key": "-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----",
			"client_email": "sa@my-project.iam.gserviceaccount.com",
			"client_id": "123"
		}`)
		meta, err := validateAndParseServiceAccountKey(key)
		require.NoError(t, err)
		assert.Equal(t, "my-project", meta.ProjectID)
		assert.Equal(t, "sa@my-project.iam.gserviceaccount.com", meta.ClientEmail)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := validateAndParseServiceAccountKey([]byte(`{invalid`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("missing required field returns error", func(t *testing.T) {
		key := []byte(`{"type": "service_account", "project_id": "p"}`)
		_, err := validateAndParseServiceAccountKey(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field")
	})

	t.Run("trims project_id and client_email", func(t *testing.T) {
		key := []byte(`{
			"type": "service_account",
			"project_id": "  proj  ",
			"private_key_id": "id",
			"private_key": "key",
			"client_email": "  sa@proj.iam.gserviceaccount.com  ",
			"client_id": "1"
		}`)
		meta, err := validateAndParseServiceAccountKey(key)
		require.NoError(t, err)
		assert.Equal(t, "proj", meta.ProjectID)
		assert.Equal(t, "sa@proj.iam.gserviceaccount.com", meta.ClientEmail)
	})
}

func Test_pathSuffixAfter(t *testing.T) {
	t.Run("prefix found returns suffix", func(t *testing.T) {
		suffix, found := pathSuffixAfter("/integrations/123/gcp/machine-types/e2-medium", pathPrefixMachineTypes)
		require.True(t, found)
		assert.Equal(t, "e2-medium", suffix)
	})

	t.Run("prefix not in path returns not found", func(t *testing.T) {
		_, found := pathSuffixAfter("/gcp/regions", pathPrefixMachineTypes)
		assert.False(t, found)
	})

	t.Run("empty suffix after prefix returns not found", func(t *testing.T) {
		_, found := pathSuffixAfter("/gcp/machine-types/", pathPrefixMachineTypes)
		assert.False(t, found)
	})

	t.Run("prefix at start", func(t *testing.T) {
		suffix, found := pathSuffixAfter("/gcp/images/family/debian-12", pathPrefixImageFamily)
		require.True(t, found)
		assert.Equal(t, "debian-12", suffix)
	})
}

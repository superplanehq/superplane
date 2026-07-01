package secrets

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecretInputFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/secret.yaml"
	content := []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: from-file\nspec:\n  provider: PROVIDER_LOCAL\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	resource, err := parseSecretInput(path, strings.NewReader(""))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-file", resource.Metadata.GetName())
}

func TestParseSecretInputFileRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/secret.yaml"
	content := []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: from-file\nspec:\n  provider: PROVIDER_LOCAL\n  unknown: true\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	_, err := parseSecretInput(path, strings.NewReader(""))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseSecretInputStdin(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: from-stdin\nspec:\n  provider: PROVIDER_LOCAL\n"

	resource, err := parseSecretInput("-", strings.NewReader(yaml))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-stdin", resource.Metadata.GetName())
}

func TestParseSecretInputStdinRejectsUnknownFields(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: from-stdin\nspec:\n  provider: PROVIDER_LOCAL\n  unknown: true\n"

	_, err := parseSecretInput("-", strings.NewReader(yaml))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseSecretInputStdinEmpty(t *testing.T) {
	_, err := parseSecretInput("-", strings.NewReader(""))
	require.Error(t, err)
}

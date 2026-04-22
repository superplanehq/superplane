package secrets

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecretFileRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/secret.yaml"
	content := []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: from-file\nspec:\n  provider: PROVIDER_LOCAL\n  unknown: true\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	_, err := parseSecretFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

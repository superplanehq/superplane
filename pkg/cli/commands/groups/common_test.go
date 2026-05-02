package groups

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGroupInputFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/group.yaml"
	content := []byte("apiVersion: v1\nkind: Group\nmetadata:\n  name: from-file\nspec:\n  displayName: From File\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	resource, err := parseGroupInput(path, strings.NewReader(""))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-file", resource.Metadata.GetName())
}

func TestParseGroupInputFileRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/group.yaml"
	content := []byte("apiVersion: v1\nkind: Group\nmetadata:\n  name: from-file\nspec:\n  displayName: From File\n  unknown: true\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	_, err := parseGroupInput(path, strings.NewReader(""))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseGroupInputStdin(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Group\nmetadata:\n  name: from-stdin\nspec:\n  displayName: From Stdin\n"

	resource, err := parseGroupInput("-", strings.NewReader(yaml))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-stdin", resource.Metadata.GetName())
}

func TestParseGroupInputStdinRejectsUnknownFields(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Group\nmetadata:\n  name: from-stdin\nspec:\n  displayName: From Stdin\n  unknown: true\n"

	_, err := parseGroupInput("-", strings.NewReader(yaml))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseGroupInputStdinEmpty(t *testing.T) {
	_, err := parseGroupInput("-", strings.NewReader(""))
	require.Error(t, err)
}

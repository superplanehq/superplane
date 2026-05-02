package roles

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRoleInputFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/role.yaml"
	content := []byte("apiVersion: v1\nkind: Role\nmetadata:\n  name: from-file\nspec:\n  displayName: From File\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	resource, err := parseRoleInput(path, strings.NewReader(""))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-file", resource.Metadata.GetName())
}

func TestParseRoleInputFileRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/role.yaml"
	content := []byte("apiVersion: v1\nkind: Role\nmetadata:\n  name: from-file\nspec:\n  displayName: From File\n  unknown: true\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	_, err := parseRoleInput(path, strings.NewReader(""))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseRoleInputStdin(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Role\nmetadata:\n  name: from-stdin\nspec:\n  displayName: From Stdin\n"

	resource, err := parseRoleInput("-", strings.NewReader(yaml))
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.Equal(t, "from-stdin", resource.Metadata.GetName())
}

func TestParseRoleInputStdinRejectsUnknownFields(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Role\nmetadata:\n  name: from-stdin\nspec:\n  displayName: From Stdin\n  unknown: true\n"

	_, err := parseRoleInput("-", strings.NewReader(yaml))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Contains(t, err.Error(), "unknown")
}

func TestParseRoleInputStdinEmpty(t *testing.T) {
	_, err := parseRoleInput("-", strings.NewReader(""))
	require.Error(t, err)
}

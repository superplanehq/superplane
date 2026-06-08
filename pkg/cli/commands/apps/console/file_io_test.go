package console

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveYAMLSourceFlagWinsAndReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(path, []byte("from-flag"), 0o600))

	data, source, err := resolveYAMLSource(bytes.NewBufferString("from-stdin"), path, "")
	require.NoError(t, err)
	require.Equal(t, "from-flag", string(data))
	require.Equal(t, path, source)
}

func TestResolveYAMLSourcePositional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(path, []byte("positional"), 0o600))

	data, source, err := resolveYAMLSource(nil, "", path)
	require.NoError(t, err)
	require.Equal(t, "positional", string(data))
	require.Equal(t, path, source)
}

func TestResolveYAMLSourceStdin(t *testing.T) {
	data, source, err := resolveYAMLSource(bytes.NewBufferString("piped"), "-", "")
	require.NoError(t, err)
	require.Equal(t, "piped", string(data))
	require.Equal(t, "stdin", source)
}

func TestResolveYAMLSourceSamePathTwiceIsFine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(path, []byte("same"), 0o600))

	_, _, err := resolveYAMLSource(nil, path, path)
	require.NoError(t, err)
}

func TestResolveYAMLSourceConflict(t *testing.T) {
	_, _, err := resolveYAMLSource(nil, "a.yaml", "b.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}

func TestResolveYAMLSourceMissing(t *testing.T) {
	_, _, err := resolveYAMLSource(nil, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no YAML source provided")
}

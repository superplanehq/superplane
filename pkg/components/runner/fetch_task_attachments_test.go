package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchTaskAttachmentsScriptVerifiesChecksum(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is not installed")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}

	payload := []byte("clip-bytes")
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"clip.mp4","dest":"01-clip.mp4","url":"`+server.URL+`","size_bytes":10,"checksum":"`+checksum+`"}]
}`), 0o644))

	cmd := exec.Command("bash", "fetch_task_attachments.sh")
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	got, err := os.ReadFile(filepath.Join(attachments, "01-clip.mp4"))
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestFetchTaskAttachmentsScriptRejectsChecksumMismatch(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is not installed")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("clip-bytes"))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"clip.mp4","dest":"01-clip.mp4","url":"`+server.URL+`","size_bytes":10,"checksum":"deadbeef"}]
}`), 0o644))

	cmd := exec.Command("bash", "fetch_task_attachments.sh")
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "checksum mismatch")
	assert.NoFileExists(t, filepath.Join(attachments, "01-clip.mp4"))
}

func TestFetchTaskAttachmentsScriptRejectsSizeMismatch(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is not installed")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("clip-bytes"))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"clip.mp4","dest":"01-clip.mp4","url":"`+server.URL+`","size_bytes":1}]
}`), 0o644))

	cmd := exec.Command("bash", "fetch_task_attachments.sh")
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "does not match")
}

package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestFetchTaskAttachmentsScriptMarksChecksumMismatchFailed(t *testing.T) {
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
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "checksum mismatch")
	assert.NoFileExists(t, filepath.Join(attachments, "01-clip.mp4"))
	assert.Equal(t, "failed", fetchManifestFile(t, attachments, "01-clip.mp4")["status"])
	assert.Equal(t, "checksum_mismatch", fetchManifestFile(t, attachments, "01-clip.mp4")["reason"])
}

func TestFetchTaskAttachmentsScriptMarksSizeMismatchFailed(t *testing.T) {
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
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "does not match")
	assert.Equal(t, "failed", fetchManifestFile(t, attachments, "01-clip.mp4")["status"])
}

func TestFetchTaskAttachmentsScriptContinuesAfterOneFailure(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is not installed")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}

	payload := []byte("ok-file")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [
    {"filename":"missing.png","dest":"01-missing.png","url":"`+server.URL+`/missing"},
    {"filename":"ok.png","dest":"02-ok.png","url":"`+server.URL+`/ok"}
  ]
}`), 0o644))

	cmd := exec.Command("bash", "fetch_task_attachments.sh")
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, "failed", fetchManifestFile(t, attachments, "01-missing.png")["status"])
	assert.Equal(t, "downloaded", fetchManifestFile(t, attachments, "02-ok.png")["status"])
	got, err := os.ReadFile(filepath.Join(attachments, "02-ok.png"))
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestFetchTaskAttachmentsScriptSkipsHostedVideo(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("<html>video</html>"))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"youtube","dest":"01-youtube","url":"`+server.URL+`","kind":"hosted_video","status":"pending"}]
}`), 0o644))

	cmd := exec.Command("bash", "fetch_task_attachments.sh")
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, 0, hits)
	assert.NoFileExists(t, filepath.Join(attachments, "01-youtube"))
	assert.Equal(t, "pending", fetchManifestFile(t, attachments, "01-youtube")["status"])
}

func fetchManifestFile(t *testing.T, attachments, dest string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(attachments, "manifest.json"))
	require.NoError(t, err)
	var manifest struct {
		Files []map[string]any `json:"files"`
	}
	require.NoError(t, json.Unmarshal(body, &manifest))
	for _, file := range manifest.Files {
		if file["dest"] == dest {
			return file
		}
	}
	t.Fatalf("missing dest %s", dest)
	return nil
}

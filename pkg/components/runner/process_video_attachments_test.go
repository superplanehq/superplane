package runner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessVideoAttachmentsScriptFailsWithoutToolchain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	attachments := filepath.Join(dir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{"version":1,"files":[{"dest":"01-clip.mp4","kind":"video"}]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "01-clip.mp4"), []byte("not-a-video"), 0o644))

	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+dir,
		"PATH=/usr/bin:/bin",
		"WHISPER_MODEL="+filepath.Join(dir, "missing.bin"),
	)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "required for video task files")
}

func TestProcessVideoAttachmentsScriptMarksMalformedVideo(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "malformed.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")

	out := runProcessVideo(t, dir, env)
	assert.Contains(t, readIndex(t, attachments), "undecodable")
	assert.Contains(t, string(out), "undecodable")
}

func TestProcessVideoAttachmentsScriptRejectsMisleadingMIME(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "misleading.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")

	_ = runProcessVideo(t, dir, env)
	assert.Contains(t, readIndex(t, attachments), "undecodable")
}

func TestProcessVideoAttachmentsScriptExtractsFramesFromSilentVideo(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "tiny.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")

	out := runProcessVideo(t, dir, env)
	index := readIndex(t, attachments)
	assert.Contains(t, index, "no_audio")
	assert.Contains(t, string(out), "frames")
	entries, err := os.ReadDir(filepath.Join(attachments, "01-clip.mp4.frames"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "ready", item["status"])
	assert.Equal(t, "no_audio", item["reason"])
}

func TestProcessVideoAttachmentsScriptRejectsOverDuration(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	clip := filepath.Join(attachments, "01-clip.mp4")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=black:s=16x16:d=2", "-an", clip)
	require.NoError(t, cmd.Run())
	writeVideoManifest(t, attachments, "01-clip.mp4")
	env = append(env, "VIDEO_MAX_DURATION_SECONDS=1")

	out := runProcessVideo(t, dir, env)
	assert.Contains(t, string(out), "exceeds")
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "failed", item["status"])
	assert.Equal(t, "duration_exceeds_limit", item["reason"])
}

func TestProcessVideoAttachmentsScriptMarksWhisperFailurePartial(t *testing.T) {
	requireLookPath(t, "ffmpeg", "ffprobe", "python3")
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "silent.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "whisper-cli"), []byte("#!/bin/sh\nexit 1\n"), 0o755))
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	env := append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"WHISPER_MODEL="+model,
	)

	_ = runProcessVideo(t, dir, env)
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "partial", item["status"])
	assert.Equal(t, "transcription_failed", item["reason"])
	assert.NotEmpty(t, item["frames"])
}

func requireLookPath(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			t.Skip(name + " is not installed")
		}
	}
}

func processVideoTestEnv(t *testing.T) []string {
	t.Helper()
	requireLookPath(t, "ffmpeg", "ffprobe", "python3")
	env := append([]string{}, os.Environ()...)
	if _, err := exec.LookPath("whisper-cli"); err == nil {
		model := os.Getenv("WHISPER_MODEL")
		if model == "" {
			model = "/usr/local/share/whisper/ggml-tiny.bin"
		}
		if _, err := os.Stat(model); err == nil {
			return append(env, "WHISPER_MODEL="+model)
		}
	}
	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "whisper-cli"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	return append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"), "WHISPER_MODEL="+model)
}

func newAttachmentDir(t *testing.T) (taskDir, attachments string) {
	t.Helper()
	taskDir = t.TempDir()
	attachments = filepath.Join(taskDir, "attachments")
	require.NoError(t, os.MkdirAll(attachments, 0o755))
	return taskDir, attachments
}

func copyMediaFixture(t *testing.T, attachments, name, dest string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "media", name))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(attachments, dest), body, 0o644))
}

func writeVideoManifest(t *testing.T, attachments, dest string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"max_duration_seconds": 900, "max_frames": 24, "max_frame_width": 1280, "process_timeout_seconds": 120, "disk_budget_bytes": 2147483648},
  "files": [{"filename":"clip.mp4","content_type":"video/mp4","dest":"`+dest+`","kind":"video"}]
}`), 0o644))
}

func runProcessVideo(t *testing.T, taskDir string, env []string) []byte {
	t.Helper()
	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Env = append(env, "SUPERPLANE_TASK_DIR="+taskDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return out
}

func readIndex(t *testing.T, attachments string) string {
	t.Helper()
	index, err := os.ReadFile(filepath.Join(attachments, "INDEX.md"))
	require.NoError(t, err)
	return string(index)
}

func manifestFile(t *testing.T, attachments, dest string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(attachments, "manifest.json"))
	require.NoError(t, err)
	var payload struct {
		Files []map[string]any `json:"files"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	for _, item := range payload.Files {
		if item["dest"] == dest {
			return item
		}
	}
	t.Fatalf("missing dest %s", dest)
	return nil
}

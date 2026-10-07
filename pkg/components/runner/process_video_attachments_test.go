package runner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	cmd.Env = replaceEnv(os.Environ(), "WHISPER_MODEL", filepath.Join(dir, "missing.bin"))
	cmd.Env = append(cmd.Env, "SUPERPLANE_TASK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "WHISPER_MODEL")
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

func TestProcessVideoAttachmentsScriptRejectsExcessiveSourcePixels(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "tiny.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")
	env = append(env, "VIDEO_MAX_SOURCE_PIXELS=100")

	_ = runProcessVideo(t, dir, env)
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "failed", item["status"])
	assert.Equal(t, "dimensions_exceed_limit", item["reason"])
}

func TestProcessVideoAttachmentsScriptEnforcesBudgetDuringFrameExtraction(t *testing.T) {
	env := processVideoTestEnv(t)
	dir, attachments := newAttachmentDir(t)
	copyMediaFixture(t, attachments, "tiny.mp4", "01-clip.mp4")
	writeVideoManifest(t, attachments, "01-clip.mp4")
	initialSize := directorySize(t, attachments)
	env = append(env, "VIDEO_DISK_BUDGET_BYTES="+strconv.FormatInt(initialSize+100, 10))

	_ = runProcessVideo(t, dir, env)
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "failed", item["status"])
	assert.Equal(t, "disk_budget_exceeded", item["reason"])
	_, err := os.Stat(filepath.Join(attachments, "01-clip.mp4.frames"))
	assert.ErrorIs(t, err, os.ErrNotExist)
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
	assert.Equal(t, float64(1), item["transcript_attempts"])
	assert.NotEmpty(t, item["frames"])
}

func TestProcessVideoAttachmentsScriptIndexesImagesWithoutMediaTools(t *testing.T) {
	t.Parallel()

	dir, attachments := newAttachmentDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "01-shot.png"), []byte("png"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"shot.png","content_type":"image/png","dest":"01-shot.png","kind":"image"}]
}`), 0o644))

	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+dir,
		"PATH=/usr/bin:/bin",
		"WHISPER_MODEL="+filepath.Join(dir, "missing.bin"),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	index := readIndex(t, attachments)
	assert.Contains(t, index, "image:")
	assert.Contains(t, index, "$SUPERPLANE_TASK_DIR/attachments/01-shot.png")
	assert.NotContains(t, index, "inspect_attachment")
	assert.Equal(t, "ready", manifestFile(t, attachments, "01-shot.png")["status"])
}

func TestProcessVideoAttachmentsScriptSkipsProcessedMediaWithoutToolchain(t *testing.T) {
	t.Parallel()

	dir, attachments := newAttachmentDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "01-ready.mp4"), []byte("ready"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "02-failed.mp4"), []byte("failed"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "03-note.wav"), []byte("audio"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [
    {"filename":"ready.mp4","content_type":"video/mp4","dest":"01-ready.mp4","kind":"video","status":"ready"},
    {"filename":"failed.mp4","content_type":"video/mp4","dest":"02-failed.mp4","kind":"video","status":"failed","reason":"undecodable"},
    {"filename":"note.wav","content_type":"audio/wav","dest":"03-note.wav","kind":"audio","status":"ready"}
  ]
}`), 0o644))

	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+dir,
		"PATH=/usr/bin:/bin",
		"WHISPER_MODEL="+filepath.Join(dir, "missing.bin"),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "No video or audio files in task attachments.")
	assert.Equal(t, "ready", manifestFile(t, attachments, "01-ready.mp4")["status"])
	assert.Equal(t, "failed", manifestFile(t, attachments, "02-failed.mp4")["status"])
	assert.Equal(t, "ready", manifestFile(t, attachments, "03-note.wav")["status"])
}

func TestProcessVideoAttachmentsScriptRetriesPartialWithoutToolchain(t *testing.T) {
	t.Parallel()

	dir, attachments := newAttachmentDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "01-note.wav"), []byte("audio"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"note.wav","content_type":"audio/wav","dest":"01-note.wav","kind":"audio","status":"partial","reason":"transcription_failed"}]
}`), 0o644))

	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+dir,
		"PATH=/usr/bin:/bin",
		"WHISPER_MODEL="+filepath.Join(dir, "missing.bin"),
	)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "WHISPER_MODEL")
}

func TestProcessVideoAttachmentsScriptStopsPartialAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	dir, attachments := newAttachmentDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "01-note.wav"), []byte("audio"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "files": [{"filename":"note.wav","content_type":"audio/wav","dest":"01-note.wav","kind":"audio","status":"partial","reason":"transcription_failed","transcript_attempts":3}]
}`), 0o644))

	cmd := exec.Command("bash", "process_video_attachments.sh")
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+dir,
		"PATH=/usr/bin:/bin",
		"WHISPER_MODEL="+filepath.Join(dir, "missing.bin"),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "No video or audio files in task attachments.")
	assert.Equal(t, "failed", manifestFile(t, attachments, "01-note.wav")["status"])
}

func TestProcessVideoAttachmentsScriptRetriesPartialAudioTranscript(t *testing.T) {
	requireLookPath(t, "ffmpeg", "ffprobe", "python3")
	dir, attachments := newAttachmentDir(t)
	clip := filepath.Join(attachments, "01-note.wav")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", clip)
	require.NoError(t, cmd.Run())
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"max_duration_seconds": 900, "max_frames": 24, "max_frame_width": 1280, "process_timeout_seconds": 120, "disk_budget_bytes": 2147483648},
  "files": [{"filename":"note.wav","content_type":"audio/wav","dest":"01-note.wav","kind":"audio","status":"partial","reason":"transcription_failed","duration_seconds":1}]
}`), 0o644))

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "whisper-cli"), []byte("#!/bin/sh\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"-of\" ]; then printf 'hello\\n' > \"$2.txt\"; shift 2; continue; fi\n  shift\ndone\n"), 0o755))
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	env := replaceEnv(replaceEnv(os.Environ(), "PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")), "WHISPER_MODEL", model)

	out := runProcessVideo(t, dir, env)
	assert.Contains(t, string(out), "transcript ready")
	item := manifestFile(t, attachments, "01-note.wav")
	assert.Equal(t, "ready", item["status"])
	assert.Equal(t, "", item["reason"])
	assert.Equal(t, "01-note.wav.transcript.txt", item["transcript"])
}

func TestProcessVideoAttachmentsScriptRetriesPartialVideoWithoutReframing(t *testing.T) {
	requireLookPath(t, "ffmpeg", "ffprobe", "python3")
	dir, attachments := newAttachmentDir(t)
	clip := filepath.Join(attachments, "01-clip.mp4")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=16x16:d=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-shortest", clip)
	require.NoError(t, cmd.Run())
	framesDir := filepath.Join(attachments, "01-clip.mp4.frames")
	require.NoError(t, os.MkdirAll(framesDir, 0o755))
	framePath := filepath.Join(framesDir, "frame-000.000.jpg")
	require.NoError(t, os.WriteFile(framePath, []byte("kept-frame"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"max_duration_seconds": 900, "max_frames": 24, "max_frame_width": 1280, "process_timeout_seconds": 120, "disk_budget_bytes": 2147483648},
  "files": [{"filename":"clip.mp4","content_type":"video/mp4","dest":"01-clip.mp4","kind":"video","status":"partial","reason":"transcription_failed","duration_seconds":1,"frames_dir":"01-clip.mp4.frames","frames":[{"path":"01-clip.mp4.frames/frame-000.000.jpg","timestamp_seconds":0}]}]
}`), 0o644))

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "whisper-cli"), []byte("#!/bin/sh\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"-of\" ]; then printf 'hello\\n' > \"$2.txt\"; shift 2; continue; fi\n  shift\ndone\n"), 0o755))
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	env := replaceEnv(replaceEnv(os.Environ(), "PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")), "WHISPER_MODEL", model)

	out := runProcessVideo(t, dir, env)
	assert.Contains(t, string(out), "transcript ready")
	item := manifestFile(t, attachments, "01-clip.mp4")
	assert.Equal(t, "ready", item["status"])
	assert.Equal(t, "", item["reason"])
	body, err := os.ReadFile(framePath)
	require.NoError(t, err)
	assert.Equal(t, "kept-frame", string(body))
}

func TestProcessVideoAttachmentsScriptTranscribesAudioWithoutFrames(t *testing.T) {
	requireLookPath(t, "ffmpeg", "ffprobe", "python3")
	dir, attachments := newAttachmentDir(t)
	clip := filepath.Join(attachments, "01-note.wav")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", clip)
	require.NoError(t, cmd.Run())
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"max_duration_seconds": 900, "max_frames": 24, "max_frame_width": 1280, "process_timeout_seconds": 120, "disk_budget_bytes": 2147483648},
  "files": [{"filename":"note.wav","content_type":"audio/wav","dest":"01-note.wav","kind":"audio"}]
}`), 0o644))

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "whisper-cli"), []byte("#!/bin/sh\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"-of\" ]; then printf 'hello\\n' > \"$2.txt\"; shift 2; continue; fi\n  shift\ndone\n"), 0o755))
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	env := replaceEnv(replaceEnv(os.Environ(), "PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")), "WHISPER_MODEL", model)

	out := runProcessVideo(t, dir, env)
	assert.Contains(t, string(out), "transcript ready")
	index := readIndex(t, attachments)
	assert.Contains(t, index, "transcript")
	assert.NotContains(t, index, "frames:")
	assert.NotContains(t, index, "no_video_stream")
	item := manifestFile(t, attachments, "01-note.wav")
	assert.Equal(t, "ready", item["status"])
	assert.Equal(t, "01-note.wav.transcript.txt", item["transcript"])
	assert.Nil(t, item["frames"])
}

func TestProcessHostedVideoDeletesDownloadAndKeepsFrames(t *testing.T) {
	requireLookPath(t, "python3")
	dir, attachments := newAttachmentDir(t)
	page := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"max_duration_seconds": 900, "hosted_video_max_duration_seconds": 300, "hosted_video_max_bytes": 268435456},
  "files": [{"filename":"youtube-dQw4w9WgXcQ","url":"`+page+`","dest":"01-youtube-dQw4w9WgXcQ","kind":"hosted_video","status":"pending"}]
}`), 0o644))

	env := hostedVideoToolEnv(t, dir, `{"duration": 2, "is_live": false, "live_status": "not_live"}`)
	out := runProcessVideo(t, dir, env)
	assert.NotContains(t, string(out), "download_failed")
	assert.NoFileExists(t, filepath.Join(dir, ".hosted-videos", "01-youtube-dQw4w9WgXcQ"))
	_, hostedDirErr := os.Stat(filepath.Join(dir, ".hosted-videos"))
	assert.ErrorIs(t, hostedDirErr, os.ErrNotExist)
	assert.NoFileExists(t, filepath.Join(attachments, "01-youtube-dQw4w9WgXcQ"))
	item := manifestFile(t, attachments, "01-youtube-dQw4w9WgXcQ")
	frames, _ := item["frames"].([]any)
	if len(frames) == 0 {
		assert.NotEmpty(t, item["reason"])
	}
	index := readIndex(t, attachments)
	assert.Contains(t, index, page)
	assert.Contains(t, index, "Do not fetch a hosted video page URL.")
}

func TestProcessHostedVideoDoesNotDownloadWhenDurationExceedsLimit(t *testing.T) {
	requireLookPath(t, "python3")
	dir, attachments := newAttachmentDir(t)
	page := "https://www.loom.com/share/0123456789abcdef0123456789abcdef"
	require.NoError(t, os.WriteFile(filepath.Join(attachments, "manifest.json"), []byte(`{
  "version": 1,
  "policy": {"hosted_video_max_duration_seconds": 300},
  "files": [{"filename":"loom","url":"`+page+`","dest":"01-loom","kind":"hosted_video","status":"pending"}]
}`), 0o644))
	downloadLog := filepath.Join(dir, "downloaded.txt")
	env := hostedVideoToolEnv(t, dir, `{"duration": 301, "is_live": false, "live_status": "not_live"}`)
	env = append(env, "FAKE_YTDLP_DOWNLOAD_LOG="+downloadLog)
	_ = runProcessVideo(t, dir, env)
	assert.NoFileExists(t, downloadLog)
	assert.NoFileExists(t, filepath.Join(dir, ".hosted-videos", "01-loom"))
	item := manifestFile(t, attachments, "01-loom")
	assert.Equal(t, "failed", item["status"])
	assert.Equal(t, "duration_exceeds_limit", item["reason"])
}

func hostedVideoToolEnv(t *testing.T, taskDir, metadata string) []string {
	t.Helper()
	binDir := t.TempDir()
	writeFakeTool(t, binDir, "yt-dlp", `#!/usr/bin/env python3
import os, sys
args = sys.argv[1:]
if "--dump-single-json" in args:
    print(os.environ.get("FAKE_YTDLP_JSON", "{}"))
    raise SystemExit(0)
out = None
for index, arg in enumerate(args):
    if arg == "-o" and index + 1 < len(args):
        out = args[index + 1]
if out:
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "wb") as handle:
        handle.write(b"short-video")
log = os.environ.get("FAKE_YTDLP_DOWNLOAD_LOG")
if log:
    with open(log, "a", encoding="utf-8") as handle:
        handle.write("download\n")
`)
	writeFakeTool(t, binDir, "ffprobe", `#!/usr/bin/env python3
import json
json.dump({"format": {"duration": "1.0", "format_name": "mov,mp4"}, "streams": [{"codec_type": "video", "codec_name": "h264", "width": 16, "height": 16}]}, __import__("sys").stdout)
`)
	writeFakeTool(t, binDir, "ffmpeg", `#!/usr/bin/env python3
import sys
out = sys.argv[-1]
if out.endswith(".jpg"):
    with open(out, "wb") as handle:
        handle.write(b"\xff\xd8\xff\xd9")
`)
	writeFakeTool(t, binDir, "whisper-cli", "#!/bin/sh\nexit 0\n")
	model := filepath.Join(binDir, "ggml-tiny.bin")
	require.NoError(t, os.WriteFile(model, []byte("x"), 0o644))
	return append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"WHISPER_MODEL="+model,
		"FAKE_YTDLP_JSON="+metadata,
	)
}

func writeFakeTool(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755))
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	next := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			next = append(next, item)
		}
	}
	return append(next, prefix+value)
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

func directorySize(t *testing.T, root string) int64 {
	t.Helper()
	var total int64
	require.NoError(t, filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	}))
	return total
}

#!/usr/bin/env bash
# Probe task videos, extract bounded timestamped frames, and transcribe audio.
# Fail before the agent starts when the media toolchain is missing.
set -euo pipefail

attachments="${SUPERPLANE_TASK_DIR:?}/attachments"
manifest="$attachments/manifest.json"
index="$attachments/INDEX.md"
WHISPER_MODEL="${WHISPER_MODEL:-/usr/local/share/whisper/ggml-tiny.bin}"

MAX_DURATION_SECONDS="${VIDEO_MAX_DURATION_SECONDS:-900}"
MAX_FRAMES="${VIDEO_MAX_FRAMES:-24}"
MAX_WIDTH="${VIDEO_MAX_FRAME_WIDTH:-1280}"
PROCESS_TIMEOUT="${VIDEO_PROCESS_TIMEOUT_SECONDS:-120}"
DISK_BUDGET_BYTES="${VIDEO_DISK_BUDGET_BYTES:-2147483648}"

if [ ! -d "$attachments" ]; then
  printf 'attachments directory is missing.\n' >&2
  exit 1
fi
if [ ! -f "$manifest" ]; then
  printf 'attachments/manifest.json is missing.\n' >&2
  exit 1
fi

missing=0
for cmd in ffmpeg ffprobe whisper-cli python3; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    printf '%s is required for video task files. Install the runner media toolchain and rebuild this image.\n' "$cmd" >&2
    missing=1
  fi
done
if [ ! -s "$WHISPER_MODEL" ]; then
  printf 'WHISPER_MODEL (%s) is missing. Bake ggml-tiny.bin into the runner image. Do not download models during a task.\n' "$WHISPER_MODEL" >&2
  missing=1
fi
if [ "$missing" -ne 0 ]; then
  exit 1
fi

export ATTACHMENTS_DIR="$attachments"
export MANIFEST_PATH="$manifest"
export INDEX_PATH="$index"
export WHISPER_MODEL
export MAX_DURATION_SECONDS MAX_FRAMES MAX_WIDTH PROCESS_TIMEOUT DISK_BUDGET_BYTES

python3 - <<'PY'
import json
import math
import os
import shutil
import subprocess
import sys
from pathlib import Path

attachments = Path(os.environ["ATTACHMENTS_DIR"])
manifest_path = Path(os.environ["MANIFEST_PATH"])
index_path = Path(os.environ["INDEX_PATH"])
whisper_model = os.environ["WHISPER_MODEL"]
max_duration = float(os.environ["MAX_DURATION_SECONDS"])
max_frames = int(os.environ["MAX_FRAMES"])
max_width = int(os.environ["MAX_WIDTH"])
process_timeout = int(os.environ["PROCESS_TIMEOUT"])
disk_budget = int(os.environ["DISK_BUDGET_BYTES"])

VIDEO_TYPES = {
    "video/mp4",
    "video/webm",
    "video/quicktime",
    "video/ogg",
    "video/x-m4v",
    "video/x-matroska",
}
VIDEO_SUFFIXES = {".mp4", ".webm", ".mov", ".ogv", ".ogg", ".m4v", ".mkv"}


def load_manifest():
    with manifest_path.open(encoding="utf-8") as handle:
        return json.load(handle)


def save_manifest(manifest):
    with manifest_path.open("w", encoding="utf-8") as handle:
        json.dump(manifest, handle, indent=2)
        handle.write("\n")


def dir_size(path: Path) -> int:
    total = 0
    for root, _, files in os.walk(path):
        for name in files:
            total += (Path(root) / name).stat().st_size
    return total


def looks_like_video(item, dest: Path) -> bool:
    content_type = (item.get("content_type") or "").split(";")[0].strip().lower()
    if content_type in VIDEO_TYPES:
        return True
    suffix = dest.suffix.lower()
    return suffix in VIDEO_SUFFIXES


def run(cmd, timeout=process_timeout):
    return subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)


def probe(path: Path):
    result = run([
        "ffprobe", "-v", "error", "-print_format", "json",
        "-show_format", "-show_streams", str(path),
    ], timeout=30)
    if result.returncode != 0:
        return None, "undecodable"
    try:
        payload = json.loads(result.stdout or "{}")
    except json.JSONDecodeError:
        return None, "undecodable"
    return payload, ""


def duration_seconds(payload) -> float:
    fmt = payload.get("format") or {}
    raw = fmt.get("duration")
    try:
        value = float(raw)
    except (TypeError, ValueError):
        return 0.0
    if math.isnan(value) or value < 0:
        return 0.0
    return value


def streams(payload, kind):
    return [s for s in payload.get("streams") or [] if s.get("codec_type") == kind]


IMAGE_CODECS = {"png", "bmp", "gif", "tiff", "webp", "ppm", "pam"}


def usable_video_stream(stream) -> bool:
    codec = (stream.get("codec_name") or "").lower()
    if codec in IMAGE_CODECS:
        return False
    try:
        width = int(stream.get("width") or 0)
        height = int(stream.get("height") or 0)
    except (TypeError, ValueError):
        return False
    return width > 0 and height > 0


def unique_timestamps(duration: float) -> list[float]:
    stamps = [0.0]
    if duration > 0:
        stamps.append(max(0.0, duration - 0.05))
        for index in range(1, 7):
            stamps.append(duration * index / 7.0)
    rounded = []
    seen = set()
    for stamp in stamps:
        key = round(stamp * 4) / 4.0
        if key in seen:
            continue
        seen.add(key)
        rounded.append(max(0.0, stamp))
    rounded.sort()
    return rounded[:max_frames]


def scene_timestamps(path: Path, duration: float) -> list[float]:
    result = run([
        "ffmpeg", "-hide_banner", "-loglevel", "info", "-i", str(path),
        "-vf", "select=gt(scene\\,0.25),showinfo", "-an", "-f", "null", "-",
    ], timeout=min(process_timeout, 60))
    stamps = []
    for line in (result.stderr or "").splitlines():
        if "pts_time:" not in line:
            continue
        try:
            part = line.split("pts_time:")[1].split()[0]
            stamp = float(part)
        except (IndexError, ValueError):
            continue
        if 0 <= stamp <= duration:
            stamps.append(stamp)
    return stamps


def extract_frame(path: Path, stamp: float, dest: Path) -> bool:
    dest.parent.mkdir(parents=True, exist_ok=True)
    result = run([
        "ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
        "-ss", f"{stamp:.3f}", "-i", str(path),
        "-frames:v", "1",
        "-vf", f"scale='min({max_width},iw)':-2",
        str(dest),
    ], timeout=30)
    return result.returncode == 0 and dest.is_file() and dest.stat().st_size > 0


def transcribe(path: Path, dest: Path, duration: float) -> str:
    wav = dest.with_suffix(".wav")
    try:
        cap = min(duration if duration > 0 else max_duration, max_duration)
        result = run([
            "ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
            "-i", str(path), "-vn", "-ac", "1", "-ar", "16000",
            "-c:a", "pcm_s16le", "-t", f"{cap:.3f}", str(wav),
        ], timeout=process_timeout)
        if result.returncode != 0 or not wav.is_file() or wav.stat().st_size == 0:
            return "transcription_failed"
        out_base = str(dest.with_suffix(""))
        whisper = run([
            "whisper-cli", "-m", whisper_model, "-f", str(wav),
            "-otxt", "-of", out_base, "-l", "auto",
        ], timeout=process_timeout)
        txt = Path(out_base + ".txt")
        if whisper.returncode != 0 or not txt.is_file():
            dest.write_text("Transcription failed.\n", encoding="utf-8")
            return "transcription_failed"
        if txt != dest:
            txt.replace(dest)
        if dest.stat().st_size == 0:
            dest.write_text("Transcription failed.\n", encoding="utf-8")
            return "transcription_failed"
        return ""
    except subprocess.TimeoutExpired:
        dest.write_text("Transcription timed out.\n", encoding="utf-8")
        return "transcription_failed"
    finally:
        if wav.exists():
            wav.unlink()


def write_index(manifest):
    policy = manifest.get("policy") or {}
    lines = [
        "# Task files",
        "",
        "Read this index first. Original files stay in this directory.",
        "For a video, use the listed frames and transcript.",
        "Do not ingest original video bytes into the model.",
        "",
        "## Policy",
        "",
        f"- Maximum media duration: {int(policy.get('max_duration_seconds', max_duration))} seconds",
        f"- Maximum frames per video: {policy.get('max_frames', max_frames)}",
        f"- Maximum frame width: {policy.get('max_frame_width', max_width)} pixels",
        f"- Per-file processing timeout: {policy.get('process_timeout_seconds', process_timeout)} seconds",
        f"- Task disk budget: {int(policy.get('disk_budget_bytes', disk_budget))} bytes",
        "",
        "## Files",
        "",
    ]
    for item in manifest.get("files") or []:
        dest = item.get("dest") or item.get("filename") or "file"
        lines.append(f"### {dest}")
        lines.append("")
        if item.get("id"):
            lines.append(f"- id: {item['id']}")
        lines.append(f"- original: attachments/{dest}")
        lines.append(f"- status: {item.get('status') or 'downloaded'}")
        if item.get("reason"):
            lines.append(f"- reason: {item['reason']}")
        if item.get("duration_seconds"):
            lines.append(f"- duration: {item['duration_seconds']:.1f}s")
        frames = item.get("frames") or []
        if frames:
            lines.append(f"- frames: attachments/{item.get('frames_dir')} ({len(frames)} stills)")
            for frame in frames:
                lines.append(f"  - {frame['timestamp_seconds']:.3f}s: attachments/{frame['path']}")
        if item.get("transcript"):
            lines.append(f"- transcript: attachments/{item['transcript']}")
        lines.append("")
    index_path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


manifest = load_manifest()
videos = []
for item in manifest.get("files") or []:
    dest_name = item.get("dest") or ""
    dest = attachments / dest_name
    if dest_name and dest.is_file() and looks_like_video(item, dest):
        videos.append((item, dest))

if not videos:
    write_index(manifest)
    save_manifest(manifest)
    print("No video files in task attachments.")
    sys.exit(0)

for item, dest in videos:
    if dir_size(attachments) > disk_budget:
        item["status"] = "failed"
        item["reason"] = "disk_budget_exceeded"
        print(f"{dest.name}: disk budget exceeded", file=sys.stderr)
        continue

    payload, reason = probe(dest)
    if payload is None:
        item["status"] = "failed"
        item["reason"] = reason
        print(f"{dest.name}: {reason}")
        continue

    format_name = ((payload.get("format") or {}).get("format_name") or "").lower()
    if any(name.strip() in {"png_pipe", "image2", "gif", "webp_pipe", "bmp_pipe"} for name in format_name.split(",")):
        item["status"] = "failed"
        item["reason"] = "undecodable"
        print(f"{dest.name}: undecodable")
        continue

    video_streams = [s for s in streams(payload, "video") if usable_video_stream(s)]
    audio_streams = streams(payload, "audio")
    if not video_streams:
        item["status"] = "failed"
        item["reason"] = "no_video_stream"
        print(f"{dest.name}: no video stream")
        continue

    duration = duration_seconds(payload)
    item["duration_seconds"] = duration
    item["has_audio"] = bool(audio_streams)
    if duration > max_duration:
        item["status"] = "failed"
        item["reason"] = "duration_exceeds_limit"
        print(f"{dest.name}: duration {duration:.1f}s exceeds {max_duration:.0f}s")
        continue

    stamps = unique_timestamps(duration)
    try:
        for stamp in scene_timestamps(dest, duration or 1.0):
            key = round(stamp * 4) / 4.0
            if all(abs(existing - stamp) > 0.2 for existing in stamps):
                stamps.append(stamp)
            if len(stamps) >= max_frames:
                break
    except subprocess.TimeoutExpired:
        pass
    stamps = sorted(stamps)[:max_frames]

    frames_dir_name = dest.name + ".frames"
    frames_dir = attachments / frames_dir_name
    if frames_dir.exists():
        shutil.rmtree(frames_dir)
    frames = []
    for stamp in stamps:
        filename = f"frame-{stamp:07.3f}.jpg".replace(":", "-")
        frame_path = frames_dir / filename
        if extract_frame(dest, stamp, frame_path):
            rel = f"{frames_dir_name}/{filename}"
            frames.append({"path": rel, "timestamp_seconds": round(stamp, 3)})

    if not frames:
        item["status"] = "failed"
        item["reason"] = "frame_extraction_failed"
        print(f"{dest.name}: frame extraction failed")
        continue

    item["frames_dir"] = frames_dir_name
    item["frames"] = frames

    if not audio_streams:
        item["status"] = "ready"
        item["reason"] = "no_audio"
        print(f"{dest.name}: {len(frames)} frames, no audio")
        continue

    transcript_name = dest.name + ".transcript.txt"
    transcript_path = attachments / transcript_name
    transcribe_reason = transcribe(dest, transcript_path, duration)
    item["transcript"] = transcript_name
    if transcribe_reason:
        item["status"] = "partial"
        item["reason"] = transcribe_reason
        print(f"{dest.name}: {len(frames)} frames, transcription failed")
        continue

    item["status"] = "ready"
    item["reason"] = ""
    print(f"{dest.name}: {len(frames)} frames, transcript ready")

write_index(manifest)
save_manifest(manifest)
PY

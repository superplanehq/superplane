#!/usr/bin/env bash
# Probe task videos and audio, extract bounded timestamped frames, and transcribe.
# Hosted videos are downloaded to a temp file, processed, and deleted.
# Image-only tasks write INDEX.md and exit without the media toolchain.
# Fail before the agent starts when video or audio is present and tools are missing.
set -euo pipefail

attachments="${SUPERPLANE_TASK_DIR:?}/attachments"
manifest="$attachments/manifest.json"
index="$attachments/INDEX.md"
WHISPER_MODEL="${WHISPER_MODEL:-/usr/local/share/whisper/ggml-tiny.bin}"

MAX_DURATION_SECONDS="${VIDEO_MAX_DURATION_SECONDS:-900}"
MAX_FRAMES="${VIDEO_MAX_FRAMES:-24}"
MAX_WIDTH="${VIDEO_MAX_FRAME_WIDTH:-1280}"
MAX_HEIGHT="${VIDEO_MAX_FRAME_HEIGHT:-1280}"
MAX_SOURCE_PIXELS="${VIDEO_MAX_SOURCE_PIXELS:-16777216}"
PROCESS_TIMEOUT="${VIDEO_PROCESS_TIMEOUT_SECONDS:-120}"
DISK_BUDGET_BYTES="${VIDEO_DISK_BUDGET_BYTES:-2147483648}"
MAX_TRANSCRIPT_ATTEMPTS="${VIDEO_MAX_TRANSCRIPT_ATTEMPTS:-3}"
HOSTED_MAX_DURATION_SECONDS="${HOSTED_VIDEO_MAX_DURATION_SECONDS:-300}"
HOSTED_MAX_BYTES="${HOSTED_VIDEO_MAX_BYTES:-268435456}"
HOSTED_DOWNLOAD_TIMEOUT="${HOSTED_VIDEO_DOWNLOAD_TIMEOUT_SECONDS:-180}"
HOSTED_METADATA_TIMEOUT="${HOSTED_VIDEO_METADATA_TIMEOUT_SECONDS:-30}"

if [ ! -d "$attachments" ]; then
  printf 'attachments directory is missing.\n' >&2
  exit 1
fi
if [ ! -f "$manifest" ]; then
  printf 'attachments/manifest.json is missing.\n' >&2
  exit 1
fi

export ATTACHMENTS_DIR="$attachments"
export MANIFEST_PATH="$manifest"
export INDEX_PATH="$index"
export WHISPER_MODEL
export MAX_DURATION_SECONDS MAX_FRAMES MAX_WIDTH MAX_HEIGHT MAX_SOURCE_PIXELS PROCESS_TIMEOUT DISK_BUDGET_BYTES MAX_TRANSCRIPT_ATTEMPTS
export HOSTED_MAX_DURATION_SECONDS HOSTED_MAX_BYTES HOSTED_DOWNLOAD_TIMEOUT HOSTED_METADATA_TIMEOUT

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
max_height = int(os.environ["MAX_HEIGHT"])
max_source_pixels = int(os.environ["MAX_SOURCE_PIXELS"])
process_timeout = int(os.environ["PROCESS_TIMEOUT"])
disk_budget = int(os.environ["DISK_BUDGET_BYTES"])
max_transcript_attempts = int(os.environ["MAX_TRANSCRIPT_ATTEMPTS"])
hosted_max_duration = float(os.environ["HOSTED_MAX_DURATION_SECONDS"])
hosted_max_bytes = int(os.environ["HOSTED_MAX_BYTES"])
hosted_download_timeout = int(os.environ["HOSTED_DOWNLOAD_TIMEOUT"])
hosted_metadata_timeout = int(os.environ["HOSTED_METADATA_TIMEOUT"])
task_dir = attachments.parent

VIDEO_TYPES = {
    "video/mp4",
    "video/webm",
    "video/quicktime",
    "video/ogg",
    "video/x-m4v",
    "video/x-matroska",
}
VIDEO_SUFFIXES = {".mp4", ".webm", ".mov", ".ogv", ".m4v", ".mkv"}
AUDIO_TYPES = {
    "audio/mpeg",
    "audio/mp4",
    "audio/wav",
    "audio/webm",
    "audio/ogg",
}
AUDIO_SUFFIXES = {".mp3", ".m4a", ".wav", ".oga", ".ogg"}
IMAGE_TYPES = {"image/png", "image/jpeg", "image/gif", "image/webp"}
IMAGE_SUFFIXES = {".png", ".jpg", ".jpeg", ".gif", ".webp"}


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


def content_type_of(item) -> str:
    return (item.get("content_type") or "").split(";")[0].strip().lower()


def looks_like_audio(item, dest: Path) -> bool:
    content_type = content_type_of(item)
    if content_type in AUDIO_TYPES:
        return True
    if content_type.startswith("video/"):
        return False
    return dest.suffix.lower() in AUDIO_SUFFIXES


def looks_like_video(item, dest: Path) -> bool:
    if looks_like_audio(item, dest):
        return False
    content_type = content_type_of(item)
    if content_type in VIDEO_TYPES:
        return True
    return dest.suffix.lower() in VIDEO_SUFFIXES


def looks_like_image(item, dest: Path) -> bool:
    content_type = content_type_of(item)
    if content_type in IMAGE_TYPES:
        return True
    return dest.suffix.lower() in IMAGE_SUFFIXES


def require_media_toolchain(extra_cmds=()):
    missing = 0
    for cmd in ("ffmpeg", "ffprobe", "whisper-cli", "python3", *extra_cmds):
        if shutil.which(cmd) is None:
            print(
                f"{cmd} is required for video and audio task files. "
                "Install the runner media toolchain and rebuild this image.",
                file=sys.stderr,
            )
            missing = 1
    model = Path(whisper_model)
    if not model.is_file() or model.stat().st_size == 0:
        print(
            f"WHISPER_MODEL ({whisper_model}) is missing. Bake ggml-tiny.bin into the runner image. "
            "Do not download models during a task.",
            file=sys.stderr,
        )
        missing = 1
    if missing:
        sys.exit(1)


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


def source_dimensions_allowed(stream) -> bool:
    width = int(stream.get("width") or 0)
    height = int(stream.get("height") or 0)
    return width <= max_source_pixels // height


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
        "-vf", f"scale=w='min({max_width},iw)':h='min({max_height},ih)':force_original_aspect_ratio=decrease",
        str(dest),
    ], timeout=30)
    return result.returncode == 0 and dest.is_file() and dest.stat().st_size > 0


def transcribe(path: Path, dest: Path, duration: float) -> str:
    wav = dest.with_suffix(".wav")
    try:
        cap = min(duration if duration > 0 else max_duration, max_duration)
        if dir_size(attachments) + math.ceil(cap * 32000) > disk_budget:
            return "disk_budget_exceeded"
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
        if dir_size(attachments) > disk_budget:
            dest.unlink(missing_ok=True)
            return "disk_budget_exceeded"
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
        "Read this index first.",
        "Uploaded files stay in this directory.",
        "A hosted video is not stored. Use its frames and transcript.",
        "For a video, use the listed frames and transcript.",
        "For audio, use the listed transcript.",
        "For an image, open the listed original path.",
        "Do not ingest original video or audio bytes.",
        "Do not fetch a hosted video page URL.",
        "",
        "## Policy",
        "",
        f"- Maximum media duration: {int(policy.get('max_duration_seconds', max_duration))} seconds",
        f"- Maximum hosted video duration: {int(policy.get('hosted_video_max_duration_seconds', hosted_max_duration))} seconds",
        f"- Maximum frames per video: {policy.get('max_frames', max_frames)}",
        f"- Maximum frame width: {policy.get('max_frame_width', max_width)} pixels",
        f"- Maximum frame height: {policy.get('max_frame_height', max_height)} pixels",
        f"- Maximum source pixels: {policy.get('max_source_pixels', max_source_pixels)}",
        f"- Per-file processing timeout: {policy.get('process_timeout_seconds', process_timeout)} seconds",
        f"- Task disk budget: {int(policy.get('disk_budget_bytes', disk_budget))} bytes",
        "",
        "## Files",
        "",
    ]
    for item in manifest.get("files") or []:
        dest = item.get("dest") or item.get("filename") or "file"
        dest_path = attachments / dest
        lines.append(f"### {dest}")
        lines.append("")
        if item.get("id"):
            lines.append(f"- id: {item['id']}")
        if item.get("kind") == "hosted_video":
            lines.append(f"- page: {item.get('url') or ''}")
        else:
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
        if looks_like_image(item, dest_path):
            lines.append(
                f"- image: $SUPERPLANE_TASK_DIR/attachments/{dest}"
            )
        lines.append("")
    index_path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def mark_ready_non_media(manifest, videos, audios):
    media_dests = {dest.name for _, dest in videos + audios}
    for item in manifest.get("files") or []:
        dest_name = item.get("dest") or ""
        dest = attachments / dest_name
        if dest_name in media_dests:
            continue
        if item.get("status") in {"failed", "partial", "ready"}:
            continue
        if dest.is_file():
            item["status"] = "ready"
            item["reason"] = ""


def is_partial(item) -> bool:
    return str(item.get("status") or "").strip().lower() == "partial"


def has_existing_frames(item) -> bool:
    frames = item.get("frames") or []
    return any((attachments / str(frame.get("path") or "")).is_file() for frame in frames)


def finish_transcript(item, dest, duration, frame_count=None):
    transcript_name = dest.name + ".transcript.txt"
    transcript_path = attachments / transcript_name
    transcribe_reason = transcribe(dest, transcript_path, duration)
    if transcript_path.is_file():
        item["transcript"] = transcript_name
    if transcribe_reason:
        attempts = int(item.get("transcript_attempts") or 0) + 1
        item["transcript_attempts"] = attempts
        item["reason"] = transcribe_reason
        if attempts >= max_transcript_attempts:
            item["status"] = "failed"
            if frame_count is None:
                print(f"{dest.name}: transcript failed after {attempts} attempts")
            else:
                print(f"{dest.name}: {frame_count} frames, transcription failed after {attempts} attempts")
            return
        item["status"] = "partial"
        if frame_count is None:
            print(f"{dest.name}: transcript failed")
        else:
            print(f"{dest.name}: {frame_count} frames, transcription failed")
        return
    item["status"] = "ready"
    item["reason"] = ""
    item["transcript_attempts"] = 0
    if frame_count is None:
        print(f"{dest.name}: transcript ready")
    else:
        print(f"{dest.name}: {frame_count} frames, transcript ready")


def process_video(item, dest, duration_limit=None):
    limit = max_duration if duration_limit is None else duration_limit
    if is_partial(item) and has_existing_frames(item):
        finish_transcript(item, dest, float(item.get("duration_seconds") or 0), len(item.get("frames") or []))
        return

    if dir_size(attachments) > disk_budget:
        item["status"] = "failed"
        item["reason"] = "disk_budget_exceeded"
        print(f"{dest.name}: disk budget exceeded", file=sys.stderr)
        return

    payload, reason = probe(dest)
    if payload is None:
        item["status"] = "failed"
        item["reason"] = reason
        print(f"{dest.name}: {reason}")
        return

    format_name = ((payload.get("format") or {}).get("format_name") or "").lower()
    if any(name.strip() in {"png_pipe", "image2", "gif", "webp_pipe", "bmp_pipe"} for name in format_name.split(",")):
        item["status"] = "failed"
        item["reason"] = "undecodable"
        print(f"{dest.name}: undecodable")
        return

    video_streams = [s for s in streams(payload, "video") if usable_video_stream(s)]
    audio_streams = streams(payload, "audio")
    if not video_streams:
        item["status"] = "failed"
        item["reason"] = "no_video_stream"
        print(f"{dest.name}: no video stream")
        return

    if any(not source_dimensions_allowed(stream) for stream in video_streams):
        item["status"] = "failed"
        item["reason"] = "dimensions_exceed_limit"
        print(f"{dest.name}: video dimensions exceed limit")
        return

    duration = duration_seconds(payload)
    item["duration_seconds"] = duration
    item["has_audio"] = bool(audio_streams)
    if duration > limit:
        item["status"] = "failed"
        item["reason"] = "duration_exceeds_limit"
        print(f"{dest.name}: duration {duration:.1f}s exceeds {limit:.0f}s")
        return

    stamps = unique_timestamps(duration)
    try:
        for stamp in scene_timestamps(dest, duration or 1.0):
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
    budget_exceeded = False
    for stamp in stamps:
        filename = f"frame-{stamp:07.3f}.jpg".replace(":", "-")
        frame_path = frames_dir / filename
        if extract_frame(dest, stamp, frame_path):
            if dir_size(attachments) > disk_budget:
                budget_exceeded = True
                shutil.rmtree(frames_dir)
                frames = []
                break
            rel = f"{frames_dir_name}/{filename}"
            frames.append({"path": rel, "timestamp_seconds": round(stamp, 3)})

    if budget_exceeded:
        item["status"] = "failed"
        item["reason"] = "disk_budget_exceeded"
        print(f"{dest.name}: disk budget exceeded", file=sys.stderr)
        return

    if not frames:
        item["status"] = "failed"
        item["reason"] = "frame_extraction_failed"
        print(f"{dest.name}: frame extraction failed")
        return

    item["frames_dir"] = frames_dir_name
    item["frames"] = frames

    if not audio_streams:
        item["status"] = "ready"
        item["reason"] = "no_audio"
        print(f"{dest.name}: {len(frames)} frames, no audio")
        return

    finish_transcript(item, dest, duration, len(frames))


def process_audio(item, dest):
    if is_partial(item):
        finish_transcript(item, dest, float(item.get("duration_seconds") or 0))
        return

    if dir_size(attachments) > disk_budget:
        item["status"] = "failed"
        item["reason"] = "disk_budget_exceeded"
        print(f"{dest.name}: disk budget exceeded", file=sys.stderr)
        return

    payload, reason = probe(dest)
    if payload is None:
        item["status"] = "failed"
        item["reason"] = reason
        print(f"{dest.name}: {reason}")
        return

    audio_streams = streams(payload, "audio")
    if not audio_streams:
        item["status"] = "failed"
        item["reason"] = "no_audio_stream"
        print(f"{dest.name}: no audio stream")
        return

    duration = duration_seconds(payload)
    item["duration_seconds"] = duration
    item["has_audio"] = True
    if duration > max_duration:
        item["status"] = "failed"
        item["reason"] = "duration_exceeds_limit"
        print(f"{dest.name}: duration {duration:.1f}s exceeds {max_duration:.0f}s")
        return

    finish_transcript(item, dest, duration)


PROCESSED_MEDIA_STATUSES = {"ready", "failed"}


def transcript_attempts_of(item) -> int:
    try:
        return int(item.get("transcript_attempts") or 0)
    except (TypeError, ValueError):
        return 0


def needs_media_processing(item) -> bool:
    status = str(item.get("status") or "").strip().lower()
    if status in PROCESSED_MEDIA_STATUSES:
        return False
    if status == "partial" and transcript_attempts_of(item) >= max_transcript_attempts:
        item["status"] = "failed"
        return False
    return True


def apply_hosted_policy(manifest):
    global hosted_max_duration, hosted_max_bytes, hosted_download_timeout
    policy = manifest.get("policy") or {}
    if policy.get("hosted_video_max_duration_seconds"):
        hosted_max_duration = float(policy["hosted_video_max_duration_seconds"])
    if policy.get("hosted_video_max_bytes"):
        hosted_max_bytes = int(policy["hosted_video_max_bytes"])
    if policy.get("hosted_video_download_timeout_seconds"):
        hosted_download_timeout = int(policy["hosted_video_download_timeout_seconds"])


def metadata_failure_reason(stderr: str) -> str:
    text = (stderr or "").lower()
    if any(token in text for token in ("private video", "sign in", "log in", "login required", "this video is private")):
        return "private_video"
    return "metadata_failed"


def hosted_is_live(meta) -> bool:
    if meta.get("is_live") is True:
        return True
    status = str(meta.get("live_status") or "").strip().lower()
    return status in {"is_live", "is_upcoming", "post_live"}


def hosted_duration(meta):
    raw = meta.get("duration")
    try:
        value = float(raw)
    except (TypeError, ValueError):
        return None
    if math.isnan(value) or value <= 0:
        return None
    return value


def dump_hosted_metadata(url: str):
    try:
        result = run([
            "yt-dlp", "--no-playlist", "--no-warnings", "--skip-download", "--dump-single-json", url,
        ], timeout=hosted_metadata_timeout)
    except subprocess.TimeoutExpired:
        return None, "metadata_failed"
    if result.returncode != 0:
        return None, metadata_failure_reason(result.stderr)
    try:
        payload = json.loads(result.stdout or "{}")
    except json.JSONDecodeError:
        return None, "metadata_failed"
    if not isinstance(payload, dict):
        return None, "metadata_failed"
    return payload, ""


def reject_hosted_metadata(item, meta) -> bool:
    availability = str(meta.get("availability") or "").strip().lower()
    if availability in {"private", "needs_auth", "subscriber_only", "premium_only"}:
        item["status"] = "failed"
        item["reason"] = "private_video"
        print(f"{item.get('dest')}: private video")
        return True
    if hosted_is_live(meta):
        item["status"] = "failed"
        item["reason"] = "live_stream"
        print(f"{item.get('dest')}: live stream")
        return True
    duration = hosted_duration(meta)
    if duration is None:
        item["status"] = "failed"
        item["reason"] = "duration_missing"
        print(f"{item.get('dest')}: duration missing")
        return True
    item["duration_seconds"] = duration
    if duration > hosted_max_duration:
        item["status"] = "failed"
        item["reason"] = "duration_exceeds_limit"
        print(f"{item.get('dest')}: duration {duration:.1f}s exceeds {hosted_max_duration:.0f}s")
        return True
    return False


def remove_hosted_download(path: Path):
    parent = path.parent
    if path.exists():
        path.unlink()
    if parent.name != ".hosted-videos" or not parent.exists():
        return
    for extra in parent.glob(path.name + "*"):
        if extra.is_file():
            extra.unlink()
    try:
        parent.rmdir()
    except OSError:
        pass


HOSTED_DOWNLOAD_SUFFIXES = {".mp4", ".webm", ".mkv", ".mov", ".m4v", ".ogv"}


def resolve_hosted_download(dest: Path):
    if dest.is_file() and dest.stat().st_size > 0:
        return dest
    preferred = dest.parent / (dest.name + ".mp4")
    if preferred.is_file() and preferred.stat().st_size > 0:
        return preferred
    if not dest.parent.exists():
        return None
    matches = []
    for path in dest.parent.glob(dest.name + ".*"):
        if not path.is_file() or path.stat().st_size == 0:
            continue
        if path.suffix.lower() not in HOSTED_DOWNLOAD_SUFFIXES:
            continue
        matches.append(path)
    if not matches:
        return None
    matches.sort(key=lambda path: (path.suffix.lower() != ".mp4", path.name))
    return matches[0]


def download_hosted_video(url: str, dest: Path):
    dest.parent.mkdir(parents=True, exist_ok=True)
    try:
        result = run([
            "yt-dlp",
            "--no-playlist",
            "--no-warnings",
            "--no-progress",
            "--no-part",
            "--max-filesize",
            str(hosted_max_bytes),
            "-f",
            "bv*[height<=720]+ba/b[height<=720]",
            "--merge-output-format",
            "mp4",
            "-o",
            str(dest),
            url,
        ], timeout=hosted_download_timeout)
    except subprocess.TimeoutExpired:
        return None, "download_failed"
    written = resolve_hosted_download(dest)
    if result.returncode != 0 or written is None:
        return None, "download_failed"
    if written.stat().st_size > hosted_max_bytes:
        return None, "download_too_large"
    return written, ""


def process_hosted_video(item):
    url = str(item.get("url") or "").strip()
    dest_name = str(item.get("dest") or "").strip()
    if not url.startswith("https://") or not dest_name or "/" in dest_name:
        item["status"] = "failed"
        item["reason"] = "metadata_failed"
        print(f"{dest_name or 'hosted video'}: invalid hosted video")
        return
    meta, reason = dump_hosted_metadata(url)
    if meta is None:
        item["status"] = "failed"
        item["reason"] = reason or "metadata_failed"
        print(f"{dest_name}: {item['reason']}")
        return
    if reject_hosted_metadata(item, meta):
        return
    require_media_toolchain(("yt-dlp",))
    tmp = task_dir / ".hosted-videos" / dest_name
    try:
        written, download_reason = download_hosted_video(url, tmp)
        if download_reason:
            item["status"] = "failed"
            item["reason"] = download_reason
            print(f"{dest_name}: {download_reason}")
            return
        if is_partial(item) and has_existing_frames(item):
            finish_transcript(item, written, float(item.get("duration_seconds") or 0), len(item.get("frames") or []))
            return
        process_video(item, written, duration_limit=hosted_max_duration)
    finally:
        remove_hosted_download(tmp)


manifest = load_manifest()
apply_hosted_policy(manifest)
videos = []
audios = []
hosted = []
for item in manifest.get("files") or []:
    if (item.get("kind") or "") == "hosted_video":
        if needs_media_processing(item):
            hosted.append(item)
        continue
    dest_name = item.get("dest") or ""
    dest = attachments / dest_name
    if not dest_name or not dest.is_file():
        continue
    if not needs_media_processing(item):
        continue
    if looks_like_video(item, dest):
        videos.append((item, dest))
    elif looks_like_audio(item, dest):
        audios.append((item, dest))

if hosted and shutil.which("yt-dlp") is None:
    print(
        "yt-dlp is required for hosted video links. Install the pinned yt-dlp in the runner image. "
        "Do not download yt-dlp during a task.",
        file=sys.stderr,
    )
    sys.exit(1)

if videos or audios:
    require_media_toolchain()
    for item, dest in videos:
        process_video(item, dest)
    for item, dest in audios:
        process_audio(item, dest)

for item in hosted:
    try:
        process_hosted_video(item)
    except subprocess.TimeoutExpired:
        if str(item.get("status") or "").strip().lower() not in {"failed", "ready", "partial"}:
            item["status"] = "failed"
            item["reason"] = "download_failed"
        print(f"{item.get('dest') or 'hosted video'}: timed out")

if not videos and not audios and not hosted:
    print("No video or audio files in task attachments.")

mark_ready_non_media(manifest, videos, audios)
write_index(manifest)
save_manifest(manifest)
PY

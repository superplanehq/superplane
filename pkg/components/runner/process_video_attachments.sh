#!/usr/bin/env bash
# Extract still frames from task videos so the agent can Read them.
# Claude Code and Codex do not ingest a 50 MB video. Frames stay local.
set -u
attachments="${SUPERPLANE_TASK_DIR:?}/attachments"
index="$attachments/INDEX.md"
if [ ! -d "$attachments" ]; then
  exit 0
fi

if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
  printf 'ffmpeg is not on PATH. Skip video frame extraction.\n'
  printf -- '- Frame extraction skipped. ffmpeg is not on this runner.\n' >>"$index"
  exit 0
fi

is_video() {
  ffprobe -v error -select_streams v:0 -show_entries stream=codec_type -of csv=p=0 "$1" 2>/dev/null | grep -q video
}

processed=0
shopt -s nullglob
for file in "$attachments"/*; do
  [ -f "$file" ] || continue
  case "$file" in
    *.md | *.txt | *.wav | *.jpg | *.jpeg | *.png) continue ;;
  esac
  if ! is_video "$file"; then
    continue
  fi

  base="$(basename "$file")"
  frames="$attachments/${base}.frames"
  mkdir -p "$frames"
  # Scene cuts first. Interval frames fill gaps for a quiet recording.
  ffmpeg -hide_banner -loglevel error -y -i "$file" \
    -vf "select='gt(scene,0.25)',scale='min(1280,iw)':-2" -vsync vfr -frames:v 24 \
    "$frames/scene-%03d.jpg" || true
  if ! ls "$frames"/scene-*.jpg >/dev/null 2>&1; then
    ffmpeg -hide_banner -loglevel error -y -i "$file" \
      -vf "fps=1/2,scale='min(1280,iw)':-2" -frames:v 24 \
      "$frames/frame-%03d.jpg" || true
  fi
  count="$(find "$frames" -type f -name '*.jpg' | wc -l | tr -d ' ')"
  printf 'Extracted %s frames from %s\n' "$count" "$base"
  printf -- '- %s frames: attachments/%s.frames/ (%s stills)\n' "$base" "$base" "$count" >>"$index"
  processed=$((processed + 1))
done

if [ "$processed" -eq 0 ]; then
  printf 'No video files in task attachments.\n'
fi
exit 0

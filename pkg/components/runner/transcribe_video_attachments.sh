#!/usr/bin/env bash
# Transcribe narration from task videos. Speech is extra context for the agent.
# The original video stays in attachments/. This step writes a sidecar .txt.
set -u
attachments="${SUPERPLANE_TASK_DIR:?}/attachments"
index="$attachments/INDEX.md"
if [ ! -d "$attachments" ]; then
  exit 0
fi

if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
  printf 'ffmpeg is not on PATH. Skip video transcription.\n'
  printf -- '- Transcription skipped. ffmpeg is not on this runner.\n' >>"$index"
  exit 0
fi

transcribe() {
  wav="$1"
  out="$2"
  if command -v whisper >/dev/null 2>&1; then
    whisper "$wav" --model tiny --output_format txt --output_dir "$(dirname "$out")" >/dev/null
    generated="${wav%.wav}.txt"
    if [ -f "$generated" ] && [ "$generated" != "$out" ]; then
      mv "$generated" "$out"
    fi
    return 0
  fi
  if command -v whisper-cli >/dev/null 2>&1; then
    whisper-cli -f "$wav" -otxt -of "${out%.txt}" >/dev/null
    return 0
  fi
  if command -v whisper.cpp >/dev/null 2>&1; then
    whisper.cpp -f "$wav" -otxt -of "${out%.txt}" >/dev/null
    return 0
  fi
  return 1
}

is_video() {
  ffprobe -v error -select_streams v:0 -show_entries stream=codec_type -of csv=p=0 "$1" 2>/dev/null | grep -q video
}

processed=0
missing_tool=0
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
  wav="$attachments/${base}.wav"
  transcript="$attachments/${base}.transcript.txt"
  ffmpeg -hide_banner -loglevel error -y -i "$file" -vn -ac 1 -ar 16000 -c:a pcm_s16le "$wav" || continue
  if transcribe "$wav" "$transcript"; then
    printf 'Transcribed %s\n' "$base"
    printf -- '- %s transcript: attachments/%s.transcript.txt\n' "$base" "$base" >>"$index"
  else
    printf 'No speech transcription tool on PATH for %s.\n' "$base"
    missing_tool=1
    printf 'No speech transcription tool on this runner.\n' >"$transcript"
    printf -- '- %s transcript skipped. whisper is not on this runner.\n' "$base" >>"$index"
  fi
  rm -f "$wav"
  processed=$((processed + 1))
done

if [ "$processed" -eq 0 ]; then
  printf 'No video files in task attachments.\n'
fi
if [ "$missing_tool" -eq 1 ]; then
  printf 'Install whisper or whisper.cpp on the runner to transcribe narration.\n'
fi
exit 0

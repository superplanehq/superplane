#!/usr/bin/env bash
set -euo pipefail

: "${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}"
SP="$SUPERPLANE_TASK_DIR"

prompt_file=${1:?prompt file required}
model=${2:-}

if [[ -f "$SP/workdir" ]]; then
  cd "$(cat "$SP/workdir")"
fi

PROMPT=$(cat "$prompt_file")

claude_bin=(claude)
if command -v stdbuf >/dev/null 2>&1; then
  claude_bin=(stdbuf -oL -eL claude)
fi

# Prefer plain terminal text in live logs (no Markdown chrome).
system_prompt='Write all assistant messages as plain terminal text. Do not use Markdown: no bold/italic markers, headings, links, tables, or fenced code blocks. Prefer plain paths, shell commands, and simple indentation.'

claude_args=(--bare -p --output-format stream-json --verbose --include-partial-messages)
claude_args+=(--permission-mode acceptEdits)
claude_args+=(--append-system-prompt "$system_prompt")
if [[ -n "$model" ]]; then
  claude_args+=(--model "$model")
fi
claude_args+=(--allowedTools Bash,Read,Edit,Write)
if [[ "$(cat "$SP/prompt_count")" -gt 0 ]]; then
  claude_args+=(--continue)
fi

"${claude_bin[@]}" "${claude_args[@]}" -- "$PROMPT" \
  | tee -a "$SP/stream.jsonl" \
  | node "$SP/format.js"

printf '%s\n' "$(($(cat "$SP/prompt_count") + 1))" >"$SP/prompt_count"
bash "$SP/write-result.sh" "$SP/stream.jsonl" "$SUPERPLANE_RESULT_FILE"

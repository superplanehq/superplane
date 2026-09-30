package runner

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	FactoryAgentName  = "SuperPlane Agent"
	FactoryAgentEmail = "superplaneagent@superplane.com"

	EnvGitAuthorName     = "GIT_AUTHOR_NAME"
	EnvGitAuthorEmail    = "GIT_AUTHOR_EMAIL"
	EnvGitCommitterName  = "GIT_COMMITTER_NAME"
	EnvGitCommitterEmail = "GIT_COMMITTER_EMAIL"

	FactoryGitWrapperPath = "bin/git"
	FactoryGitHookPath    = "git-hooks/prepare-commit-msg"

	// FactoryCommitIdentityPrompt tells the agent not to override the identity
	// the environment, wrapper, and hook already enforce.
	FactoryCommitIdentityPrompt = "Do not set git user.email, and do not add Co-authored-by or Signed-off-by trailers."
)

// AttachFactoryCommitIdentity forces SuperPlane Agent
// <superplaneagent@superplane.com> on factory-owned agent tasks.
// A canvas that is not a factory app is left unchanged.
func AttachFactoryCommitIdentity(
	ctx core.ExecutionContext,
	environment []BrokerEnvironmentVariable,
	files []BrokerTaskFile,
) ([]BrokerEnvironmentVariable, []BrokerTaskFile) {
	if !canvasBelongsToFactory(ctx) {
		return environment, files
	}

	environment = upsertEnvironment(environment, EnvGitAuthorName, FactoryAgentName)
	environment = upsertEnvironment(environment, EnvGitAuthorEmail, FactoryAgentEmail)
	environment = upsertEnvironment(environment, EnvGitCommitterName, FactoryAgentName)
	environment = upsertEnvironment(environment, EnvGitCommitterEmail, FactoryAgentEmail)
	files = upsertTaskFile(files, BrokerTaskFile{
		Path:    FactoryGitWrapperPath,
		Content: FactoryGitWrapperScript(),
		Mode:    "0755",
	})
	files = upsertTaskFile(files, BrokerTaskFile{
		Path:    FactoryGitHookPath,
		Content: FactoryPrepareCommitMessageHook(),
		Mode:    "0755",
	})
	return environment, files
}

func canvasBelongsToFactory(ctx core.ExecutionContext) bool {
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return false
	}
	canvasID, err := uuid.Parse(strings.TrimSpace(ctx.WorkflowID))
	if err != nil {
		return false
	}
	factoryID, err := models.FindFactoryIDForCanvas(database.DB(context.Background()), orgID, canvasID)
	if err != nil || factoryID == nil {
		return false
	}
	return true
}

func upsertEnvironment(environment []BrokerEnvironmentVariable, name, value string) []BrokerEnvironmentVariable {
	for i := range environment {
		if environment[i].Name == name {
			environment[i].Value = value
			return environment
		}
	}
	return append(environment, BrokerEnvironmentVariable{Name: name, Value: value})
}

func upsertTaskFile(files []BrokerTaskFile, file BrokerTaskFile) []BrokerTaskFile {
	for i := range files {
		if files[i].Path == file.Path {
			files[i] = file
			return files
		}
	}
	return append(files, file)
}

// FactoryRepoCommitSetup runs after clone, inside the repository.
// It sets the local identity and installs the trailer hook so a later
// factory reset cannot restore the old hook.
func FactoryRepoCommitSetup() string {
	return strings.Join([]string{
		`git config user.email "` + FactoryAgentEmail + `"`,
		`git config user.name "` + FactoryAgentName + `"`,
		`export GIT_AUTHOR_NAME="` + FactoryAgentName + `"`,
		`export GIT_AUTHOR_EMAIL="` + FactoryAgentEmail + `"`,
		`export GIT_COMMITTER_NAME="` + FactoryAgentName + `"`,
		`export GIT_COMMITTER_EMAIL="` + FactoryAgentEmail + `"`,
		"cat > .git/hooks/prepare-commit-msg <<'HOOK'",
		FactoryPrepareCommitMessageHook() + "HOOK",
		"chmod +x .git/hooks/prepare-commit-msg",
	}, "\n")
}

// FactoryPrepareCommitMessageHook drops other agent trailers from the
// trailer block and keeps one SuperPlane Agent sign-off. A quoted trailer
// line in the body stays. Human co-authors and the agent co-author stay.
func FactoryPrepareCommitMessageHook() string {
	return `#!/bin/sh
set -u

msg_file="${1:?}"
signoff='Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>'
tmp="${msg_file}.sp-identity"
coauthors_file="${msg_file}.sp-coauthors"
kept_file="${msg_file}.sp-kept"
block_file="${msg_file}.sp-block"
filtered_file="${msg_file}.sp-filtered"

email_of() {
  printf '%s\n' "$1" | sed -n 's/.*<\([^>]*\)>.*/\1/p' | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]'
}

is_other_agent_email() {
  case "$1" in
    agent@superplane.com|agent@superplane.ai|opencode@superplane.io)
      return 0
      ;;
  esac
  return 1
}

trailer_key() {
  printf '%s\n' "$1" | sed -n '1s/^\([A-Za-z0-9-][A-Za-z0-9-]*\)[ \t]*:.*/\1/p' | tr '[:upper:]' '[:lower:]'
}

is_continuation_line() {
  printf '%s\n' "$1" | grep -q '^[ 	][^ 	]'
}

keep_trailer() {
  key=$(trailer_key "$1")
  case "$key" in
    signed-off-by|co-authored-by) ;;
    *)
      return 0
      ;;
  esac
  if is_other_agent_email "$(email_of "$1")"; then
    return 1
  fi
  if [ "$key" = "signed-off-by" ]; then
    return 1
  fi
  if grep -qxF "$1" "$kept_file"; then
    return 1
  fi
  printf '%s\n' "$1" >> "$kept_file"
  return 0
}

filter_trailer_block() {
  block_in="$1"
  block_out="$2"
  : > "$block_out"
  pending=""
  pending_set=0
  while IFS= read -r line || [ -n "$line" ]; do
    if [ "$pending_set" -eq 1 ] && is_continuation_line "$line"; then
      pending="${pending}
${line}"
      continue
    fi
    if [ "$pending_set" -eq 1 ] && keep_trailer "$pending"; then
      printf '%s\n' "$pending" >> "$block_out"
    fi
    pending="$line"
    pending_set=1
  done < "$block_in"
  if [ "$pending_set" -eq 1 ] && keep_trailer "$pending"; then
    printf '%s\n' "$pending" >> "$block_out"
  fi
}

rm -f "$tmp" "$coauthors_file" "$kept_file" "$block_file" "$filtered_file"
trap 'rm -f "$tmp" "$coauthors_file" "$kept_file" "$block_file" "$filtered_file"' EXIT
: > "$kept_file"

bounds=$(awk '
function is_blank(line) { return line ~ /^[ \t]*$/ }
function is_divider(line) { return line ~ /^---$/ || line ~ /^---[ \t]/ }
function is_comment(line) { return line ~ /^#/ }
function is_trailer(line) { return line ~ /^[A-Za-z0-9-]+[ \t]*:/ }
function is_continuation(line) { return line ~ /^[ \t][^ \t]/ }
{
  lines[NR] = $0
}
END {
  n = NR
  limit = n + 1
  for (i = 1; i <= n; i++) {
    if (is_divider(lines[i])) {
      limit = i
      break
    }
  }
  end = limit - 1
  while (end >= 1 && is_blank(lines[end])) end--
  while (end >= 1 && (is_comment(lines[end]) || is_blank(lines[end]))) end--
  while (end >= 1 && is_blank(lines[end])) end--
  block_end = end
  block_start = end + 1
  while (block_end >= 1 && (is_trailer(lines[block_end]) || is_continuation(lines[block_end]))) {
    block_start = block_end
    block_end--
  }
  block_end = end
  if (block_start <= block_end && block_start > 1 && is_blank(lines[block_start - 1]) && is_trailer(lines[block_start])) {
    print block_start, block_end
  } else {
    print 0, 0
  }
}
' "$msg_file")
block_start=${bounds%% *}
block_end=${bounds#* }

: > "$tmp"
if [ "$block_start" -eq 0 ]; then
  cat "$msg_file" > "$tmp"
else
  if [ "$block_start" -gt 1 ]; then
    sed -n "1,$((block_start - 1))p" "$msg_file" > "$tmp"
  fi
  sed -n "${block_start},${block_end}p" "$msg_file" > "$block_file"
  filter_trailer_block "$block_file" "$filtered_file"
  cat "$filtered_file" >> "$tmp"
  line_count=$(awk 'END { print NR + 0 }' "$msg_file")
  if [ "$block_end" -lt "$line_count" ]; then
    sed -n "$((block_end + 1)),\$p" "$msg_file" >> "$tmp"
  fi
fi
mv "$tmp" "$msg_file"

printf '%s\n' "${COAUTHORS:-}" > "$coauthors_file"
while IFS= read -r trailer || [ -n "$trailer" ]; do
  [ -n "$trailer" ] || continue
  if is_other_agent_email "$(email_of "$trailer")"; then
    continue
  fi
  git interpret-trailers --in-place --if-exists addIfDifferent --trailer "$trailer" "$msg_file" || exit 1
done < "$coauthors_file"

git interpret-trailers --in-place --if-exists addIfDifferent \
  --trailer "$signoff" "$msg_file" || exit 1

exit 0
`
}

// FactoryGitWrapperScript shadows git on factory agent tasks. Commit and
// merge keep the caller's hook directory and replace only prepare-commit-msg.
// A disabled hooks path skips repository hooks. The identity hook still runs.
func FactoryGitWrapperScript() string {
	return `#!/bin/bash
set -euo pipefail

sh_quote() {
  local value="$1"
  value="${value//\'/\'\\\'\'}"
  printf "'%s'" "$value"
}

hooks_path_override() {
  local spec="$1"
  case "$spec" in
    core.hooksPath=*)
      printf '%s\n' "${spec#core.hooksPath=}"
      return 0
      ;;
    core.hooksPath)
      printf '\n'
      return 0
      ;;
  esac
  return 1
}

resolve_hooks_path() {
  local configured="$1"
  local work_tree=""
  if [[ "$configured" == "~" && -n "${HOME:-}" ]]; then
    configured="$HOME"
  elif [[ "$configured" == "~/"* && -n "${HOME:-}" ]]; then
    configured="${HOME}/${configured:2}"
  fi
  if [[ "$configured" != /* ]]; then
    work_tree="$("$real_git" "${global[@]}" rev-parse --show-toplevel 2>/dev/null || true)"
    work_tree="${work_tree%%$'\n'*}"
    if [[ -n "$work_tree" ]]; then
      configured="${work_tree}/${configured}"
    fi
  fi
  printf '%s\n' "$configured"
}

repository_hooks_path() {
  local configured="" hooks_path="" parent=""
  configured="$("$real_git" "${global[@]}" config --get core.hooksPath 2>/dev/null || true)"
  configured="${configured%%$'\n'*}"
  if [[ -n "$configured" ]]; then
    resolve_hooks_path "$configured"
    return 0
  fi

  hooks_path="$("$real_git" "${global[@]}" rev-parse --git-path hooks 2>/dev/null || true)"
  hooks_path="${hooks_path%%$'\n'*}"
  if [[ -z "$hooks_path" ]]; then
    return 0
  fi
  if [[ "$hooks_path" != /* ]]; then
    parent="$(dirname "$hooks_path")"
    if [[ ! -d "$parent" ]]; then
      return 0
    fi
    hooks_path="$(cd "$parent" && pwd)/$(basename "$hooks_path")"
  fi
  printf '%s\n' "$hooks_path"
}

forward_repository_hooks() {
  local source_dir="$1"
  local active_dir="$2"
  local hook name
  for hook in "$source_dir"/*; do
    [[ -e "$hook" ]] || continue
    [[ -f "$hook" && -x "$hook" ]] || continue
    name="$(basename "$hook")"
    case "$name" in
      prepare-commit-msg|*.sample) continue ;;
    esac
    {
      printf '%s\n' '#!/bin/sh'
      printf 'exec %s "$@"\n' "$(sh_quote "$hook")"
    } > "$active_dir/$name"
    chmod +x "$active_dir/$name"
  done
}

write_prepare_commit_msg() {
  local source_dir="$1"
  local active_dir="$2"
  local identity="$hooks_dir/prepare-commit-msg"
  local original=""
  local target="$active_dir/prepare-commit-msg"
  if [[ -n "$source_dir" && -x "$source_dir/prepare-commit-msg" ]]; then
    original="$source_dir/prepare-commit-msg"
    if [[ "$original" -ef "$identity" ]]; then
      original=""
    fi
  fi
  {
    printf '%s\n' '#!/bin/sh'
    printf '%s\n' 'set -u'
    if [[ -n "$original" ]]; then
      printf '%s "$@" || exit $?\n' "$(sh_quote "$original")"
    fi
    printf 'exec %s "$@"\n' "$(sh_quote "$identity")"
  } > "$target"
  chmod +x "$target"
}

task_dir="${SUPERPLANE_TASK_DIR:?}"
bin_dir="${task_dir}/bin"
hooks_dir="${task_dir}/git-hooks"
if [[ -d "$hooks_dir" ]]; then
  hooks_dir="$(cd "$hooks_dir" && pwd)"
fi

real_git=""
IFS=: read -r -a path_dirs <<< "$PATH"
for dir in "${path_dirs[@]}"; do
  if [[ "$dir" == "$bin_dir" ]]; then
    continue
  fi
  candidate="${dir}/git"
  if [[ -x "$candidate" && ! -d "$candidate" ]]; then
    if [[ "$candidate" -ef "$0" ]]; then
      continue
    fi
    real_git="$candidate"
    break
  fi
done

if [[ -z "$real_git" ]]; then
  echo "git: real git binary not found" >&2
  exit 127
fi

export GIT_AUTHOR_NAME="SuperPlane Agent"
export GIT_AUTHOR_EMAIL="superplaneagent@superplane.com"
export GIT_COMMITTER_NAME="SuperPlane Agent"
export GIT_COMMITTER_EMAIL="superplaneagent@superplane.com"

args=("$@")
global=()
caller_hooks_path=""
caller_set_hooks_path=0
i=0
while [[ $i -lt ${#args[@]} ]]; do
  arg="${args[$i]}"
  case "$arg" in
    -C|-c|--git-dir|--work-tree|--namespace|--exec-path|--super-prefix|--config-env)
      if [[ "$arg" == "-c" && $((i + 1)) -lt ${#args[@]} ]]; then
        next="${args[$((i + 1))]}"
        if hooks_value="$(hooks_path_override "$next")"; then
          caller_set_hooks_path=1
          caller_hooks_path="$hooks_value"
          i=$((i + 2))
          continue
        fi
      fi
      global+=("$arg")
      if [[ $((i + 1)) -lt ${#args[@]} ]]; then
        global+=("${args[$((i + 1))]}")
        i=$((i + 2))
        continue
      fi
      ;;
    --git-dir=*|--work-tree=*|--namespace=*|--exec-path=*|--super-prefix=*|--config-env=*)
      global+=("$arg")
      i=$((i + 1))
      ;;
    -c*)
      value="${arg#-c}"
      if hooks_value="$(hooks_path_override "$value")"; then
        caller_set_hooks_path=1
        caller_hooks_path="$hooks_value"
        i=$((i + 1))
        continue
      fi
      global+=("$arg")
      i=$((i + 1))
      ;;
    -C*)
      global+=("$arg")
      i=$((i + 1))
      ;;
    --paginate|--no-pager|--bare|--no-replace-objects|--literal-pathspecs|--glob-pathspecs|--noglob-pathspecs|--icase-pathspecs|--no-optional-locks|-p)
      global+=("$arg")
      i=$((i + 1))
      ;;
    -*)
      global+=("$arg")
      i=$((i + 1))
      ;;
    *)
      break
      ;;
  esac
done

if [[ $i -ge ${#args[@]} ]]; then
  exec "$real_git" "$@"
fi

subcommand="${args[$i]}"
i=$((i + 1))
case "$subcommand" in
  commit|merge) ;;
  *) exec "$real_git" "$@" ;;
esac

rest=()
skip_next=0
reset_author=0
has_reset_author=0
while [[ $i -lt ${#args[@]} ]]; do
  arg="${args[$i]}"
  if [[ $skip_next -eq 1 ]]; then
    skip_next=0
    i=$((i + 1))
    continue
  fi
  case "$arg" in
    --author)
      skip_next=1
      ;;
    --author=*)
      ;;
    --amend)
      reset_author=1
      rest+=("$arg")
      ;;
    --reset-author)
      has_reset_author=1
      rest+=("$arg")
      ;;
    -C|-c|--reuse-message|--reedit-message)
      if [[ "$subcommand" == "commit" ]]; then
        reset_author=1
      fi
      rest+=("$arg")
      ;;
    -C?*|--reuse-message=*|--reedit-message=*)
      if [[ "$subcommand" == "commit" ]]; then
        reset_author=1
      fi
      rest+=("$arg")
      ;;
    *)
      rest+=("$arg")
      ;;
  esac
  i=$((i + 1))
done

if [[ "$subcommand" == "commit" && $reset_author -eq 1 && $has_reset_author -eq 0 ]]; then
  rest+=("--reset-author")
fi

if [[ ! -x "$hooks_dir/prepare-commit-msg" ]]; then
  echo "git: factory prepare-commit-msg hook is missing" >&2
  exit 1
fi

active_hooks="$(mktemp -d "${task_dir}/git-hooks-active.XXXXXX")"
if [[ $caller_set_hooks_path -eq 1 ]]; then
  if [[ -n "$caller_hooks_path" ]]; then
    source_hooks="$(resolve_hooks_path "$caller_hooks_path")"
  else
    source_hooks=""
  fi
else
  source_hooks="$(repository_hooks_path)"
fi
if [[ -n "$source_hooks" && -d "$source_hooks" ]]; then
  source_hooks="$(cd "$source_hooks" && pwd)"
  if [[ "$source_hooks" == "$hooks_dir" ]]; then
    source_hooks=""
  else
    forward_repository_hooks "$source_hooks" "$active_hooks"
  fi
else
  source_hooks=""
fi
write_prepare_commit_msg "$source_hooks" "$active_hooks"

exec "$real_git" "${global[@]}" -c "core.hooksPath=${active_hooks}" "$subcommand" "${rest[@]}"
`
}

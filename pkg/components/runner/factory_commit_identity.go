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

// FactoryPrepareCommitMessageHook drops other agent trailers and keeps one
// SuperPlane Agent sign-off. Human co-authors and the agent co-author stay.
func FactoryPrepareCommitMessageHook() string {
	return `#!/bin/sh
set -u

msg_file="${1:?}"
signoff='Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>'
agent_email='superplaneagent@superplane.com'
tmp="${msg_file}.sp-identity"
coauthors_file="${msg_file}.sp-coauthors"

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

rm -f "$tmp" "$coauthors_file"
trap 'rm -f "$tmp" "$coauthors_file"' EXIT

: > "$tmp"
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    'Signed-off-by:'*|'Co-authored-by:'*)
      email=$(email_of "$line")
      if is_other_agent_email "$email"; then
        continue
      fi
      if [ "$email" = "$agent_email" ]; then
        case "$line" in
          'Signed-off-by:'*)
            continue
            ;;
        esac
      fi
      ;;
  esac
  printf '%s\n' "$line" >> "$tmp"
done < "$msg_file"

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
// merge use the task-local hook so a stored prepare-commit-msg cannot
// restore a second agent identity.
func FactoryGitWrapperScript() string {
	return `#!/bin/bash
set -euo pipefail

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
i=0
while [[ $i -lt ${#args[@]} ]]; do
  arg="${args[$i]}"
  case "$arg" in
    -C|-c|--git-dir|--work-tree|--namespace|--exec-path|--super-prefix|--config-env)
      if [[ "$arg" == "-c" && $((i + 1)) -lt ${#args[@]} ]]; then
        next="${args[$((i + 1))]}"
        if [[ "$next" == core.hooksPath || "$next" == core.hooksPath=* ]]; then
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
      if [[ "$value" == core.hooksPath || "$value" == core.hooksPath=* ]]; then
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

exec "$real_git" "${global[@]}" -c "core.hooksPath=${hooks_dir}" "$subcommand" "${rest[@]}"
`
}

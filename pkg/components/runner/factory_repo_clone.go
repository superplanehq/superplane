package runner

import "strings"

// FactoryResolvedBaseFile is the clone base later Implement steps read.
// The file holds the branch that was cloned, not the requested name when
// that name is missing.
const FactoryResolvedBaseFile = "resolved-base"

// FactoryRepoCloneCommand clones ${REPO_URL} into repo.
// It uses ${BASE:-main} when that branch exists, clones with no branch when
// the remote has no heads, and clones the remote default branch when the
// named branch is missing. Branch names are compared as fixed strings.
// It records the resolved base in $SUPERPLANE_TASK_DIR/resolved-base so
// later steps do not keep the missing name. A failed clone exits immediately.
// Bash steps run inside "{ ... } || status", which disables set -e, so this
// command checks each failure itself.
func FactoryRepoCloneCommand() string {
	return strings.Join([]string{
		`named_branch="${BASE:-main}"`,
		`if ! heads="$(git ls-remote --heads "${REPO_URL}")"; then`,
		`  echo "clone failed: could not list remote branches" >&2`,
		`  exit 1`,
		`fi`,
		``,
		`named_present=0`,
		`needle="refs/heads/${named_branch}"`,
		`while IFS= read -r line || [ -n "$line" ]; do`,
		`  ref="${line#*$'\t'}"`,
		`  if [ "$ref" = "$needle" ]; then`,
		`    named_present=1`,
		`    break`,
		`  fi`,
		`done < <(printf '%s\n' "$heads")`,
		``,
		`resolved_base="$named_branch"`,
		`if [ "$named_present" -eq 1 ]; then`,
		`  if ! git clone --depth 1 --branch "$named_branch" "${REPO_URL}" repo; then`,
		`    echo "clone failed" >&2`,
		`    exit 1`,
		`  fi`,
		`elif [ -z "$heads" ]; then`,
		`  if ! git clone --depth 1 "${REPO_URL}" repo; then`,
		`    echo "clone failed" >&2`,
		`    exit 1`,
		`  fi`,
		`else`,
		`  if ! symref="$(git ls-remote --symref "${REPO_URL}" HEAD)"; then`,
		`    echo "clone failed: could not read the remote default branch" >&2`,
		`    exit 1`,
		`  fi`,
		`  default_branch=""`,
		`  while IFS= read -r line || [ -n "$line" ]; do`,
		`    case "$line" in`,
		`      "ref: "*)`,
		`        rest="${line#ref: }"`,
		`        ref="${rest%%[[:space:]]*}"`,
		`        case "$ref" in`,
		`          refs/heads/*)`,
		`            default_branch="${ref#refs/heads/}"`,
		`            ;;`,
		`        esac`,
		`        break`,
		`        ;;`,
		`    esac`,
		`  done < <(printf '%s\n' "$symref")`,
		`  if [ -z "$default_branch" ]; then`,
		`    echo "clone failed: remote default branch is missing" >&2`,
		`    exit 1`,
		`  fi`,
		`  resolved_base="$default_branch"`,
		`  if ! git clone --depth 1 --branch "$default_branch" "${REPO_URL}" repo; then`,
		`    echo "clone failed" >&2`,
		`    exit 1`,
		`  fi`,
		`fi`,
		`if [ -z "${SUPERPLANE_TASK_DIR:-}" ]; then`,
		`  echo "clone failed: SUPERPLANE_TASK_DIR is required" >&2`,
		`  exit 1`,
		`fi`,
		`if ! printf '%s\n' "$resolved_base" > "$SUPERPLANE_TASK_DIR/` + FactoryResolvedBaseFile + `"; then`,
		`  echo "clone failed: could not record the resolved base branch" >&2`,
		`  exit 1`,
		`fi`,
	}, "\n")
}

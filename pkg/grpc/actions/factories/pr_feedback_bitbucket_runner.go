package factories

import "strings"

func bitbucketAPIAuthLines() []string {
	return []string{
		`if [ -n "${BITBUCKET_EMAIL:-}" ]; then`,
		`  AUTH_HEADER="Authorization: Basic $(printf '%s:%s' "${BITBUCKET_EMAIL}" "${BITBUCKET_TOKEN}" | base64 | tr -d '\n')"`,
		`else`,
		`  AUTH_HEADER="Authorization: Bearer ${BITBUCKET_TOKEN}"`,
		`fi`,
	}
}

func bitbucketPullRequestSourceLines() []string {
	return append(bitbucketAPIAuthLines(),
		`PR_JSON=$(curl -fsSL -H "${AUTH_HEADER}" "https://api.bitbucket.org/2.0/repositories/${REPO}/pullrequests/${PR_NUMBER}")`,
		`SOURCE_REPO=$(printf '%s' "${PR_JSON}" | jq -r '.source.repository.full_name // empty')`,
		`SOURCE_BRANCH=$(printf '%s' "${PR_JSON}" | jq -r '.source.branch.name // empty')`,
		`SOURCE_HASH=$(printf '%s' "${PR_JSON}" | jq -r '.source.commit.hash // empty')`,
		`if [ -z "${SOURCE_REPO}" ] || [ "${SOURCE_REPO}" = "null" ]; then`,
		`  SOURCE_REPO="${REPO}"`,
		`fi`,
		`if [ -z "${SOURCE_BRANCH}" ] || [ "${SOURCE_BRANCH}" = "null" ] || [ -z "${SOURCE_HASH}" ] || [ "${SOURCE_HASH}" = "null" ]; then`,
		`  echo "Could not resolve the pull request source repository." >&2`,
		`  exit 1`,
		`fi`,
		`if ! printf '%s' "${SOURCE_REPO}" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'; then`,
		`  echo "Invalid pull request source repository." >&2`,
		`  exit 1`,
		`fi`,
		`if ! git check-ref-format --branch "${SOURCE_BRANCH}" >/dev/null 2>&1; then`,
		`  echo "Invalid pull request source branch." >&2`,
		`  exit 1`,
		`fi`,
		`if ! printf '%s' "${SOURCE_HASH}" | grep -Eq '^[0-9a-fA-F]{7,64}$'; then`,
		`  echo "Invalid pull request source commit." >&2`,
		`  exit 1`,
		`fi`,
	)
}

// bitbucketValidateStoredSourceLines re-validates the source values after
// they are re-read from .superplane files in the push path.
func bitbucketValidateStoredSourceLines() []string {
	return []string{
		`if ! printf '%s' "${SOURCE_REPO}" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'; then`,
		`  echo "Invalid pull request source repository." >&2`,
		`  exit 1`,
		`fi`,
		`if ! git check-ref-format --branch "${SOURCE_BRANCH}" >/dev/null 2>&1; then`,
		`  echo "Invalid pull request source branch." >&2`,
		`  exit 1`,
		`fi`,
		`if ! printf '%s' "${SOURCE_HASH}" | grep -Eq '^[0-9a-fA-F]{7,64}$'; then`,
		`  echo "Invalid pull request source commit." >&2`,
		`  exit 1`,
		`fi`,
		`if ! printf '%s' "${CHECKED_OUT}" | grep -Eq '^[0-9a-fA-F]{7,64}$'; then`,
		`  echo "Invalid local source commit." >&2`,
		`  exit 1`,
		`fi`,
	}
}

// bitbucketCheckoutCommand clones the pull request source repository and
// checks out that source commit. A same-named branch on the destination
// repository is not a checkout target.
func bitbucketCheckoutCommand() string {
	lines := []string{"set -euo pipefail"}
	lines = append(lines, bitbucketPullRequestSourceLines()...)
	lines = append(lines,
		`if [ -n "${PR_REVISION:-}" ] && [ "${SOURCE_HASH}" != "${PR_REVISION}" ]; then`,
		`  echo "Remote pull request head changed. Stop without checking out." >&2`,
		`  exit 1`,
		`fi`,
		`mkdir -p .superplane`,
		`printf '%s' "${SOURCE_REPO}" > .superplane/source-repo`,
		`printf '%s' "${SOURCE_BRANCH}" > .superplane/source-branch`,
		`printf '%s' "${SOURCE_HASH}" > .superplane/source-hash`,
		`git clone "https://bitbucket.org/${SOURCE_REPO}.git" repo`,
		`cd repo`,
		`git fetch origin -- "${SOURCE_BRANCH}"`,
		`FETCHED=$(git rev-parse FETCH_HEAD)`,
		`if [ "${FETCHED}" != "${SOURCE_HASH}" ]; then`,
		`  echo "Source branch tip does not match the pull request commit. Stop without changing it." >&2`,
		`  exit 1`,
		`fi`,
		`git checkout -B -- "${SOURCE_BRANCH}" "${SOURCE_HASH}"`,
		`if [ "$(git rev-parse HEAD)" != "${SOURCE_HASH}" ]; then`,
		`  echo "Local checkout does not match the pull request source commit." >&2`,
		`  exit 1`,
		`fi`,
	)
	return strings.Join(lines, "\n")
}

// bitbucketCommitPushCommand pushes only when the local checkout is the
// source commit and the remote source commit has not moved.
func bitbucketCommitPushCommand(message string) string {
	lines := []string{"set -euo pipefail"}
	lines = append(lines, bitbucketPullRequestSourceLines()...)
	lines = append(lines,
		`CHECKED_OUT=$(cat ../.superplane/source-hash)`,
		`SOURCE_REPO=$(cat ../.superplane/source-repo)`,
		`SOURCE_BRANCH=$(cat ../.superplane/source-branch)`,
	)
	lines = append(lines, bitbucketValidateStoredSourceLines()...)
	lines = append(lines,
		`if [ "${SOURCE_HASH}" != "${CHECKED_OUT}" ]; then`,
		`  echo "Remote pull request head changed. Stop without pushing."`,
		`  exit 0`,
		`fi`,
		`if [ -n "${PR_REVISION:-}" ] && [ "${SOURCE_HASH}" != "${PR_REVISION}" ]; then`,
		`  echo "Remote pull request head changed. Stop without pushing."`,
		`  exit 0`,
		`fi`,
		`if ! git merge-base --is-ancestor "${CHECKED_OUT}" HEAD; then`,
		`  echo "Local checkout is not the pull request source commit. Stop without pushing." >&2`,
		`  exit 1`,
		`fi`,
		`git add -A`,
		`if ! git diff --cached --quiet; then`,
		`  git commit -s -m "`+message+`"`,
		`  git push -- "https://bitbucket.org/${SOURCE_REPO}.git" "HEAD:${SOURCE_BRANCH}"`,
		`fi`,
	)
	return strings.Join(lines, "\n")
}

package factories

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__PRFeedbackResolveAddressedThreadsCommand(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}

	const localThread = "PRRT_local"
	const foreignThread = "PRRT_foreign"
	listJSON := reviewThreadListJSON(localThread)

	t.Run("replies then resolves a thread on the current pull request", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			jsonl:    `{"id":"` + localThread + `","reply":"Added the ownership check."}` + "\n",
			listJSON: listJSON,
		})

		require.NoError(t, run.err, run.output)
		assert.Equal(t, []string{
			"reviewThreads",
			"addPullRequestReviewThreadReply:" + localThread,
			"resolveReviewThread:" + localThread,
		}, run.operations)
		assert.Contains(t, run.calls[0], "owner=acme")
		assert.Contains(t, run.calls[0], "name=app")
		assert.Contains(t, run.calls[0], "number=42")
		assert.Contains(t, strings.Join(run.calls, "\n"), "body=Added the ownership check.")
	})

	t.Run("skips a thread that is not on the current pull request", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			jsonl:    `{"id":"` + foreignThread + `","reply":"Should not apply."}` + "\n",
			listJSON: listJSON,
		})

		require.NoError(t, run.err, run.output)
		assert.Equal(t, []string{"reviewThreads"}, run.operations)
		assert.Contains(t, run.output, foreignThread)
	})

	t.Run("does not resolve after a failed reply", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			jsonl:    `{"id":"` + localThread + `","reply":"Added coverage."}` + "\n",
			listJSON: listJSON,
			extraEnv: []string{"GH_FAIL_REPLY_ID=" + localThread},
		})

		require.NoError(t, run.err, run.output)
		assert.Equal(t, []string{
			"reviewThreads",
			"addPullRequestReviewThreadReply:" + localThread,
		}, run.operations)
		assert.Contains(t, run.output, "failed to reply")
	})

	t.Run("skips invalid lines and still handles a valid thread", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			jsonl: strings.Join([]string{
				`not-json`,
				`{"id":"` + localThread + `"}`,
			}, "\n") + "\n",
			listJSON: listJSON,
		})

		require.NoError(t, run.err, run.output)
		assert.Equal(t, []string{
			"reviewThreads",
			"addPullRequestReviewThreadReply:" + localThread,
			"resolveReviewThread:" + localThread,
		}, run.operations)
		assert.Contains(t, strings.Join(run.calls, "\n"), "body=Addressed in the latest commit.")
		assert.Contains(t, run.output, "skipping invalid")
	})

	t.Run("does not call GitHub when the file is missing", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			listJSON: listJSON,
		})

		require.NoError(t, run.err, run.output)
		assert.Empty(t, run.operations)
	})

	t.Run("does not mutate threads when the current pull request list fails", func(t *testing.T) {
		run := runResolveAddressedThreadsCommand(t, resolveThreadsCommandOptions{
			jsonl:    `{"id":"` + localThread + `","reply":"Added coverage."}` + "\n",
			listJSON: listJSON,
			extraEnv: []string{"GH_FAIL_LIST=1"},
		})

		require.NoError(t, run.err, run.output)
		assert.Equal(t, []string{"reviewThreads"}, run.operations)
		assert.Contains(t, run.output, "could not list review threads")
	})
}

type resolveThreadsCommandOptions struct {
	jsonl    string
	listJSON string
	extraEnv []string
}

type resolveThreadsCommandRun struct {
	output     string
	calls      []string
	operations []string
	err        error
}

func runResolveAddressedThreadsCommand(t *testing.T, options resolveThreadsCommandOptions) resolveThreadsCommandRun {
	t.Helper()

	root := t.TempDir()
	repoDir := filepath.Join(root, "repo")
	binDir := filepath.Join(root, "bin")
	require.NoError(t, os.MkdirAll(repoDir, 0o755))
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	if options.jsonl != "" {
		require.NoError(t, os.WriteFile(filepath.Join(root, "addressed-review-threads.jsonl"), []byte(options.jsonl), 0o644))
	}

	listPath := filepath.Join(root, "threads.json")
	require.NoError(t, os.WriteFile(listPath, []byte(options.listJSON), 0o644))
	logPath := filepath.Join(root, "gh-calls.log")
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "gh"), []byte(ghResolveThreadsStub), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "jq"), []byte(jqResolveThreadsStub), 0o755))

	cmd := exec.Command("bash", "-c", prFeedbackResolveAddressedThreadsCommand())
	cmd.Dir = repoDir
	cmd.Env = append(append([]string{
		"PATH=" + binDir + ":" + os.Getenv("PATH"),
		"HOME=" + root,
		"REPO=acme/app",
		"PR_NUMBER=42",
		"GH_CALL_LOG=" + logPath,
		"GH_THREAD_LIST_JSON=" + listPath,
	}, options.extraEnv...), "GITHUB_TOKEN=")
	output, err := cmd.CombinedOutput()

	calls := readGhCallLog(t, logPath)
	return resolveThreadsCommandRun{
		output:     string(output),
		calls:      calls,
		operations: ghCallOperations(calls),
		err:        err,
	}
}

func readGhCallLog(t *testing.T, path string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func ghCallOperations(calls []string) []string {
	operations := make([]string, 0, len(calls))
	for _, call := range calls {
		id := graphqlFieldValue(call, "id")
		switch {
		case strings.Contains(call, "reviewThreads"):
			operations = append(operations, "reviewThreads")
		case strings.Contains(call, "addPullRequestReviewThreadReply"):
			operations = append(operations, "addPullRequestReviewThreadReply:"+id)
		case strings.Contains(call, "resolveReviewThread"):
			operations = append(operations, "resolveReviewThread:"+id)
		}
	}
	return operations
}

func graphqlFieldValue(call string, name string) string {
	prefix := name + "="
	for _, field := range strings.Fields(call) {
		if strings.HasPrefix(field, prefix) {
			return strings.TrimPrefix(field, prefix)
		}
	}
	return ""
}

func reviewThreadListJSON(ids ...string) string {
	nodes := make([]string, 0, len(ids))
	for _, id := range ids {
		nodes = append(nodes, `{"id":"`+id+`"}`)
	}
	return `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[` + strings.Join(nodes, ",") + `]}}}}}`
}

const ghResolveThreadsStub = `#!/usr/bin/env bash
set -euo pipefail
log="${GH_CALL_LOG:?}"
{
  printf '%s' "$1"
  shift
  for arg in "$@"; do
    printf ' %s' "$arg"
  done
  printf '\n'
} >> "$log"

id=""
prev=""
for arg in "$@"; do
  case "$prev" in
    -f|-F)
      case "$arg" in
        id=*) id="${arg#id=}" ;;
      esac
      ;;
  esac
  prev="$arg"
done

joined="$*"
if [[ "$joined" == *reviewThreads* ]]; then
  if [ "${GH_FAIL_LIST:-}" = "1" ]; then
    echo "list failed" >&2
    exit 1
  fi
  cat "${GH_THREAD_LIST_JSON:?}"
  exit 0
fi
if [[ "$joined" == *addPullRequestReviewThreadReply* ]]; then
  if [ -n "${GH_FAIL_REPLY_ID:-}" ] && [ "$id" = "${GH_FAIL_REPLY_ID}" ]; then
    echo "reply failed" >&2
    exit 1
  fi
  printf '%s\n' '{"data":{"addPullRequestReviewThreadReply":{"comment":{"id":"IC_1"}}}}'
  exit 0
fi
if [[ "$joined" == *resolveReviewThread* ]]; then
  if [ -n "${GH_FAIL_RESOLVE_ID:-}" ] && [ "$id" = "${GH_FAIL_RESOLVE_ID}" ]; then
    echo "resolve failed" >&2
    exit 1
  fi
  printf '%s\n' '{"data":{"resolveReviewThread":{"thread":{"isResolved":true}}}}'
  exit 0
fi
echo "unexpected gh invocation: $joined" >&2
exit 1
`

const jqResolveThreadsStub = `#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "-r" ]; then
  shift
fi
expr="${1:-}"
input=$(cat || true)
case "$expr" in
  '.id // empty')
    if ! printf '%s' "$input" | grep -q '^{'; then
      exit 1
    fi
    printf '%s' "$input" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
    ;;
  '.reply // empty')
    printf '%s' "$input" | sed -n 's/.*"reply"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
    ;;
  '.data.repository.pullRequest.reviewThreads.nodes // [] | .[].id')
    printf '%s' "$input" | grep -o '"id":"[^"]*"' | cut -d'"' -f4 || true
    ;;
  *)
    echo "unexpected jq expression: $expr" >&2
    exit 1
    ;;
esac
`

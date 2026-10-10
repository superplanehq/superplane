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

func TestBitbucketFeedbackAcceptsAbbreviatedCommits(t *testing.T) {
	remote := bareRepository(t)
	commitRemoteBranch(t, remote, "feedback")
	previous := strings.TrimSpace(runGit(t, remote, "rev-parse", "refs/heads/feedback"))
	work := t.TempDir()
	runGit(t, work, "clone", "--branch", "feedback", fileURL(remote), ".")
	require.NoError(t, os.WriteFile(filepath.Join(work, "README"), []byte("updated\n"), 0o644))
	runGit(t, work, "commit", "-am", "update")
	runGit(t, work, "push", "origin", "feedback")
	head := strings.TrimSpace(runGit(t, remote, "rev-parse", "refs/heads/feedback"))
	for _, test := range []struct {
		name, source, revision string
		wantCheckout           bool
	}{
		{"short source", head[:12], "", true},
		{"short revision", head, head[:12], true},
		{"full revision", head[:12], head, true},
		{"wrong source", "abcdef012345", "", false},
		{"wrong revision", head[:12], "abcdef012345", false},
		{"stale source", previous[:12], "", false},
		{"stale revision", head[:12], previous[:12], false},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			require.NoError(t, os.Mkdir(bin, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "curl"), []byte("#!/bin/sh\nprintf '{}'\n"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "jq"), []byte(`#!/bin/sh
cat >/dev/null
case "$2" in
  '.source.repository.full_name // empty') printf '%s' acme/widgets ;;
  '.source.branch.name // empty') printf '%s' feedback ;;
  '.source.commit.hash // empty') printf '%s' "$FIXTURE_HASH" ;;
  *) exit 1 ;;
esac
`), 0o755))
			config := filepath.Join(dir, "gitconfig")
			denyNetworkGit(t, config)
			runGit(t, dir, "config", "--file", config, "url."+fileURL(remote)+".insteadOf", "https://bitbucket.org/acme/widgets.git")
			env := []string{"PATH=" + bin + ":" + os.Getenv("PATH"), "GIT_CONFIG_GLOBAL=" + config, "GIT_CONFIG_NOSYSTEM=1",
				"GIT_TERMINAL_PROMPT=0", "FIXTURE_HASH=" + test.source, "REPO=acme/widgets", "PR_NUMBER=7", "BITBUCKET_TOKEN=test", "PR_REVISION=" + test.revision,
				"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com"}
			command := exec.Command("bash", "-c", bitbucketCheckoutCommand())
			command.Dir, command.Env = dir, env
			output, err := command.CombinedOutput()
			if !test.wantCheckout {
				require.Error(t, err, "%s", output)
				return
			}
			require.NoError(t, err, "%s", output)
			stored, err := os.ReadFile(filepath.Join(dir, ".superplane", "source-hash"))
			require.NoError(t, err)
			assert.Equal(t, head, string(stored))
			repo := filepath.Join(dir, "repo")
			assert.Equal(t, head, strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD")))
			require.NoError(t, os.WriteFile(filepath.Join(repo, "fix.txt"), []byte(test.name), 0o644))
			command = exec.Command("bash", "-c", bitbucketCommitPushCommand("fix: address feedback"))
			staleEnv := append([]string(nil), env...)
			for i, value := range staleEnv {
				if strings.HasPrefix(value, "FIXTURE_HASH=") {
					staleEnv[i] = "FIXTURE_HASH=" + previous[:12]
				}
			}
			command.Dir, command.Env = repo, staleEnv
			output, err = command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.Contains(t, string(output), "Stop without pushing")
			assert.Equal(t, head, strings.TrimSpace(runGit(t, remote, "rev-parse", "refs/heads/feedback")))
			command = exec.Command("bash", "-c", bitbucketCommitPushCommand("fix: address feedback"))
			command.Dir, command.Env = repo, env
			output, err = command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			pushed := strings.TrimSpace(runGit(t, remote, "rev-parse", "refs/heads/feedback"))
			assert.NotEqual(t, head, pushed)
			assert.Equal(t, strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD")), pushed)
			runGit(t, remote, "update-ref", "refs/heads/feedback", head)
		})
	}
}

func TestBitbucketCheckoutCommandChecksOutTheSourceBranch(t *testing.T) {
	command := bitbucketCheckoutCommand()

	assert.Contains(t, command, `git checkout -B "${SOURCE_BRANCH}" "${SOURCE_HASH}"`)
	assert.NotContains(t, command, `git checkout -B --`)
	assert.Contains(t, command, `git check-ref-format --branch "${SOURCE_BRANCH}"`)
}

func TestRefreshBitbucketRunnerStepsReplacesCheckoutAndPush(t *testing.T) {
	configuration := func() map[string]any {
		return map[string]any{"steps": []any{
			map[string]any{"name": "Set Up Git User", "command": "git config"},
			map[string]any{"name": "Checkout Pull Request", "command": "echo stale-checkout"},
			map[string]any{"name": "Address PR feedback", "type": "prompt", "prompt": "keep me"},
			map[string]any{"name": "Commit and Push", "command": "echo stale-push"},
		}}
	}

	discussion := configuration()
	refreshBitbucketRunnerSteps(discussion, false)
	byName := map[string]map[string]any{}
	for _, item := range discussion["steps"].([]any) {
		step := item.(map[string]any)
		byName[step["name"].(string)] = step
	}
	assert.Equal(t, bitbucketCheckoutCommand(), byName["Checkout Pull Request"]["command"])
	assert.Equal(t, bitbucketCommitPushCommand("fix: address PR #${PR_NUMBER} feedback"), byName["Commit and Push"]["command"])
	assert.Equal(t, "git config", byName["Set Up Git User"]["command"])
	assert.Equal(t, "keep me", byName["Address PR feedback"]["prompt"])

	checks := configuration()
	refreshBitbucketRunnerSteps(checks, true)
	for _, item := range checks["steps"].([]any) {
		step := item.(map[string]any)
		if step["name"] == "Commit and Push" {
			assert.Equal(t, bitbucketCommitPushCommand("fix: repair failing builds on PR #${PR_NUMBER}"), step["command"])
		}
	}

	refreshBitbucketRunnerSteps(map[string]any{}, false)
}

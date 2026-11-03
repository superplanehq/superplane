package factories

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestGitHubFeedbackPushesSignedCommitWithCleanWorkingTree(t *testing.T) {
	cases := []struct {
		name  string
		build func(prFeedbackBuildRequest) *yaml.Canvas
	}{
		{name: "discussion", build: buildDiscussionPRFeedbackCanvas},
		{name: "checks", build: buildChecksPRFeedbackCanvas},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			remote := bareRepository(t)
			commitRemoteBranch(t, remote, "feedback")
			work := t.TempDir()
			runGit(t, work, "clone", "--branch", "feedback", fileURL(remote), ".")
			base := strings.TrimSpace(runGit(t, work, "rev-parse", "HEAD"))
			canvas := testCase.build(prFeedbackBuildRequest{Repository: "acme/app"})
			runner := findSpecNode(t, canvas, prFeedbackRunnerNodeID)

			bin := filepath.Join(t.TempDir(), "bin")
			require.NoError(t, os.Mkdir(bin, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "curl"), []byte("#!/bin/sh\nprintf '{}'\n"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "jq"), []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' \"$REMOTE_HEAD_FIXTURE\"\n"), 0o755))
			env := []string{
				"PATH=" + bin + ":" + os.Getenv("PATH"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
				"REPO=acme/app", "PR_NUMBER=7", "GITHUB_TOKEN=test", "PR_REVISION=" + base,
				"REMOTE_HEAD_FIXTURE=" + base,
			}
			setup := exec.Command("bash", "-c", runnerStepCommand(t, runner, "Set Up DCO Signing"))
			setup.Dir, setup.Env = work, env
			output, err := setup.CombinedOutput()
			require.NoError(t, err, "%s", output)

			require.NoError(t, os.WriteFile(filepath.Join(work, "README"), []byte("signed\n"), 0o644))
			commit := exec.Command(gitBinary(t), "commit", "-am", "fix: address feedback")
			commit.Dir, commit.Env = work, env
			output, err = commit.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.Empty(t, strings.TrimSpace(runGit(t, work, "status", "--porcelain")))

			push := exec.Command("bash", "-c", runnerStepCommand(t, runner, "Commit and Push"))
			push.Dir, push.Env = work, env
			output, err = push.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.Equal(t, strings.TrimSpace(runGit(t, work, "rev-parse", "HEAD")),
				strings.TrimSpace(runGit(t, remote, "rev-parse", "refs/heads/feedback")))
		})
	}
}

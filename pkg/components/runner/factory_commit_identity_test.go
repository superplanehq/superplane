package runner_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAttachFactoryCommitIdentitySkipsCanvasWithoutFactory(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	original := []runner.BrokerEnvironmentVariable{{
		Name:  runner.EnvGitAuthorEmail,
		Value: "agent@superplane.com",
	}}

	environment, files := runner.AttachFactoryCommitIdentity(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, original, nil)

	assert.Equal(t, original, environment)
	assert.Empty(t, files)
}

func TestAttachFactoryCommitIdentitySkipsInvalidCanvasID(t *testing.T) {
	environment, files := runner.AttachFactoryCommitIdentity(core.ExecutionContext{
		OrganizationID: "not-a-uuid",
		WorkflowID:     "also-not",
	}, nil, nil)

	assert.Empty(t, environment)
	assert.Empty(t, files)
}

func TestAttachFactoryCommitIdentitySetsAuthorEnvironmentAndGitWrapper(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")

	environment, files := runner.AttachFactoryCommitIdentity(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, []runner.BrokerEnvironmentVariable{
		{Name: runner.EnvGitAuthorEmail, Value: "agent@superplane.com"},
		{Name: "KEEP", Value: "yes"},
	}, nil)

	assert.Equal(t, runner.FactoryAgentEmail, envValue(t, environment, runner.EnvGitAuthorEmail))
	assert.Equal(t, runner.FactoryAgentName, envValue(t, environment, runner.EnvGitAuthorName))
	assert.Equal(t, runner.FactoryAgentEmail, envValue(t, environment, runner.EnvGitCommitterEmail))
	assert.Equal(t, runner.FactoryAgentName, envValue(t, environment, runner.EnvGitCommitterName))
	assert.Equal(t, "yes", envValue(t, environment, "KEEP"))

	wrapper := taskFile(t, files, runner.FactoryGitWrapperPath)
	hook := taskFile(t, files, runner.FactoryGitHookPath)
	assert.Equal(t, "0755", wrapper.Mode)
	assert.Equal(t, "0755", hook.Mode)
	assert.Equal(t, runner.FactoryGitWrapperScript(), wrapper.Content)
	assert.Equal(t, runner.FactoryPrepareCommitMessageHook(), hook.Content)
	assert.Contains(t, hook.Content, "agent@superplane.com")
	assert.Contains(t, hook.Content, "agent@superplane.ai")
	assert.Contains(t, hook.Content, "opencode@superplane.io")
	assert.Contains(t, hook.Content, runner.FactoryAgentEmail)
}

func TestFactoryCommitHookDropsOtherAgentTrailers(t *testing.T) {
	taskDir := t.TempDir()
	hookPath := filepath.Join(taskDir, "prepare-commit-msg")
	require.NoError(t, os.WriteFile(hookPath, []byte(runner.FactoryPrepareCommitMessageHook()), 0o755))
	stubDir := filepath.Join(taskDir, "stub")
	require.NoError(t, os.MkdirAll(stubDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stubDir, "git"), []byte(interpretTrailersStub), 0o755))

	message := strings.Join([]string{
		"feat: remove Ask mode",
		"",
		"The body can mention agent@superplane.com.",
		"",
		"Signed-off-by: SuperPlane Agent <agent@superplane.com>",
		"Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>",
		"Co-authored-by: SuperPlane Agent <superplaneagent@superplane.com>",
		"Co-authored-by: André Calil <andre@superplane.com>",
		"Co-authored-by: SuperPlane Agent <agent@superplane.com>",
		"Co-authored-by: OpenCode <opencode@superplane.io>",
		"Signed-off-by: Agent <agent@superplane.ai>",
	}, "\n")
	messagePath := filepath.Join(taskDir, "COMMIT_EDITMSG")
	require.NoError(t, os.WriteFile(messagePath, []byte(message), 0o644))

	coauthors := strings.Join([]string{
		"Co-authored-by: André Calil <andre@superplane.com>",
		"Co-authored-by: Pat Example <pat@example.com>",
		"Co-authored-by: SuperPlane Agent <agent@superplane.com>",
	}, "\n")
	runHook := func() {
		cmd := exec.Command(hookPath, messagePath)
		cmd.Env = append(os.Environ(), "COAUTHORS="+coauthors, "PATH="+stubDir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	runHook()
	first := readText(t, messagePath)
	runHook()
	second := readText(t, messagePath)

	assert.Equal(t, first, second)
	assert.Contains(t, second, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>")
	assert.Equal(t, 1, strings.Count(second, "Signed-off-by:"))
	assert.Contains(t, second, "Co-authored-by: SuperPlane Agent <superplaneagent@superplane.com>")
	assert.Contains(t, second, "Co-authored-by: André Calil <andre@superplane.com>")
	assert.Contains(t, second, "Co-authored-by: Pat Example <pat@example.com>")
	assert.Contains(t, second, "The body can mention agent@superplane.com.")
	assert.NotContains(t, second, "<agent@superplane.com>")
	assert.NotContains(t, second, "agent@superplane.ai")
	assert.NotContains(t, second, "opencode@superplane.io")
	assert.Equal(t, 1, strings.Count(second, "Co-authored-by: André Calil <andre@superplane.com>"))
}

func TestFactoryGitWrapperForcesIdentityOnCommitAndMerge(t *testing.T) {
	taskDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "git-hooks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "bin", "git"), []byte(runner.FactoryGitWrapperScript()), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "git-hooks", "prepare-commit-msg"), []byte(runner.FactoryPrepareCommitMessageHook()), 0o755))

	stubDir := filepath.Join(taskDir, "stub")
	require.NoError(t, os.MkdirAll(stubDir, 0o755))
	recordPath := filepath.Join(taskDir, "git-record")
	stub := strings.ReplaceAll(recordingGitStub, "__RECORD__", recordPath)
	require.NoError(t, os.WriteFile(filepath.Join(stubDir, "git"), []byte(stub), 0o755))

	runWrapper := func(args ...string) string {
		require.NoError(t, os.RemoveAll(recordPath))
		cmd := exec.Command(filepath.Join(taskDir, "bin", "git"), args...)
		cmd.Env = append(withoutGitIdentity(os.Environ()),
			"SUPERPLANE_TASK_DIR="+taskDir,
			"PATH="+filepath.Join(taskDir, "bin")+":"+stubDir,
			"COAUTHORS=Co-authored-by: Pat Example <pat@example.com>",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
		return readText(t, recordPath)
	}

	commit := runWrapper("commit", "--no-verify", "--author", "Bad Agent <agent@superplane.com>", "-m", "feat: first")
	assert.Contains(t, commit, "GIT_AUTHOR_EMAIL=superplaneagent@superplane.com")
	assert.Contains(t, commit, "GIT_COMMITTER_EMAIL=superplaneagent@superplane.com")
	assert.Contains(t, commit, "core.hooksPath="+filepath.Join(taskDir, "git-hooks"))
	assert.Contains(t, commit, "--no-verify")
	assert.NotContains(t, commit, "--author")
	assert.Contains(t, commit, "exec="+filepath.Join(stubDir, "git"))

	amend := runWrapper("commit", "--amend", "--no-edit", "--no-verify", "--author=Bad Agent <agent@superplane.com>")
	assert.Contains(t, amend, "--amend")
	assert.Contains(t, amend, "--reset-author")
	assert.NotContains(t, amend, "--author")
	assert.Contains(t, amend, "core.hooksPath=")

	merge := runWrapper("merge", "--no-ff", "feature", "-m", "Merge branch feature")
	assert.Contains(t, merge, "\nmerge\n")
	assert.Contains(t, merge, "core.hooksPath=")
	assert.Contains(t, merge, "GIT_AUTHOR_NAME=SuperPlane Agent")
	assert.NotContains(t, merge, "--reset-author")

	status := runWrapper("status", "--short")
	assert.Contains(t, status, "\nstatus\n")
	assert.NotContains(t, status, "core.hooksPath")
}

const interpretTrailersStub = `#!/bin/sh
set -eu
trailer=""
file=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--trailer" ]; then
    trailer="$arg"
  fi
  prev="$arg"
  file="$arg"
done
if [ "$1" != "interpret-trailers" ] || [ -z "$trailer" ] || [ -z "$file" ]; then
  echo "unexpected git $*" >&2
  exit 1
fi
if grep -qxF "$trailer" "$file"; then
  exit 0
fi
printf '%s\n' "$trailer" >> "$file"
exit 0
`

const recordingGitStub = `#!/bin/sh
set -eu
{
  echo "exec=$0"
  echo "GIT_AUTHOR_NAME=${GIT_AUTHOR_NAME-}"
  echo "GIT_AUTHOR_EMAIL=${GIT_AUTHOR_EMAIL-}"
  echo "GIT_COMMITTER_NAME=${GIT_COMMITTER_NAME-}"
  echo "GIT_COMMITTER_EMAIL=${GIT_COMMITTER_EMAIL-}"
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} > "__RECORD__"
exit 0
`

func withoutGitIdentity(env []string) []string {
	out := make([]string, 0, len(env))
	for _, pair := range env {
		name, _, _ := strings.Cut(pair, "=")
		switch name {
		case "GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_AUTHOR_DATE",
			"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "GIT_COMMITTER_DATE":
			continue
		default:
			out = append(out, pair)
		}
	}
	return out
}

func envValue(t *testing.T, environment []runner.BrokerEnvironmentVariable, name string) string {
	t.Helper()
	for _, variable := range environment {
		if variable.Name == name {
			return variable.Value
		}
	}
	t.Fatalf("environment has no %s", name)
	return ""
}

func taskFile(t *testing.T, files []runner.BrokerTaskFile, path string) runner.BrokerTaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("task has no file %s", path)
	return runner.BrokerTaskFile{}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

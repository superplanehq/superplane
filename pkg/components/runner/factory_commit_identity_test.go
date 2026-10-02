package runner_test

import (
	"fmt"
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
		"The old commit used this trailer:",
		"Signed-off-by: SuperPlane Agent <agent@superplane.com>",
		"That line is an example, not a trailer.",
		"",
		"Signed-off-by: SuperPlane Agent <agent@superplane.com>",
		"Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>",
		"Signed-off-by: Jane Doe <jane@example.com>",
		"Signed-off-by: Jane Doe <jane@example.com>",
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
	assert.Equal(t, 1, strings.Count(second, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>"))
	assert.Contains(t, second, "Signed-off-by: Jane Doe <jane@example.com>")
	assert.Equal(t, 1, strings.Count(second, "Signed-off-by: Jane Doe <jane@example.com>"))
	assert.Equal(t, 3, strings.Count(second, "Signed-off-by:"))
	assert.Contains(t, second, "Co-authored-by: SuperPlane Agent <superplaneagent@superplane.com>")
	assert.Contains(t, second, "Co-authored-by: André Calil <andre@superplane.com>")
	assert.Contains(t, second, "Co-authored-by: Pat Example <pat@example.com>")
	assert.Contains(t, second, "The body can mention agent@superplane.com.")
	assert.Contains(t, second, "The old commit used this trailer:\nSigned-off-by: SuperPlane Agent <agent@superplane.com>\nThat line is an example, not a trailer.")
	assert.Equal(t, 1, strings.Count(second, "<agent@superplane.com>"))
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
			"PATH="+filepath.Join(taskDir, "bin")+string(os.PathListSeparator)+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"COAUTHORS=Co-authored-by: Pat Example <pat@example.com>",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
		return readText(t, recordPath)
	}

	commit := runWrapper("commit", "--no-verify", "--author", "Bad Agent <agent@superplane.com>", "-m", "feat: first")
	assert.Contains(t, commit, "GIT_AUTHOR_EMAIL=superplaneagent@superplane.com")
	assert.Contains(t, commit, "GIT_COMMITTER_EMAIL=superplaneagent@superplane.com")
	assertHooksPathUsesIdentityHook(t, taskDir, commit)
	assert.Contains(t, commit, "--no-verify")
	assert.NotContains(t, commit, "--author")
	assert.Contains(t, commit, "exec="+filepath.Join(stubDir, "git"))

	amend := runWrapper("commit", "--amend", "--no-edit", "--no-verify", "--author=Bad Agent <agent@superplane.com>")
	assert.Contains(t, amend, "--amend")
	assert.Contains(t, amend, "--reset-author")
	assert.NotContains(t, amend, "--author")
	assertHooksPathUsesIdentityHook(t, taskDir, amend)

	merge := runWrapper("merge", "--no-ff", "feature", "-m", "Merge branch feature")
	assert.Contains(t, merge, "\nmerge\n")
	assertHooksPathUsesIdentityHook(t, taskDir, merge)
	assert.Contains(t, merge, "GIT_AUTHOR_NAME=SuperPlane Agent")
	assert.NotContains(t, merge, "--reset-author")

	status := runWrapper("status", "--short")
	assert.Contains(t, status, "\nstatus\n")
	assert.NotContains(t, status, "core.hooksPath")

	disabled := runWrapper("-c", "core.hooksPath=/dev/null", "commit", "-m", "feat: disabled")
	assert.NotContains(t, disabled, "/dev/null")
	assertHooksPathUsesIdentityHook(t, taskDir, disabled)

	unchanged := runWrapper("-c", "core.hooksPath=/dev/null", "status", "--short")
	assert.Contains(t, unchanged, "core.hooksPath=/dev/null")
	assert.NotContains(t, unchanged, "git-hooks-active.")
}

func TestFactoryGitWrapperCommitUsesAgentIdentityAndRepositoryHooks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	t.Run("configured hooks path", func(t *testing.T) {
		assertFactoryCommitKeepsRepositoryHooks(t, true)
	})
	t.Run("default hooks directory", func(t *testing.T) {
		assertFactoryCommitKeepsRepositoryHooks(t, false)
	})
	t.Run("a failing repository hook stops the commit", func(t *testing.T) {
		taskDir := t.TempDir()
		repo := filepath.Join(taskDir, "repo")
		logPath := filepath.Join(taskDir, "hooks.log")
		wrapper := installFactoryGitWrapper(t, taskDir)
		env := factoryGitCommandEnv(taskDir)
		initRepository(t, repo, env, "")
		writeExec(t, filepath.Join(repo, ".git", "hooks", "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'pre-commit\\n' >> %q\nexit 1\n", logPath))
		require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("hello\n"), 0o644))
		runGit(t, repo, wrapper, env, "add", "README")

		cmd := exec.Command(wrapper, "commit", "-m", "feat: blocked")
		cmd.Dir = repo
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		require.Error(t, err, string(out))
		assert.Contains(t, readText(t, logPath), "pre-commit")
		head := exec.Command("git", "rev-parse", "--verify", "HEAD")
		head.Dir = repo
		head.Env = env
		headOut, headErr := head.CombinedOutput()
		require.Error(t, headErr, string(headOut))
	})
	t.Run("a disabled hooks path skips repository hooks", func(t *testing.T) {
		for _, args := range [][]string{
			{"-c", "core.hooksPath=/dev/null"},
			{"-c", "core.hooksPath="},
			{"-ccore.hooksPath=/dev/null"},
		} {
			t.Run(strings.Join(args, " "), func(t *testing.T) {
				assertDisabledHooksPathSkipsRepositoryHooks(t, args)
			})
		}
	})
	t.Run("a caller hooks path replaces the repository hooks", func(t *testing.T) {
		assertCallerHooksPathReplacesRepositoryHooks(t)
	})
}

func assertFactoryCommitKeepsRepositoryHooks(t *testing.T, configuredHooksPath bool) {
	t.Helper()
	taskDir := t.TempDir()
	repo := filepath.Join(taskDir, "repo")
	logPath := filepath.Join(taskDir, "hooks.log")
	wrapper := installFactoryGitWrapper(t, taskDir)
	env := factoryGitCommandEnv(taskDir)

	hooksDir := filepath.Join(repo, ".git", "hooks")
	configured := ""
	if configuredHooksPath {
		hooksDir = filepath.Join(repo, ".githooks")
		configured = ".githooks"
	}
	initRepository(t, repo, env, configured)
	if configuredHooksPath {
		writeExec(t, filepath.Join(repo, ".git", "hooks", "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'decoy\\n' >> %q\n", logPath))
	}
	writeExec(t, filepath.Join(hooksDir, "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'pre-commit\\n' >> %q\n", logPath))
	writeExec(t, filepath.Join(hooksDir, "commit-msg"), fmt.Sprintf(`#!/bin/sh
printf 'commit-msg\n' >> %q
if git interpret-trailers --parse "$1" | grep -q '<agent@superplane.com>\|<opencode@superplane.io>\|<agent@superplane.ai>'; then
  echo 'bad agent trailer remains' >&2
  exit 1
fi
if ! git interpret-trailers --parse "$1" | grep -q 'Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>'; then
  echo 'sign-off missing before commit-msg' >&2
  exit 1
fi
`, logPath))
	writeExec(t, filepath.Join(hooksDir, "prepare-commit-msg"), fmt.Sprintf(`#!/bin/sh
printf 'prepare-commit-msg\n' >> %q
tmp=$(mktemp)
awk 'NR==1 { print; print ""; print "Repository hook ran."; next } { print }' "$1" > "$tmp"
mv "$tmp" "$1"
printf 'Co-authored-by: SuperPlane Agent <agent@superplane.com>\n' >> "$1"
`, logPath))
	writeExec(t, filepath.Join(hooksDir, "pre-commit.sample"), fmt.Sprintf("#!/bin/sh\nprintf 'sample\\n' >> %q\n", logPath))

	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("hello\n"), 0o644))
	messagePath := filepath.Join(taskDir, "message.txt")
	require.NoError(t, os.WriteFile(messagePath, []byte(strings.Join([]string{
		"feat: one identity",
		"",
		"The old commit used this trailer:",
		"Signed-off-by: SuperPlane Agent <agent@superplane.com>",
		"That line is an example, not a trailer.",
		"",
		"The body can mention agent@superplane.com.",
		"",
		"Signed-off-by: SuperPlane Agent <agent@superplane.com>",
		"Signed-off-by: Jane Doe <jane@example.com>",
		"Co-authored-by: OpenCode <opencode@superplane.io>",
	}, "\n")), 0o644))
	runGit(t, repo, wrapper, env, "add", "README")
	runGit(t, repo, wrapper, env, "commit", "--author", "Bad Agent <agent@superplane.com>", "-F", messagePath)

	assert.Equal(t, "SuperPlane Agent", gitShow(t, repo, env, "%an"))
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ae"))
	assert.Equal(t, "SuperPlane Agent", gitShow(t, repo, env, "%cn"))
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ce"))
	assert.Equal(t, "feat: one identity", gitShow(t, repo, env, "%s"))

	body := gitShow(t, repo, env, "%B")
	assert.Contains(t, body, "The body can mention agent@superplane.com.")
	assert.Contains(t, body, "The old commit used this trailer:\nSigned-off-by: SuperPlane Agent <agent@superplane.com>\nThat line is an example, not a trailer.")
	assert.Contains(t, body, "Repository hook ran.")
	assert.Contains(t, body, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>")
	assert.Contains(t, body, "Signed-off-by: Jane Doe <jane@example.com>")
	assert.Contains(t, body, "Co-authored-by: Pat Example <pat@example.com>")
	assert.Equal(t, 1, strings.Count(body, "<agent@superplane.com>"))
	assert.NotContains(t, body, "opencode@superplane.io")
	assert.NotContains(t, body, "agent@superplane.ai")
	assert.Equal(t, 1, strings.Count(body, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>"))
	assert.Equal(t, 1, strings.Count(body, "Signed-off-by: Jane Doe <jane@example.com>"))
	assert.Equal(t, 1, strings.Count(body, "Co-authored-by: Pat Example <pat@example.com>"))

	trailers := gitTrailers(t, repo, env)
	assert.Contains(t, trailers, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>")
	assert.Contains(t, trailers, "Signed-off-by: Jane Doe <jane@example.com>")
	assert.Contains(t, trailers, "Co-authored-by: Pat Example <pat@example.com>")
	assert.NotContains(t, trailers, "Repository hook ran.")
	assert.NotContains(t, trailers, "<agent@superplane.com>")

	runGit(t, repo, wrapper, env, "commit", "--amend", "--no-edit")
	amended := gitShow(t, repo, env, "%B")
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ae"))
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ce"))
	assert.Equal(t, 1, strings.Count(amended, "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>"))
	assert.Equal(t, 1, strings.Count(amended, "Signed-off-by: Jane Doe <jane@example.com>"))
	assert.Equal(t, 1, strings.Count(amended, "Co-authored-by: Pat Example <pat@example.com>"))
	assert.Contains(t, amended, "Repository hook ran.")
	assert.Contains(t, amended, "That line is an example, not a trailer.")
	assert.Equal(t, 1, strings.Count(amended, "<agent@superplane.com>"))

	hookLog := readText(t, logPath)
	assert.Contains(t, hookLog, "pre-commit")
	assert.Contains(t, hookLog, "prepare-commit-msg")
	assert.Contains(t, hookLog, "commit-msg")
	assert.NotContains(t, hookLog, "decoy")
	assert.NotContains(t, hookLog, "sample")
}

func assertDisabledHooksPathSkipsRepositoryHooks(t *testing.T, prefix []string) {
	t.Helper()
	taskDir := t.TempDir()
	repo := filepath.Join(taskDir, "repo")
	logPath := filepath.Join(taskDir, "hooks.log")
	wrapper := installFactoryGitWrapper(t, taskDir)
	env := factoryGitCommandEnv(taskDir)
	initRepository(t, repo, env, "")
	writeExec(t, filepath.Join(repo, ".git", "hooks", "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'pre-commit\\n' >> %q\nexit 1\n", logPath))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("hello\n"), 0o644))
	runGit(t, repo, wrapper, env, "add", "README")

	args := append(append([]string{}, prefix...), "commit", "-m", "feat: hooks disabled")
	runGit(t, repo, wrapper, env, args...)
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ae"))
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ce"))
	assert.Contains(t, gitShow(t, repo, env, "%B"), "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>")
	_, err := os.Stat(logPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func assertCallerHooksPathReplacesRepositoryHooks(t *testing.T) {
	t.Helper()
	taskDir := t.TempDir()
	repo := filepath.Join(taskDir, "repo")
	logPath := filepath.Join(taskDir, "hooks.log")
	wrapper := installFactoryGitWrapper(t, taskDir)
	env := factoryGitCommandEnv(taskDir)
	initRepository(t, repo, env, "")
	writeExec(t, filepath.Join(repo, ".git", "hooks", "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'default\\n' >> %q\nexit 1\n", logPath))
	custom := filepath.Join(repo, ".githooks")
	writeExec(t, filepath.Join(custom, "pre-commit"), fmt.Sprintf("#!/bin/sh\nprintf 'custom\\n' >> %q\n", logPath))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("hello\n"), 0o644))
	runGit(t, repo, wrapper, env, "add", "README")
	runGit(t, repo, wrapper, env, "-c", "core.hooksPath=.githooks", "commit", "-m", "feat: custom hooks")

	assert.Equal(t, "custom\n", readText(t, logPath))
	assert.Equal(t, runner.FactoryAgentEmail, gitShow(t, repo, env, "%ae"))
	assert.Contains(t, gitShow(t, repo, env, "%B"), "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>")
}

func installFactoryGitWrapper(t *testing.T, taskDir string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "git-hooks"), 0o755))
	wrapper := filepath.Join(taskDir, "bin", "git")
	require.NoError(t, os.WriteFile(wrapper, []byte(runner.FactoryGitWrapperScript()), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "git-hooks", "prepare-commit-msg"), []byte(runner.FactoryPrepareCommitMessageHook()), 0o755))
	return wrapper
}

func factoryGitCommandEnv(taskDir string) []string {
	return append(withoutGitIdentity(os.Environ()),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"PATH="+filepath.Join(taskDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Bad Agent",
		"GIT_AUTHOR_EMAIL=agent@superplane.com",
		"GIT_COMMITTER_NAME=Bad Agent",
		"GIT_COMMITTER_EMAIL=agent@superplane.com",
		"COAUTHORS=Co-authored-by: Pat Example <pat@example.com>",
	)
}

func initRepository(t *testing.T, repo string, env []string, hooksPath string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(repo, 0o755))
	runGit(t, repo, "git", env, "init", "-b", "main")
	runGit(t, repo, "git", env, "config", "user.name", "Bad Agent")
	runGit(t, repo, "git", env, "config", "user.email", "agent@superplane.com")
	runGit(t, repo, "git", env, "config", "commit.gpgsign", "false")
	if hooksPath != "" {
		runGit(t, repo, "git", env, "config", "core.hooksPath", hooksPath)
	}
}

func runGit(t *testing.T, dir, binary string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}

func gitShow(t *testing.T, repo string, env []string, format string) string {
	t.Helper()
	return strings.TrimRight(runGit(t, repo, "git", env, "log", "-1", "--format="+format), "\n")
}

func gitTrailers(t *testing.T, repo string, env []string) string {
	t.Helper()
	message := gitShow(t, repo, env, "%B")
	cmd := exec.Command("git", "interpret-trailers", "--parse")
	cmd.Dir = repo
	cmd.Env = env
	cmd.Stdin = strings.NewReader(message)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func assertHooksPathUsesIdentityHook(t *testing.T, taskDir, record string) {
	t.Helper()
	hooksPath := hooksPathFromRecord(t, record)
	assert.NotEqual(t, filepath.Join(taskDir, "git-hooks"), hooksPath)
	assert.Contains(t, hooksPath, filepath.Join(taskDir, "git-hooks-active."))
	hook := readText(t, filepath.Join(hooksPath, "prepare-commit-msg"))
	assert.Contains(t, hook, filepath.Join(taskDir, "git-hooks", "prepare-commit-msg"))
}

func hooksPathFromRecord(t *testing.T, record string) string {
	t.Helper()
	lines := strings.Split(record, "\n")
	found := ""
	for i, line := range lines {
		if line != "-c" || i+1 >= len(lines) {
			continue
		}
		value, ok := strings.CutPrefix(lines[i+1], "core.hooksPath=")
		if ok {
			found = value
		}
	}
	if found == "" {
		t.Fatalf("record has no core.hooksPath: %s", record)
	}
	return found
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

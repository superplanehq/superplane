package factories

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestImplementCloneUsesRemoteDefaultWhenNamedBranchIsMissing(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, []string{"develop"}, "develop")
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), remote, "main")

	require.NoError(t, err, output)
	assert.Equal(t, "develop", templateCloneBranch(t, filepath.Join(dir, "repo")))
	assert.FileExists(t, filepath.Join(dir, "repo", ".git", "hooks", "prepare-commit-msg"))
}

func TestImplementCloneInstallsHookOnlyAfterEmptyClone(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, nil, "")
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), remote, "main")

	require.NoError(t, err, output)
	hook := filepath.Join(dir, "repo", ".git", "hooks", "prepare-commit-msg")
	info, statErr := os.Stat(hook)
	require.NoError(t, statErr, output)
	assert.NotZero(t, info.Mode()&0o111)
	body, readErr := os.ReadFile(hook)
	require.NoError(t, readErr)
	assert.Contains(t, string(body), runner.FactoryAgentEmail)
}

func TestImplementCloneUsesNamedBranchWhenItExists(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, []string{"develop", "main"}, "develop")
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), remote, "main")

	require.NoError(t, err, output)
	assert.Equal(t, "main", templateCloneBranch(t, filepath.Join(dir, "repo")))
}

func TestImplementCloneMatchesBranchNameAsFixedString(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, []string{"develop", "release.1", "releaseX1", "mainly"}, "develop")
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), remote, "release.1")

	require.NoError(t, err, output)
	assert.Equal(t, "release.1", templateCloneBranch(t, filepath.Join(dir, "repo")))

	prefixDir := t.TempDir()
	prefixOut, prefixErr := runTemplateClone(t, prefixDir, implementCloneCommand(t), remote, "main")
	require.NoError(t, prefixErr, prefixOut)
	assert.Equal(t, "develop", templateCloneBranch(t, filepath.Join(prefixDir, "repo")))
}

func TestImplementCloneStopsBeforeHookWhenCloneCommandFails(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, []string{"main"}, "main")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "repo"), []byte("not a directory"), 0o644))

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), remote, "main")

	require.Error(t, err, output)
	assert.Contains(t, output, "clone failed")
	assert.NotContains(t, output, "prepare-commit-msg")
}

func TestImplementCloneStopsBeforeHookWhenRemoteIsMissing(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, implementCloneCommand(t), filepath.Join(dir, "missing.git"), "main")

	require.Error(t, err, output)
	assert.Contains(t, output, "clone failed")
	assert.NotContains(t, output, "prepare-commit-msg")
	assert.NotContains(t, output, "cd: repo")
	_, statErr := os.Stat(filepath.Join(dir, "repo", ".git", "hooks", "prepare-commit-msg"))
	assert.Error(t, statErr)
}

func TestIntakeCloneUsesRemoteDefaultWhenNamedBranchIsMissing(t *testing.T) {
	requireGit(t)
	remote := templateCloneRemote(t, []string{"develop"}, "develop")
	dir := t.TempDir()

	output, err := runTemplateClone(t, dir, intakeAnalysisCloneCommand(), remote, "main")

	require.NoError(t, err, output)
	assert.Equal(t, "develop", templateCloneBranch(t, filepath.Join(dir, "repo")))
	_, statErr := os.Stat(filepath.Join(dir, "repo", ".git", "hooks", "prepare-commit-msg"))
	assert.Error(t, statErr)
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func implementCloneCommand(t *testing.T) string {
	t.Helper()
	result, err := materializeFactoryTemplate("line-implementation", factoryTemplateInput{
		appID:   "app-1",
		appName: "Implement",
	})
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	command, ok := implementationStep(t, findYAMLNode(t, canvas, "implementation-agent-no-issue"), "Clone Repo")["command"].(string)
	require.True(t, ok)
	return command
}

func runTemplateClone(t *testing.T, dir, command, remote, base string) (string, error) {
	t.Helper()
	requireGit(t)
	wrapped := "status=0\n{\n" + command + "\n} || status=$?\nexit \"$status\"\n"
	cmd := exec.Command("bash", "-c", wrapped)
	cmd.Dir = dir
	home := t.TempDir()
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"REPO_URL=" + remote,
		"BASE=" + base,
		"GITHUB_TOKEN=test-token",
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func templateCloneRemote(t *testing.T, branches []string, head string) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTemplateGit(t, "", "init", "--bare", remote)
	if len(branches) == 0 {
		return remote
	}
	work := t.TempDir()
	runTemplateGit(t, work, "init", "-b", branches[0])
	runTemplateGit(t, work, "config", "user.email", "test@example.com")
	runTemplateGit(t, work, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(work, "README"), []byte("hi\n"), 0o644))
	runTemplateGit(t, work, "add", "README")
	runTemplateGit(t, work, "commit", "-m", "init")
	runTemplateGit(t, work, "remote", "add", "origin", remote)
	runTemplateGit(t, work, "push", "origin", branches[0])
	for _, branch := range branches[1:] {
		runTemplateGit(t, work, "checkout", "-b", branch)
		runTemplateGit(t, work, "push", "origin", branch)
	}
	if head != "" {
		runTemplateGit(t, remote, "symbolic-ref", "HEAD", "refs/heads/"+head)
	}
	return remote
}

func runTemplateGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	home := t.TempDir()
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func templateCloneBranch(t *testing.T, repo string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return strings.TrimSpace(string(out))
}

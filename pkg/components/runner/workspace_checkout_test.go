package runner_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

func requireGitAndNode(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not installed")
	}
	return git
}

func initBareRepo(t *testing.T, branch string) string {
	t.Helper()
	git := requireGitAndNode(t)
	remote := filepath.Join(t.TempDir(), "origin.git")
	cmd := exec.Command(git, "init", "--bare", remote)
	require.NoError(t, cmd.Run())
	if branch != "" {
		work := t.TempDir()
		cmd = exec.Command(git, "clone", remote, work)
		require.NoError(t, cmd.Run())
		require.NoError(t, os.WriteFile(filepath.Join(work, "README"), []byte("seed\n"), 0o644))
		for _, args := range [][]string{
			{"-C", work, "add", "README"},
			{"-C", work, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "seed"},
			{"-C", work, "push", "origin", "HEAD:" + branch},
		} {
			cmd = exec.Command(git, args...)
			require.NoError(t, cmd.Run(), strings.Join(args, " "))
		}
		// Point the bare repo HEAD at the branch so default-branch clones work.
		cmd = exec.Command(git, "--git-dir="+remote, "symbolic-ref", "HEAD", "refs/heads/"+branch)
		require.NoError(t, cmd.Run())
	}
	return remote
}

func runCheckoutScript(t *testing.T, dir, repoURL, base, context string) (string, error) {
	t.Helper()
	return runCheckoutScriptFull(t, dir, repoURL, base, checkoutScriptOptions{context: context})
}

type checkoutScriptOptions struct {
	context     string
	commitSetup string
	home        string
	// probe runs after the helper is sourced (like the template Clone
	// step does), so tests can assert on the caller environment.
	probe string
}

func runCheckoutScriptFull(t *testing.T, dir, repoURL, base string, opts checkoutScriptOptions) (string, error) {
	t.Helper()
	requireGitAndNode(t)
	script := filepath.Join(dir, "checkout.sh")
	require.NoError(t, os.WriteFile(script, []byte(runner.WorkspaceCheckoutScript()), 0o755))
	if opts.context != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace_context.json"), []byte(opts.context), 0o644))
	}
	home := opts.home
	if home == "" {
		home = t.TempDir()
	}
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/bin:/bin:/usr/local/bin"
	}
	env := []string{
		"PATH=" + pathEnv,
		"HOME=" + home,
		"SUPERPLANE_TASK_DIR=" + dir,
		"REPO_URL=" + repoURL,
		"BASE=" + base,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
	if opts.commitSetup != "" {
		env = append(env, runner.WorkspaceCheckoutCommitSetupEnv+"="+opts.commitSetup)
	}
	var cmd *exec.Cmd
	if opts.probe != "" {
		cmd = exec.Command("bash", "-c", "source ./checkout.sh && "+opts.probe)
	} else {
		cmd = exec.Command("bash", script)
	}
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runGitQuiet runs git with an isolated HOME and no system config. An
// empty dir runs in a fresh temporary directory; an empty home gets one.
func runGitQuiet(t *testing.T, home, dir string, args ...string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	require.NoError(t, err)
	if home == "" {
		home = t.TempDir()
	}
	if dir == "" {
		dir = t.TempDir()
	}
	cmd := exec.Command(git, args...)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + home,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

func gitRepoConfig(t *testing.T, repoDir, key string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	require.NoError(t, err)
	cmd := exec.Command(git, "-C", repoDir, "config", "--get", key)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir(), "GIT_CONFIG_NOSYSTEM=1"}
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func contextJSON(t *testing.T, entries map[string]string) string {
	t.Helper()
	type entry struct {
		URL       string `json:"url"`
		Directory string `json:"directory"`
	}
	repos := make([]entry, 0, len(entries))
	for dir, url := range entries {
		repos = append(repos, entry{URL: url, Directory: dir})
	}
	payload, err := json.Marshal(map[string]any{"repositories": repos})
	require.NoError(t, err)
	return string(payload)
}

func TestWorkspaceCheckoutScriptClonesMainAndContext(t *testing.T) {
	main := initBareRepo(t, "main")
	extra := initBareRepo(t, "main")
	dir := t.TempDir()
	output, err := runCheckoutScript(t, dir, "file://"+main, "main", contextJSON(t, map[string]string{
		"context/acme/extra": "file://" + extra,
	}))
	require.NoError(t, err, output)
	assert.DirExists(t, filepath.Join(dir, "repo", ".git"))
	assert.DirExists(t, filepath.Join(dir, "context", "acme", "extra", ".git"))
	assert.Contains(t, output, "Checkout complete")
}

func TestWorkspaceCheckoutScriptReusesExistingCheckouts(t *testing.T) {
	main := initBareRepo(t, "main")
	dir := t.TempDir()
	_, err := runCheckoutScript(t, dir, "file://"+main, "main", "")
	require.NoError(t, err)
	output, err := runCheckoutScript(t, dir, "file://"+main, "main", "")
	require.NoError(t, err, output)
	assert.Contains(t, output, "already checked out")
}

func TestWorkspaceCheckoutScriptFailsOnMissingBranch(t *testing.T) {
	main := initBareRepo(t, "develop")
	dir := t.TempDir()
	_, err := runCheckoutScript(t, dir, "file://"+main, "main", "")
	require.Error(t, err)
	assert.NoDirExists(t, filepath.Join(dir, "repo", ".git"))
}

func TestWorkspaceCheckoutScriptSkipsUnsafeContextDirectory(t *testing.T) {
	main := initBareRepo(t, "main")
	extra := initBareRepo(t, "main")
	dir := t.TempDir()
	output, err := runCheckoutScript(t, dir, "file://"+main, "main", contextJSON(t, map[string]string{
		"../escape": "file://" + extra,
	}))
	require.NoError(t, err, output)
	assert.DirExists(t, filepath.Join(dir, "repo", ".git"))
	assert.Contains(t, output, "unsafe directory")
}

func TestWorkspaceCheckoutScriptCommitSetupDefaultsOff(t *testing.T) {
	main := initBareRepo(t, "main")
	dir := t.TempDir()
	output, err := runCheckoutScript(t, dir, "file://"+main, "main", "")
	require.NoError(t, err, output)
	assert.DirExists(t, filepath.Join(dir, "repo", ".git"))
	assert.NoFileExists(t, filepath.Join(dir, "repo", ".git", "hooks", "prepare-commit-msg"))
	assert.Empty(t, gitRepoConfig(t, filepath.Join(dir, "repo"), "user.email"))
}

func TestWorkspaceCheckoutScriptCommitSetupInstallsIdentity(t *testing.T) {
	main := initBareRepo(t, "main")
	extra := initBareRepo(t, "main")
	dir := t.TempDir()
	home := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+main, "main", checkoutScriptOptions{
		home:        home,
		commitSetup: "1",
		context: contextJSON(t, map[string]string{
			"context/acme/extra": "file://" + extra,
		}),
	})
	require.NoError(t, err, output)
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, runner.FactoryAgentEmail, gitRepoConfig(t, repo, "user.email"))
	assert.Equal(t, runner.FactoryAgentName, gitRepoConfig(t, repo, "user.name"))
	hook := filepath.Join(repo, ".git", "hooks", "prepare-commit-msg")
	info, err := os.Stat(hook)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o111, "hook is executable")
	content, err := os.ReadFile(hook)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Signed-off-by: SuperPlane Agent")
	assert.Equal(t, runner.FactoryAgentEmail, runGitQuiet(t, home, "", "config", "--global", "--get", "user.email"))
	// Context clones still land relative to the task directory: the
	// commit setup restores the working directory it borrows.
	assert.DirExists(t, filepath.Join(dir, "context", "acme", "extra", ".git"))
}

func TestWorkspaceCheckoutScriptCommitSetupExportsIdentityToCaller(t *testing.T) {
	main := initBareRepo(t, "main")
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	output, err := runCheckoutScriptFull(t, dir, "file://"+main, "main", checkoutScriptOptions{
		commitSetup: "1",
		probe:       `printf 'PROBE|%s|%s|%s' "$GIT_AUTHOR_EMAIL" "$GIT_COMMITTER_NAME" "$PWD"`,
	})
	require.NoError(t, err, output)
	assert.Contains(t, output, "PROBE|"+runner.FactoryAgentEmail+"|"+runner.FactoryAgentName+"|"+dir)
}

func TestWorkspaceCheckoutScriptCommitSetupCreatesBranchOnEmptyRemote(t *testing.T) {
	remote := initBareRepo(t, "")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{commitSetup: "1"})
	require.NoError(t, err, output)
	heads := runGitQuiet(t, "", "", "ls-remote", "--heads", "file://"+remote)
	assert.Contains(t, heads, "refs/heads/main")
	assert.Equal(t, "main", runGitQuiet(t, "", filepath.Join(dir, "repo"), "rev-parse", "--abbrev-ref", "HEAD"))
}

func TestWorkspaceCheckoutScriptCommitSetupSkipsBranchCreationOnReadonlyCheckout(t *testing.T) {
	remote := initBareRepo(t, "")
	dir := t.TempDir()
	output, err := runCheckoutScript(t, dir, "file://"+remote, "main", "")
	require.NoError(t, err, output)
	assert.DirExists(t, filepath.Join(dir, "repo", ".git"))
	heads := runGitQuiet(t, "", "", "ls-remote", "--heads", "file://"+remote)
	assert.Empty(t, strings.TrimSpace(heads), "read-only checkout must not push branches")
}

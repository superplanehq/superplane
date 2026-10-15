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
	requireGitAndNode(t)
	script := filepath.Join(dir, "checkout.sh")
	require.NoError(t, os.WriteFile(script, []byte(runner.WorkspaceCheckoutScript()), 0o755))
	if context != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace_context.json"), []byte(context), 0o644))
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = dir
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/bin:/bin:/usr/local/bin"
	}
	cmd.Env = []string{
		"PATH=" + pathEnv,
		"HOME=" + t.TempDir(),
		"SUPERPLANE_TASK_DIR=" + dir,
		"REPO_URL=" + repoURL,
		"BASE=" + base,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
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

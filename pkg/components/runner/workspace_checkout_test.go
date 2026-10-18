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
	// workDir overrides the process working directory. Production Clone
	// steps run at the task root, but prompt steps run inside repo/.
	workDir     string
	extraEnv    []string
	omitRepoURL bool
	setup       string
}

func runCheckoutScriptFull(t *testing.T, dir, repoURL, base string, opts checkoutScriptOptions) (string, error) {
	t.Helper()
	requireGitAndNode(t)
	// Stage the checkout command where production steps find it: on PATH
	// in the task bin directory.
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "checkout"), []byte(runner.WorkspaceCheckoutScript()), 0o755))
	if opts.context != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace_context.json"), []byte(opts.context), 0o644))
	}
	if opts.setup != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace_setup.json"), []byte(opts.setup), 0o644))
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
		"PATH=" + binDir + string(os.PathListSeparator) + pathEnv,
		"HOME=" + home,
		"SUPERPLANE_TASK_DIR=" + dir,
		"BASE=" + base,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
	if !opts.omitRepoURL {
		env = append(env, "REPO_URL="+repoURL)
	}
	env = append(env, opts.extraEnv...)
	if opts.commitSetup != "" {
		env = append(env, runner.WorkspaceCheckoutCommitSetupEnv+"="+opts.commitSetup)
	}
	cmd := exec.Command("bash", "-c", "checkout")
	cmd.Dir = opts.workDir
	if cmd.Dir == "" {
		cmd.Dir = dir
	}
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

func TestWorkspaceCheckoutScriptAnchorsToTaskDirFromSubdirectory(t *testing.T) {
	main := initBareRepo(t, "main")
	dir := t.TempDir()
	// Prompt steps run inside repo/, so invoke the helper from a nested
	// directory the way an agent following the checkout hint would.
	nested := filepath.Join(dir, "session")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	output, err := runCheckoutScriptFull(t, dir, "file://"+main, "main", checkoutScriptOptions{workDir: nested})
	require.NoError(t, err, output)
	assert.DirExists(t, filepath.Join(dir, "repo", ".git"))
	assert.NoDirExists(t, filepath.Join(nested, "repo"))
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

func TestWorkspaceCheckoutScriptCommitSetupCreatesBranchOnEmptyRemote(t *testing.T) {
	remote := initBareRepo(t, "")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{commitSetup: "1"})
	require.NoError(t, err, output)
	heads := runGitQuiet(t, "", "", "ls-remote", "--heads", "file://"+remote)
	assert.Contains(t, heads, "refs/heads/main")
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, "main", runGitQuiet(t, "", repo, "rev-parse", "--abbrev-ref", "HEAD"))
	// The initial commit carries the agent identity.
	assert.Equal(t, runner.FactoryAgentEmail+"|"+runner.FactoryAgentName,
		runGitQuiet(t, "", repo, "log", "-1", "--format=%ae|%an"))
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

func pushCommit(t *testing.T, remote, ref, filename, content string) string {
	t.Helper()
	git := requireGitAndNode(t)
	work := t.TempDir()
	cmd := exec.Command(git, "clone", remote, work)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NoError(t, os.WriteFile(filepath.Join(work, filename), []byte(content), 0o644))
	for _, args := range [][]string{
		{"-C", work, "add", filename},
		{"-C", work, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", filename},
	} {
		cmd = exec.Command(git, args...)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))
	}
	cmd = exec.Command(git, "-C", work, "rev-parse", "HEAD")
	sha, err := cmd.Output()
	require.NoError(t, err)
	cmd = exec.Command(git, "-C", work, "push", "origin", "HEAD:"+ref)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(sha))
}

func TestWorkspaceCheckoutScriptExitsWhenRepositoryIsUnset(t *testing.T) {
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "", "", checkoutScriptOptions{omitRepoURL: true})
	require.NoError(t, err, output)
	assert.Contains(t, output, "No repository to check out.")
	assert.NoDirExists(t, filepath.Join(dir, "repo"))
}

func TestWorkspaceCheckoutScriptDetachesAtPullRequestRevision(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/pull/9/head", "change", "pr\n")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		commitSetup: "1",
		extraEnv:    []string{"PR_NUMBER=9", "PR_REVISION=" + sha},
	})
	require.NoError(t, err, output)
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, sha, runGitQuiet(t, "", repo, "rev-parse", "HEAD"))
	assert.Equal(t, "HEAD", runGitQuiet(t, "", repo, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.NoFileExists(t, filepath.Join(repo, ".git", "hooks", "prepare-commit-msg"))
}

func TestWorkspaceCheckoutScriptKeepsGitHubBranchWhenBitbucketTokenIsPresent(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/pull/4/head", "feature", "branch\n")
	dir := t.TempDir()
	home := t.TempDir()
	git := requireGitAndNode(t)
	cmd := exec.Command(git, "config", "--file", filepath.Join(home, ".gitconfig"),
		"url."+remote+".insteadOf", "https://github.com/acme/widgets.git")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "GIT_CONFIG_NOSYSTEM=1"}
	require.NoError(t, cmd.Run())
	output, err := runCheckoutScriptFull(t, dir, "", "main", checkoutScriptOptions{
		home:        home,
		omitRepoURL: true,
		extraEnv: []string{
			"REPO=acme/widgets",
			"PR_NUMBER=4",
			"PR_HEAD=feature",
			"PR_REVISION=" + sha,
			"GITHUB_TOKEN=github-token",
			"BITBUCKET_TOKEN=bitbucket-token",
		},
	})
	require.NoError(t, err, output)
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, "feature", runGitQuiet(t, "", repo, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, sha, runGitQuiet(t, "", repo, "rev-parse", "HEAD"))
}

func TestWorkspaceCheckoutScriptUsesBitbucketWhenGitHubTokenIsAbsent(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/pull/4/head", "feature", "branch\n")
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "curl"), []byte("#!/bin/sh\necho bitbucket-api >&2\nexit 1\n"), 0o755))
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		extraEnv: []string{
			"REPO=acme/widgets",
			"PR_NUMBER=4",
			"PR_HEAD=feature",
			"PR_REVISION=" + sha,
			"BITBUCKET_TOKEN=bitbucket-token",
		},
	})
	require.Error(t, err)
	assert.Contains(t, output, "bitbucket-api")
	assert.NotContains(t, output, "Checkout complete")
	assert.NoDirExists(t, filepath.Join(dir, "repo", ".git"))
}

func TestWorkspaceCheckoutScriptChecksOutPullRequestBranch(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/pull/4/head", "feature", "branch\n")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		commitSetup: "1",
		extraEnv: []string{
			"PR_NUMBER=4",
			"PR_HEAD=feature",
			"PR_REVISION=" + sha,
		},
	})
	require.NoError(t, err, output)
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, "feature", runGitQuiet(t, "", repo, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, sha, runGitQuiet(t, "", repo, "rev-parse", "HEAD"))
	assert.NoFileExists(t, filepath.Join(repo, ".git", "hooks", "prepare-commit-msg"))
}

func TestWorkspaceCheckoutScriptDetachesBitbucketSourceRevision(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/heads/feature", "bb", "bb\n")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		extraEnv: []string{
			"REPO=acme/widgets",
			"SOURCE_REPO=acme/widgets",
			"PR_SOURCE=feature",
			"PR_REVISION=" + sha,
			"BITBUCKET_TOKEN=present",
		},
	})
	require.NoError(t, err, output)
	repo := filepath.Join(dir, "repo")
	assert.Equal(t, sha, runGitQuiet(t, "", repo, "rev-parse", "HEAD"))
	assert.Equal(t, "HEAD", runGitQuiet(t, "", repo, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.NoDirExists(t, filepath.Join(dir, ".superplane"))
	diff, err := os.ReadFile("/tmp/pr.diff")
	require.NoError(t, err)
	assert.Contains(t, string(diff), "bb")
}

func TestWorkspaceCheckoutScriptWritesGitHubPullRequestDiff(t *testing.T) {
	remote := initBareRepo(t, "main")
	sha := pushCommit(t, remote, "refs/pull/9/head", "change", "pr\n")
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$SUPERPLANE_TASK_DIR/gh-args\"\nprintf '%s\\n' 'diff --git a/change'\n"), 0o755))
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		extraEnv: []string{
			"REPO=acme/widgets",
			"PR_NUMBER=9",
			"PR_REVISION=" + sha,
		},
	})
	require.NoError(t, err, output)
	args, err := os.ReadFile(filepath.Join(dir, "gh-args"))
	require.NoError(t, err)
	assert.Contains(t, string(args), "/repos/acme/widgets/compare/main..."+sha)
	diff, err := os.ReadFile("/tmp/pr.diff")
	require.NoError(t, err)
	assert.Contains(t, string(diff), "diff --git a/change")
}

func TestWorkspaceCheckoutScriptRunsSetupCommandsInRepo(t *testing.T) {
	remote := initBareRepo(t, "main")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		setup: `{"commands":["pwd > setup-pwd","touch from-setup"]}`,
	})
	require.NoError(t, err, output)
	assert.FileExists(t, filepath.Join(dir, "repo", "from-setup"))
	assert.NoFileExists(t, filepath.Join(dir, "from-setup"))
	pwd, err := os.ReadFile(filepath.Join(dir, "repo", "setup-pwd"))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(strings.TrimSpace(string(pwd)), string(os.PathSeparator)+"repo"))
}

func TestWorkspaceCheckoutScriptFailsWhenSetupCommandFails(t *testing.T) {
	remote := initBareRepo(t, "main")
	dir := t.TempDir()
	output, err := runCheckoutScriptFull(t, dir, "file://"+remote, "main", checkoutScriptOptions{
		setup: `{"commands":["echo setup-failed >&2","exit 1"]}`,
	})
	require.Error(t, err)
	assert.Contains(t, output, "setup-failed")
}

func commandPtr(command string) *string {
	return &command
}

func TestInsertFactoryCheckoutCommandBeforeCanvasSteps(t *testing.T) {
	commands := []runner.BrokerCommand{
		{Name: "Auth", Kind: runner.LiveLogKindSetup, Command: "gh auth"},
		{Name: "Prompt", Kind: runner.LiveLogKindPrompt, Command: "do work"},
	}
	steps := []runner.AgentStep{
		{Name: "Score", Type: runner.AgentStepPrompt, Prompt: commandPtr("review the checkout")},
		{Name: "Diff", Type: runner.AgentStepBash, Command: commandPtr("git rev-parse HEAD > /tmp/pr.diff")},
	}
	out := runner.InsertFactoryCheckoutCommand(commands, steps, nil)
	require.Len(t, out, 3)
	assert.Equal(t, "Auth", out[0].Name)
	assert.Equal(t, "Checkout", out[1].Name)
	assert.Equal(t, runner.LiveLogKindBash, out[1].Kind)
	assert.Equal(t, runner.WrapAgentStepCommand("checkout"), out[1].Command)
	assert.Equal(t, runner.LiveLogText("Checkout"), out[1].Preview)
	assert.Equal(t, "Prompt", out[2].Name)
}

func TestInsertFactoryCheckoutCommandSkipsExistingClone(t *testing.T) {
	commands := []runner.BrokerCommand{{Name: "Prompt", Kind: runner.LiveLogKindPrompt, Command: "do work"}}
	cases := []runner.AgentStep{
		{Name: "Clone", Type: runner.AgentStepBash, Command: commandPtr("git clone https://example.com/repo.git repo")},
		{Name: "Helper", Type: runner.AgentStepBash, Command: commandPtr("set -euo pipefail\ncheckout\n")},
	}
	for _, step := range cases {
		out := runner.InsertFactoryCheckoutCommand(commands, []runner.AgentStep{step}, nil)
		assert.Equal(t, commands, out, step.Name)
	}
}

func TestApplyFactoryCheckoutKeepsInstructionsForSavedClone(t *testing.T) {
	files := checkoutHintFiles()
	steps := []runner.AgentStep{{
		Name:    "Clone",
		Type:    runner.AgentStepBash,
		Command: commandPtr("git clone https://example.com/repo.git repo"),
	}}
	commands := []runner.BrokerCommand{{Name: "Prompt", Kind: runner.LiveLogKindPrompt, Command: "do work"}}
	out := runner.ApplyFactoryCheckout(files, commands, steps)
	assert.Equal(t, commands, out)
	assert.Contains(t, files[0].Content, "run `checkout`")
	assert.NotContains(t, files[0].Content, "already ran")
}

func TestApplyFactoryCheckoutMarksInstructionsDone(t *testing.T) {
	files := checkoutHintFiles()
	commands := []runner.BrokerCommand{{Name: "Prompt", Kind: runner.LiveLogKindPrompt, Command: "do work"}}
	out := runner.ApplyFactoryCheckout(files, commands, nil)
	require.Len(t, out, 2)
	assert.Equal(t, "Checkout", out[0].Name)
	assert.Contains(t, files[0].Content, "Setup commands already ran in repo/ after checkout, in order: npm ci")
	assert.Contains(t, files[0].Content, "Context repositories are already checked out: acme/extra")
	assert.NotContains(t, files[0].Content, "run `checkout`")
}

func checkoutHintFiles() []runner.BrokerTaskFile {
	return []runner.BrokerTaskFile{
		{
			Path: "prompts/implement.txt",
			Content: "Implement.\nYou can use these resources." +
				"\nSetup commands (run in repo/ after clone, in order): npm ci" +
				"\nCheckout: run `checkout` to check out repo/ and context repositories. Extra repositories (already checked out when the automation runs it): acme/extra (context/acme/extra)\n",
		},
		{Path: runner.WorkspaceCheckoutScriptPath, Content: "#!/bin/bash\n"},
	}
}

func TestApplyFactoryCheckoutRequiresHelperFile(t *testing.T) {
	commands := []runner.BrokerCommand{{Name: "Prompt", Kind: runner.LiveLogKindPrompt, Command: "do work"}}
	out := runner.ApplyFactoryCheckout(nil, commands, nil)
	assert.Equal(t, commands, out)

	out = runner.ApplyFactoryCheckout(
		[]runner.BrokerTaskFile{{Path: runner.WorkspaceCheckoutScriptPath}},
		commands,
		nil,
	)
	require.Len(t, out, 2)
	assert.Equal(t, "Checkout", out[0].Name)
	assert.Equal(t, "Prompt", out[1].Name)
}

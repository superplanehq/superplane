package factories

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestCloneRepositoryChecksRemoteHeads(t *testing.T) {
	analysis := intakeAnalysisCloneCommand("")
	implementation := implementationCloneCommand(t)
	clone := cloneRepositoryCommand()

	assert.Contains(t, analysis, clone)
	assert.Contains(t, implementation, clone)
	rewriteAt := strings.Index(analysis, "insteadOf")
	headsAt := strings.Index(analysis, "git ls-remote --heads")
	require.GreaterOrEqual(t, rewriteAt, 0)
	require.GreaterOrEqual(t, headsAt, 0)
	assert.Less(t, rewriteAt, headsAt)

	scripts := []struct {
		name   string
		script string
	}{
		{name: "analysis", script: analysis},
		{name: "implementation", script: implementation},
	}
	for _, script := range scripts {
		t.Run(script.name, func(t *testing.T) {
			t.Run("empty remote clones without a branch", func(t *testing.T) {
				remote := bareRepository(t)
				result := runCloneScript(t, script.script, fileURL(remote), "main")

				require.NoError(t, result.err, result.stderr)
				assert.DirExists(t, filepath.Join(result.dir, "repo"))
				cloneLine := cloneInvocation(result.gitArgs)
				assert.NotContains(t, cloneLine, "--branch")
				assert.NotContains(t, cloneLine, "--depth")
				assert.NotContains(t, cloneLine, "github.com")
				heads := gitOutput(t, remote, "ls-remote", "--heads", fileURL(remote))
				if script.name == "implementation" {
					assert.Contains(t, heads, "refs/heads/main")
					assert.Equal(t, "0", revListCount(t, filepath.Join(result.dir, "repo"), "origin/main..HEAD"))
				} else {
					assert.Empty(t, strings.TrimSpace(heads))
				}
			})

			t.Run("named branch uses a depth-1 clone", func(t *testing.T) {
				remote := bareRepository(t)
				commitRemoteBranch(t, remote, "develop")
				result := runCloneScript(t, script.script, fileURL(remote), "develop")

				require.NoError(t, result.err, result.stderr)
				cloneLine := cloneInvocation(result.gitArgs)
				assert.Contains(t, cloneLine, "--depth 1")
				assert.Contains(t, cloneLine, "--branch develop")
				assert.NotContains(t, cloneLine, "github.com")
				assert.FileExists(t, filepath.Join(result.dir, "repo", ".git", "shallow"))
				head := gitOutput(t, filepath.Join(result.dir, "repo"), "rev-parse", "--abbrev-ref", "HEAD")
				assert.Equal(t, "develop", strings.TrimSpace(head))
				assert.Equal(t, "1", revListCount(t, remote, "develop"))
			})

			t.Run("missing named branch fails", func(t *testing.T) {
				remote := bareRepository(t)
				commitRemoteBranch(t, remote, "develop")
				result := runCloneScript(t, script.script, fileURL(remote), "main")

				require.Error(t, result.err)
				assert.Contains(t, result.stderr, "Remote branch main not found. The repository has other branches.")
				assert.Empty(t, cloneInvocation(result.gitArgs))
				assert.NoDirExists(t, filepath.Join(result.dir, "repo"))
			})
		})
	}
}

func TestEmptyRepositoryUsesBranchFromLostInitRace(t *testing.T) {
	remote := bareRepository(t)
	result := runCloneScriptWithPushShim(
		t,
		implementationCloneCommand(t),
		fileURL(remote),
		"develop",
		seedRemoteBranchScript(t, "develop"),
	)

	require.NoError(t, result.err, result.stderr)
	repo := filepath.Join(result.dir, "repo")
	assert.Equal(t, "winner", strings.TrimSpace(gitOutput(t, repo, "log", "-1", "--format=%s")))
	assert.Equal(t, "0", revListCount(t, repo, "origin/develop..HEAD"))
	content, err := os.ReadFile(filepath.Join(repo, "README"))
	require.NoError(t, err)
	assert.Equal(t, "winner\n", string(content))
	assert.Equal(t, "develop", strings.TrimSpace(gitOutput(t, repo, "rev-parse", "--abbrev-ref", "HEAD")))
}

func TestEmptyRepositoryFailsWhenInitPushFails(t *testing.T) {
	remote := bareRepository(t)
	result := runCloneScriptWithPushShim(
		t,
		implementationCloneCommand(t),
		fileURL(remote),
		"develop",
		"#!/bin/sh\nexit 1\n",
	)

	require.Error(t, result.err)
	assert.Contains(t, result.stderr, "Failed to create develop.")
	heads := gitOutput(t, remote, "ls-remote", "--heads", fileURL(remote))
	assert.Empty(t, strings.TrimSpace(heads))
	assert.NoFileExists(t, filepath.Join(result.dir, "repo", "README"))
}

func TestEmptyRepositoryCreatesNamedBaseBranch(t *testing.T) {
	remote := bareRepository(t)
	result := runCloneScript(t, implementationCloneCommand(t), fileURL(remote), "develop")

	require.NoError(t, result.err, result.stderr)
	heads := gitOutput(t, remote, "ls-remote", "--heads", fileURL(remote))
	assert.Contains(t, heads, "refs/heads/develop")
	assert.NotContains(t, heads, "refs/heads/main")
	assert.NotContains(t, heads, "refs/heads/master")

	repo := filepath.Join(result.dir, "repo")
	assert.Equal(t, "0", revListCount(t, repo, "origin/develop..HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("work\n"), 0o644))
	gitOutput(t, repo, "checkout", "-b", "feature")
	gitOutput(t, repo, "add", "README")
	gitOutput(t, repo, "commit", "-m", "feat: Add work")
	assert.Equal(t, "1", revListCount(t, repo, "origin/develop..HEAD"))
}

func TestCloneRepositoryUsesMainWhenBranchIsEmpty(t *testing.T) {
	remote := bareRepository(t)
	commitRemoteBranch(t, remote, "main")
	result := runCloneScript(t, cloneRepositoryCommand(), fileURL(remote), "")

	require.NoError(t, result.err, result.stderr)
	cloneLine := cloneInvocation(result.gitArgs)
	assert.Contains(t, cloneLine, "--depth 1")
	assert.Contains(t, cloneLine, "--branch main")
}

func implementationCloneCommand(t *testing.T) string {
	t.Helper()

	result, err := materializeFactoryTemplate("line-implementation", "", factoryTemplateInput{
		appID:   "app-1",
		appName: "Implement refunds",
		installParams: map[string]string{
			"appRepository": "acme/refunds",
			"defaultBranch": "develop",
		},
		integrations: map[string]factoryTemplateIntegration{
			"github": {id: "github-1", name: "acme-github"},
		},
		agent: &factoryTemplateAgent{
			component:                 "runnerOpenRouter",
			model:                     "anthropic/claude-sonnet-4-6",
			credentialSource:          "integration",
			credentialIntegrationName: "acme-openrouter",
		},
	})
	require.NoError(t, err)

	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	command, ok := implementationStep(t, findYAMLNode(t, canvas, "implementation-agent-no-issue"), "Clone Repo")["command"].(string)
	require.True(t, ok)
	return command
}

type cloneScriptResult struct {
	dir     string
	stderr  string
	gitArgs string
	err     error
}

func runCloneScript(t *testing.T, script, repoURL, base string) cloneScriptResult {
	t.Helper()
	return runCloneScriptWithPushShim(t, script, repoURL, base, "")
}

func runCloneScriptWithPushShim(t *testing.T, script, repoURL, base, pushShim string) cloneScriptResult {
	t.Helper()

	gitPath := gitBinary(t)
	dir := t.TempDir()
	home := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o755))
	logPath := filepath.Join(dir, "git-args")
	shimPath := ""
	if pushShim != "" {
		shimPath = filepath.Join(dir, "push-shim")
		require.NoError(t, os.WriteFile(shimPath, []byte(pushShim), 0o755))
	}
	wrapper := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"if [ \"$1\" = push ] && [ -n \"$PUSH_SHIM\" ] && [ ! -f \"$PUSH_SHIM_ONCE\" ]; then\n" +
		"  touch \"$PUSH_SHIM_ONCE\"\n" +
		"  \"$PUSH_SHIM\" || exit $?\n" +
		"fi\n" +
		"exec " + shellQuote(gitPath) + " \"$@\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "git"), []byte(wrapper), 0o755))

	configPath := filepath.Join(home, ".gitconfig")
	denyNetworkGit(t, configPath)

	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + binDir + ":/usr/bin:/bin",
		"HOME=" + home,
		"GIT_CONFIG_GLOBAL=" + configPath,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_REAL=" + gitPath,
		"LANG=C",
		"LC_ALL=C",
		"REPO_URL=" + repoURL,
		"BASE=" + base,
		"GITHUB_TOKEN=test-token",
		"PUSH_SHIM=" + shimPath,
		"PUSH_SHIM_ONCE=" + filepath.Join(dir, "push-shim-once"),
	}
	output, err := cmd.CombinedOutput()
	args, readErr := os.ReadFile(logPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	return cloneScriptResult{
		dir:     dir,
		stderr:  string(output),
		gitArgs: string(args),
		err:     err,
	}
}

func seedRemoteBranchScript(t *testing.T, branch string) string {
	t.Helper()
	return "#!/bin/sh\n" +
		"set -eu\n" +
		"work=$(mktemp -d)\n" +
		"\"$GIT_REAL\" clone \"$REPO_URL\" \"$work\"\n" +
		"cd \"$work\"\n" +
		"\"$GIT_REAL\" checkout -B " + shellQuote(branch) + "\n" +
		"printf '%s\\n' 'winner' > README\n" +
		"\"$GIT_REAL\" add README\n" +
		"\"$GIT_REAL\" -c user.email=winner@example.com -c user.name=Winner commit -m winner\n" +
		"\"$GIT_REAL\" push origin " + shellQuote("HEAD:refs/heads/"+branch) + "\n"
}

func bareRepository(t *testing.T) string {
	t.Helper()

	remote := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, t.TempDir(), "init", "--bare", remote)
	return remote
}

func commitRemoteBranch(t *testing.T, remote, branch string) {
	t.Helper()

	work := t.TempDir()
	runGit(t, work, "clone", fileURL(remote), ".")
	require.NoError(t, os.WriteFile(filepath.Join(work, "README"), []byte("seed\n"), 0o644))
	runGit(t, work, "add", "README")
	runGit(t, work, "commit", "-m", "seed")
	runGit(t, work, "push", "origin", "HEAD:"+branch)
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(gitBinary(t), args...)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + t.TempDir(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"LANG=C",
		"LC_ALL=C",
	}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s: %s", strings.Join(args, " "), output)
	return string(output)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return runGit(t, dir, args...)
}

func revListCount(t *testing.T, dir, rangeSpec string) string {
	t.Helper()
	return strings.TrimSpace(gitOutput(t, dir, "rev-list", "--count", rangeSpec))
}

func denyNetworkGit(t *testing.T, configPath string) {
	t.Helper()

	for _, protocol := range []string{"https", "http", "ssh", "git"} {
		cmd := exec.Command(gitBinary(t), "config", "--file", configPath, "protocol."+protocol+".allow", "never")
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir(), "GIT_CONFIG_NOSYSTEM=1"}
		require.NoError(t, cmd.Run())
	}
}

var (
	gitOnce sync.Once
	gitPath string
	gitErr  error
)

func gitBinary(t *testing.T) string {
	t.Helper()
	gitOnce.Do(func() {
		gitPath, gitErr = findGit()
	})
	if gitErr != nil {
		t.Skip(gitErr.Error())
	}
	return gitPath
}

func findGit() (string, error) {
	if path, err := exec.LookPath("git"); err == nil {
		return path, nil
	}
	if _, err := os.Stat("/usr/bin/git"); err == nil {
		return "/usr/bin/git", nil
	}
	return "", fmt.Errorf("git is not installed")
}

func fileURL(path string) string {
	return "file://" + path
}

func cloneInvocation(gitArgs string) string {
	for _, line := range strings.Split(gitArgs, "\n") {
		if strings.HasPrefix(line, "clone ") {
			return line
		}
	}
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

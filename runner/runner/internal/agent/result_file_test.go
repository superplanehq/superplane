package agent

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/superplane/runner/shared/api"
)

// TestSetResultEnvRegressionPTYPath documents the production bug: runHostShellDirectives sets
// cmd.Env from the task, then runShellPTYSession calls setResultEnv. Replacing cmd.Env with
// os.Environ() dropped task variables whenever a result file path was set.
func TestSetResultEnvRegressionPTYPath(t *testing.T) {
	taskEnv, err := processEnvironment([]api.EnvironmentVariable{
		{Name: "RUNNER_TEST_ENV", Value: "from-task"},
	})
	if err != nil {
		t.Fatal(err)
	}
	shellCmd := exec.Command("true")
	shellCmd.Env = taskEnv
	setResultEnv(shellCmd, "/tmp/superplane-result-task.json")

	if !envContains(shellCmd.Env, "RUNNER_TEST_ENV=from-task") {
		t.Fatalf("task env missing after setResultEnv: %#v", shellCmd.Env)
	}
	if !envHasPrefix(shellCmd.Env, envSuperplaneResultFile+"=") {
		t.Fatalf("result file env missing: %#v", shellCmd.Env)
	}
}

func TestSetResultEnvAppendsToExistingEnv(t *testing.T) {
	cmd := exec.Command("true")
	cmd.Env = []string{"ONLY_TASK=1"}
	setResultEnv(cmd, "/var/result.json")
	if len(cmd.Env) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(cmd.Env), cmd.Env)
	}
	if cmd.Env[0] != "ONLY_TASK=1" {
		t.Fatalf("first entry = %q", cmd.Env[0])
	}
	if cmd.Env[1] != envSuperplaneResultFile+"=/var/result.json" {
		t.Fatalf("second entry = %q", cmd.Env[1])
	}
}

func TestSetResultEnvUsesOsEnvironWhenUnset(t *testing.T) {
	t.Setenv("RUNNER_SETRESULTENV_PROBE", "probe-value")
	cmd := exec.Command("true")
	setResultEnv(cmd, "/tmp/result.json")
	joined := strings.Join(cmd.Env, "\n")
	if !strings.Contains(joined, "RUNNER_SETRESULTENV_PROBE=probe-value") {
		t.Fatalf("expected os env baseline: %#v", cmd.Env)
	}
	if !strings.Contains(joined, envSuperplaneResultFile+"=/tmp/result.json") {
		t.Fatalf("expected result env: %#v", cmd.Env)
	}
}

func TestSetResultEnvNoopOnEmptyPath(t *testing.T) {
	cmd := exec.Command("true")
	cmd.Env = []string{"KEEP=1"}
	setResultEnv(cmd, "  ")
	if len(cmd.Env) != 1 || cmd.Env[0] != "KEEP=1" {
		t.Fatalf("cmd.Env = %#v", cmd.Env)
	}
}

func TestApplyCmdEnvArgvAndPipePaths(t *testing.T) {
	taskEnv, err := processEnvironment([]api.EnvironmentVariable{
		{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("true")
	applyCmdEnv(cmd, taskEnv, "/data/result.json")

	if !envContains(cmd.Env, "COMMIT_AUTHOR=alice@example.com") {
		t.Fatalf("task env missing: %#v", cmd.Env)
	}
	if !envContains(cmd.Env, envSuperplaneResultFile+"=/data/result.json") {
		t.Fatalf("result env missing: %#v", cmd.Env)
	}
}

func envContains(env []string, want string) bool {
	for _, pair := range env {
		if pair == want {
			return true
		}
	}
	return false
}

func envHasPrefix(env []string, prefix string) bool {
	for _, pair := range env {
		if strings.HasPrefix(pair, prefix) {
			return true
		}
	}
	return false
}

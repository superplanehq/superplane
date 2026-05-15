package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

// skipIfNoDocker skips tests when docker is unavailable. Production runners
// have it; local dev machines and minimal CI builders often do not.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not on PATH: %v", err)
	}
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err != nil {
		t.Skipf("docker daemon not reachable: %v: %s", err, strings.TrimSpace(string(out)))
	}
}

func TestSanitizeDockerName(t *testing.T) {
	cases := map[string]string{
		"":            "x",
		"i-0abc":      "i-0abc",
		"runner@host": "runner_host",
		"  spaced  ":  "spaced",
		"abc/def":     "abc_def",
		"Some.Name":   "Some.Name",
	}
	for in, want := range cases {
		if got := sanitizeDockerName(in); got != want {
			t.Errorf("sanitizeDockerName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContainerNameIncludesRunnerAndTask(t *testing.T) {
	name := containerName("runner-1", "task-abc")
	if !strings.HasPrefix(name, dockerNamePrefix) {
		t.Fatalf("name %q should have prefix %q", name, dockerNamePrefix)
	}
	if !strings.Contains(name, "runner-1") || !strings.Contains(name, "task-abc") {
		t.Fatalf("name %q should contain both runner_id and task_id", name)
	}
}

func TestCapWriterTruncatesAfterMax(t *testing.T) {
	w := &capWriter{max: 5}
	w.WriteString("ab")
	w.WriteString("cdefg")
	w.WriteString("hijkl")
	got := w.String()
	if !strings.HasPrefix(got, "abcde") {
		t.Errorf("got prefix %q, want prefix abcde, full=%q", got[:5], got)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("expected truncation marker, got %q", got)
	}
}

func TestDockerExecutorArgvSucceeds(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: "test-argv"}
	task := &api.TaskPayload{
		ID:            uniqueTaskID(),
		ExecutionMode: "docker",
		DockerImage:   "alpine:3.20",
		Command:       []string{"echo", "hello-argv"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	exit, output, err := d.Execute(ctx, task, nil, "")
	t.Logf("exit=%d output=%q err=%v", exit, output, err)
	if err != nil || exit != 0 {
		t.Fatalf("docker argv: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "hello-argv") {
		t.Fatalf("expected hello-argv in output, got %q", output)
	}
	assertContainerGone(t, ctx, containerName(d.RunnerID, task.ID))
}

func TestDockerExecutorLiveWriterReceivesExecOutput(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	var live bytes.Buffer
	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: "test-live"}
	task := &api.TaskPayload{
		ID:            uniqueTaskID(),
		ExecutionMode: "docker",
		DockerImage:   "alpine:3.20",
		Commands:      []string{"echo live-exec-marker"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	exit, output, err := d.Execute(ctx, task, &live, "")
	if err != nil || exit != 0 {
		t.Fatalf("docker live writer: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "live-exec-marker") {
		t.Fatalf("returned output missing marker: %q", output)
	}
	liveStr := live.String()
	if !strings.Contains(liveStr, "live-exec-marker") {
		t.Fatalf("live writer missing exec stdout; live=%q", liveStr)
	}
	assertContainerGone(t, ctx, containerName(d.RunnerID, task.ID))
}

func TestDockerExecutorBundledCommandsShareEnv(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: "test-env"}
	task := &api.TaskPayload{
		ID:            uniqueTaskID(),
		ExecutionMode: "docker",
		DockerImage:   "alpine:3.20",
		Commands: []string{
			"export DOCKER_TEST_VAR=hello-shared-env",
			"echo $DOCKER_TEST_VAR",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	exit, output, err := d.Execute(ctx, task, nil, "")
	t.Logf("exit=%d output=%q err=%v", exit, output, err)
	if err != nil || exit != 0 {
		t.Fatalf("env share: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "hello-shared-env") {
		t.Fatalf("expected exported var visible across directives, got %q", output)
	}
	assertContainerGone(t, ctx, containerName(d.RunnerID, task.ID))
}

func TestDockerExecutorBundledCommandsFailFast(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: "test-failfast"}
	task := &api.TaskPayload{
		ID:            uniqueTaskID(),
		ExecutionMode: "docker",
		DockerImage:   "alpine:3.20",
		Commands: []string{
			"echo before",
			"false",
			"echo NEVER",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	exit, output, err := d.Execute(ctx, task, nil, "")
	t.Logf("exit=%d output=%q err=%v", exit, output, err)
	if exit == 0 {
		t.Fatalf("expected non-zero exit on `false`, got 0 (err=%v output=%q)", err, output)
	}
	if !strings.Contains(output, "before") {
		t.Errorf("expected 'before' in output, got %q", output)
	}
	if strings.Contains(output, "NEVER") {
		t.Errorf("set -e should have stopped before NEVER, output=%q", output)
	}
	assertContainerGone(t, ctx, containerName(d.RunnerID, task.ID))
}

func TestDockerExecutorBadImageFailsCleanly(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: "test-badimg"}
	task := &api.TaskPayload{
		ID:            uniqueTaskID(),
		ExecutionMode: "docker",
		DockerImage:   "superplane-nonexistent-image-xxyyzz/nope:0.0.1",
		Command:       []string{"echo", "hi"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	exit, output, err := d.Execute(ctx, task, nil, "")
	t.Logf("exit=%d output=%q err=%v", exit, output, err)
	if err == nil {
		t.Fatalf("expected pull failure error, got nil (exit=%d output=%q)", exit, output)
	}
	if exit == 0 {
		t.Fatalf("expected non-zero exit on bad image, got 0")
	}
	if !strings.Contains(err.Error(), "docker pull") {
		t.Errorf("expected error to mention docker pull, got %v", err)
	}
	assertContainerGone(t, ctx, containerName(d.RunnerID, task.ID))
}

func TestDockerExecutorRequiresRunnerID(t *testing.T) {
	d := &DockerExecutor{MaxOutputBytes: 4096, RunnerID: ""}
	task := &api.TaskPayload{
		ID:          uniqueTaskID(),
		DockerImage: "alpine:3.20",
		Command:     []string{"true"},
	}
	_, _, err := d.Execute(context.Background(), task, nil, "")
	if err == nil || !strings.Contains(err.Error(), "runner_id") {
		t.Fatalf("want runner_id required error, got %v", err)
	}
}

func TestSweepDockerOrphansSkipsEmptyRunnerID(t *testing.T) {
	removed, err := SweepDockerOrphans(context.Background(), "  ")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for empty runner id, got %d", removed)
	}
}

func TestSweepDockerOrphansRemovesPrefixedContainers(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	runnerID := "sweep-test-" + uniqueTaskID()
	name := containerName(runnerID, "leftover")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if out, err := exec.CommandContext(ctx, "docker", "pull", "alpine:3.20").CombinedOutput(); err != nil {
		t.Fatalf("pre-pull alpine: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", name,
		"--entrypoint", dockerIdleEntrypoint,
		"alpine:3.20", dockerIdleArg).CombinedOutput(); err != nil {
		t.Fatalf("create stray container: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	removed, err := SweepDockerOrphans(ctx, runnerID)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed < 1 {
		t.Fatalf("expected at least 1 container removed, got %d", removed)
	}
	assertContainerGone(t, ctx, name)
}

func TestSweepDockerOrphansIgnoresOtherRunners(t *testing.T) {
	skipIfNoDocker(t)
	t.Parallel()

	ownerRunner := "owner-" + uniqueTaskID()
	otherRunner := "other-" + uniqueTaskID()
	otherName := containerName(otherRunner, "alive")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if out, err := exec.CommandContext(ctx, "docker", "pull", "alpine:3.20").CombinedOutput(); err != nil {
		t.Fatalf("pre-pull alpine: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", otherName,
		"--entrypoint", dockerIdleEntrypoint,
		"alpine:3.20", dockerIdleArg).CombinedOutput(); err != nil {
		t.Fatalf("create other container: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", otherName).Run()
	})

	removed, err := SweepDockerOrphans(ctx, ownerRunner)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 containers removed (different runner_id), got %d", removed)
	}
	if out, err := exec.CommandContext(ctx, "docker", "inspect", otherName).CombinedOutput(); err != nil {
		t.Fatalf("other runner's container should still exist: %v\n%s", err, out)
	}
}

func assertContainerGone(t *testing.T, ctx context.Context, name string) {
	t.Helper()
	out, err := exec.CommandContext(ctx, "docker", "inspect", name).CombinedOutput()
	if err == nil {
		t.Fatalf("container %s should be gone but inspect succeeded: %s", name, out)
	}
}

func uniqueTaskID() string {
	return fmt.Sprintf("t%d", time.Now().UnixNano())
}

package agent

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

func TestResolveTaskWorkDirUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ResolveTaskWorkDir()
	if err != nil {
		t.Fatalf("ResolveTaskWorkDir: %v", err)
	}
	if got != home {
		t.Fatalf("got %q, want %q", got, home)
	}
}

func TestResolveTaskWorkDirRejectsMissing(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "missing"))

	_, err := ResolveTaskWorkDir()
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestHostExecutorUsesTaskWorkDirArgv(t *testing.T) {
	workDir := t.TempDir()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}
	task := &api.TaskPayload{
		ID:      "task-workdir-argv",
		Command: []string{sh, "-c", `pwd`},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("host argv pwd: exit=%d err=%v output=%q", exit, err, output)
	}
	if got := strings.TrimSpace(output); got != workDir {
		t.Fatalf("pwd = %q, want %q", got, workDir)
	}
}

func TestHostExecutorUsesTaskWorkDirCommands(t *testing.T) {
	workDir := t.TempDir()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")
	task := &api.TaskPayload{
		ID:       "task-workdir-commands",
		Commands: []string{`pwd`},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("host commands pwd: exit=%d err=%v output=%q", exit, err, output)
	}
	if got := strings.TrimSpace(output); got != workDir {
		t.Fatalf("pwd = %q, want %q", got, workDir)
	}
}

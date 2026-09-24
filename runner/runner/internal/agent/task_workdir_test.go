package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
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
		Commands: models.CommandList{{Command: `pwd`}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("host commands pwd: exit=%d err=%v output=%q", exit, err, output)
	}
	// Pipe path emits cmd_start/cmd_end markers around command stdout.
	if !strings.Contains(output, workDir) {
		t.Fatalf("output = %q, want workdir %q", output, workDir)
	}
}

func TestHostExecutorResetTaskHomeIgnoresLeftoverRepo(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}
	workDir := t.TempDir()
	stale := filepath.Join(workDir, "repo")
	if err := os.MkdirAll(stale, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "stale.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	task := &api.TaskPayload{
		ID:      "task-reset-home",
		Command: []string{sh, "-c", `printf "cwd=%s\nhome=%s\n" "$(pwd)" "$HOME"; test ! -e repo && echo clean`},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{TaskWorkDir: workDir, ResetTaskHome: true}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("reset home: exit=%d err=%v output=%q", exit, err, output)
	}
	wantHome, err := filepath.Abs(isolatedTaskHome(workDir, task.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "home="+wantHome) {
		t.Fatalf("output = %q, want HOME %q", output, wantHome)
	}
	if !strings.Contains(output, ".superplane/homes/"+task.ID) {
		t.Fatalf("output = %q, want isolated cwd", output)
	}
	if !strings.Contains(output, "clean") {
		t.Fatalf("output = %q, want leftover repo hidden", output)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("runner home leftover should stay: %v", err)
	}
	isolated := isolatedTaskHome(workDir, task.ID)
	if _, err := os.Stat(isolated); !os.IsNotExist(err) {
		t.Fatalf("expected isolated home removed, stat err=%v", err)
	}
}

func TestHostExecutorResetTaskHomeWipesCreatedFiles(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}
	workDir := t.TempDir()
	task := &api.TaskPayload{
		ID:      "task-wipe-home",
		Command: []string{sh, "-c", `mkdir repo && printf leftover > repo/file.txt && pwd -P`},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{TaskWorkDir: workDir, ResetTaskHome: true}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("wipe home: exit=%d err=%v output=%q", exit, err, output)
	}
	isolated := strings.TrimSpace(output)
	if isolated == "" || isolated == workDir {
		t.Fatalf("expected isolated cwd, got %q", isolated)
	}
	if _, err := os.Stat(filepath.Join(isolated, "repo")); !os.IsNotExist(err) {
		t.Fatalf("expected task home wiped, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "repo")); !os.IsNotExist(err) {
		t.Fatalf("runner home must stay empty of repo, stat err=%v", err)
	}
}

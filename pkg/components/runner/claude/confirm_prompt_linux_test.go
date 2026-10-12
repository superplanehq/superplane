//go:build linux

package claude

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfirmPromptGuardDiesWithParentProcessGroup(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "confirm_prompt.js"))
	require.NoError(t, err)
	cmd := exec.Command("node", script, "sleep 30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	require.NoError(t, cmd.Start())
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	require.NoError(t, err)
	defer func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		_ = cmd.Wait()
	}()

	sleepPid := waitForDescendant(t, cmd.Process.Pid, "sleep")
	childGroup, err := syscall.Getpgid(sleepPid)
	require.NoError(t, err)
	require.Equal(t, pgid, childGroup)

	require.NoError(t, syscall.Kill(-pgid, syscall.SIGKILL))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if processGone(sleepPid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("sleep %d survived the guard process group", sleepPid)
}

func TestConfirmPromptGuardKillsCommandThatIgnoresTerm(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "confirm_prompt.js"))
	require.NoError(t, err)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	childScript := filepath.Join(dir, "child.js")
	require.NoError(t, os.WriteFile(childScript, []byte(ignoreTermChild), 0o644))

	command := fmt.Sprintf("node %s %s; true", quoteShell(childScript), quoteShell(pidFile))
	cmd := exec.Command("node", script, command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	reader, writer, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	defer writer.Close()
	defer reader.Close()
	cmd.Stdin = reader
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	require.NoError(t, cmd.Start())
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	require.NoError(t, err)
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		done <- cmd.Wait()
		close(finished)
	}()
	t.Cleanup(func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		<-finished
	})

	select {
	case err := <-done:
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		require.Equal(t, 1, exitErr.ExitCode())
	case <-time.After(5 * time.Second):
		t.Fatal("guard stayed running after a command ignored SIGTERM")
	}

	pidText, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidText)))
	require.NoError(t, err)
	require.True(t, processGone(pid), "command %d kept running after the guard stopped it", pid)
	require.Contains(t, buf.String(), "Ok to proceed?")
}

func quoteShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

const ignoreTermChild = `#!/usr/bin/env node
"use strict";
const fs = require("fs");
process.on("SIGTERM", () => {});
process.on("SIGHUP", () => {});
fs.writeFileSync(process.argv[2], String(process.pid));
process.stdout.write("Ok to proceed? (y)\n");
try {
  fs.readSync(0, Buffer.alloc(1));
} catch (_err) {}
setInterval(() => {}, 86400000);
`

func processGone(pid int) bool {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	closeParen := bytes.LastIndex(data, []byte(")"))
	if closeParen < 0 || closeParen+2 >= len(data) {
		return false
	}
	fields := strings.Fields(string(data[closeParen+2:]))
	if len(fields) == 0 {
		return false
	}
	return fields[0] == "Z" || fields[0] == "X"
}

func waitForDescendant(t *testing.T, root int, command string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if pid := findDescendant(root, command); pid != 0 {
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("missing descendant %s under %d", command, root)
	return 0
}

func findDescendant(root int, command string) int {
	var found int
	var walk func(int)
	walk = func(pid int) {
		if found != 0 || pid == 0 {
			return
		}
		cmdline, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
		if err == nil && pid != root && strings.HasPrefix(string(cmdline), command+"\x00") {
			found = pid
			return
		}
		children, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/task/" + strconv.Itoa(pid) + "/children")
		if err != nil {
			return
		}
		for _, field := range strings.Fields(string(children)) {
			child, convErr := strconv.Atoi(field)
			if convErr != nil {
				continue
			}
			walk(child)
		}
	}
	walk(root)
	return found
}

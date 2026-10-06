//go:build unix

package claude

import (
	"bytes"
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

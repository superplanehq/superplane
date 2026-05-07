//go:build unix

package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func runHostShellDirectives(ctx context.Context, maxOut int, scripts []string) (int, string, error) {
	bash, err := resolveBash()
	if err != nil {
		return 1, "", err
	}
	cmd := exec.CommandContext(ctx, bash, "--norc", "--noprofile", "+m", "--noediting", "-i")
	return runShellPTYSession(ctx, maxOut, cmd, scripts)
}

func runDockerShellDirectives(ctx context.Context, maxOut int, image string, scripts []string) (int, string, error) {
	args := []string{"run", "-i", "--rm", strings.TrimSpace(image), "/bin/bash", "--norc", "--noprofile", "+m", "--noediting", "-i"}
	cmd := exec.CommandContext(ctx, "docker", args...)
	return runShellPTYSession(ctx, maxOut, cmd, scripts)
}

func resolveBash() (string, error) {
	if p := strings.TrimSpace(os.Getenv("RUNNER_SHELL")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	path, err := exec.LookPath("bash")
	if err != nil {
		return "", fmt.Errorf("interactive runner needs bash (set RUNNER_SHELL or install bash): %w", err)
	}
	return path, nil
}

func bashSingleQuotedPath(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}

type shellSession struct {
	mu     sync.Mutex
	raw    []byte
	out    bytes.Buffer
	maxOut int

	master io.Writer
}

func (s *shellSession) appendOut(p []byte) {
	if s.maxOut <= 0 || len(p) == 0 {
		return
	}
	room := s.maxOut - s.out.Len()
	if room <= 0 {
		return
	}
	if len(p) > room {
		s.out.Write(p[:room])
		s.out.WriteString("\n…(truncated)")
		return
	}
	s.out.Write(p)
}

func (s *shellSession) push(b []byte) {
	if len(b) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raw = append(s.raw, b...)
}

func (s *shellSession) consumePrefix(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 {
		return
	}
	if n >= len(s.raw) {
		s.raw = nil
		return
	}
	s.raw = append([]byte(nil), s.raw[n:]...)
}

func (s *shellSession) snapshot() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.raw...)
}

func (s *shellSession) writeLine(line string) error {
	_, err := s.master.Write(append([]byte(line), '\n'))
	return err
}

func randomMark(prefix string) string {
	return fmt.Sprintf("%s-%016x-%016x", prefix, rand.Uint64(), rand.Uint64())
}

func runShellPTYSession(ctx context.Context, maxOut int, shellCmd *exec.Cmd, directives []string) (_ int, out string, err error) {
	parts := normalizeDirectiveLines(directives)
	if len(parts) == 0 {
		return 1, "", errEmptyCommands()
	}

	tmpRoot, err := os.MkdirTemp("", "runner-sh-*")
	if err != nil {
		return 1, "", err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	// github.com/creack/pty Start sets Setsid+Setctty (Linux). Combining that with Setpgid has
	// produced fork/exec /bin/bash EPERM on Ubuntu EC2; omit Setpgid on Linux only. Non-Linux
	// Unix (e.g. darwin dev) still uses Setpgid for stable process-group teardown.
	if runtime.GOOS != "linux" {
		shellCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	master, err := pty.Start(shellCmd)
	if err != nil {
		return 1, "", fmt.Errorf("pty start shell: %w", err)
	}

	stopped := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			if shellCmd.Process != nil {
				_ = syscall.Kill(-shellCmd.Process.Pid, syscall.SIGKILL)
			}
		case <-stopped:
		}
	}()

	defer func() {
		close(stopped)
		if shellCmd.Process != nil {
			_ = syscall.Kill(-shellCmd.Process.Pid, syscall.SIGKILL)
			_, _ = shellCmd.Process.Wait()
		}
		_ = master.Close()
		wg.Wait()
	}()

	max := maxOut
	if max <= 0 {
		max = 512 * 1024
	}

	sess := &shellSession{master: master, maxOut: max}

	readerErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, rerr := master.Read(buf)
			if n > 0 {
				sess.push(buf[:n])
			}
			if rerr != nil {
				if errors.Is(rerr, io.EOF) {
					readerErr <- io.EOF
					return
				}
				readerErr <- rerr
				return
			}
		}
	}()

	bootMarker := fmt.Sprintf("bootready-%d", time.Now().UnixNano())
	if err := sess.writeLine(fmt.Sprintf(`set +m; export PS1=''; set +H; stty -echo 2>/dev/null || true; echo '%s'`, bootMarker)); err != nil {
		return 1, truncateString(sess.out.String(), max), err
	}
	if err := waitForSubstring(ctx, sess, []byte(bootMarker), readerErr, 30*time.Second); err != nil {
		return 1, truncateString(sess.out.String(), max), err
	}

	for i, dir := range parts {
		dPath := filepath.Join(tmpRoot, fmt.Sprintf("d%d.sh", i))
		if err := os.WriteFile(dPath, []byte(dir+"\n"), 0600); err != nil {
			return 1, truncateString(sess.out.String(), max), err
		}
		start := randomMark("s")
		end := randomMark("e")
		instr := fmt.Sprintf(
			`echo -e "\001 %s"; source %s; AGENT_CMD_RESULT=$?; echo -e "\001 %s $AGENT_CMD_RESULT"; echo "exit $AGENT_CMD_RESULT" | sh`,
			start,
			bashSingleQuotedPath(dPath),
			end,
		)
		if err := sess.writeLine(instr); err != nil {
			return 1, truncateString(sess.out.String(), max), err
		}
		if err := discardThroughStartMarker(ctx, sess, start, readerErr, 90*time.Second); err != nil {
			return 1, truncateString(sess.out.String(), max), err
		}
		code, perr := readThroughEndMarker(ctx, sess, end, readerErr, 8*time.Minute)
		if perr != nil {
			return code, truncateString(sess.out.String(), max), perr
		}
		if code != 0 {
			return code, truncateString(sess.out.String(), max), fmt.Errorf("exit code %d", code)
		}
	}

	return 0, truncateString(sess.out.String(), max), nil
}

func waitForSubstring(ctx context.Context, sess *shellSession, needle []byte, readerErr <-chan error, deadline time.Duration) error {
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timeout waiting for shell")
		case r := <-readerErr:
			if r == io.EOF {
				return fmt.Errorf("shell closed during boot")
			}
			if r != nil {
				return r
			}
		default:
			data := sess.snapshot()
			if idx := bytes.Index(data, needle); idx >= 0 {
				sess.consumePrefix(idx + len(needle))
				return nil
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// discardThroughStartMarker drops PTY noise up to and including "start\r\n".
func discardThroughStartMarker(ctx context.Context, sess *shellSession, startMark string, readerErr <-chan error, deadline time.Duration) error {
	cr := []byte(startMark + "\r\n")
	lf := []byte(startMark + "\n")
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timeout waiting for start marker")
		case r := <-readerErr:
			if r == io.EOF {
				return fmt.Errorf("shell closed before start marker")
			}
			if r != nil {
				return r
			}
		default:
			data := sess.snapshot()
			var idx, skip int
			if j := bytes.Index(data, cr); j >= 0 {
				idx, skip = j, len(cr)
			} else if j := bytes.Index(data, lf); j >= 0 {
				idx, skip = j, len(lf)
			}
			if idx >= 0 {
				sess.consumePrefix(idx + skip)
				return nil
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func readThroughEndMarker(ctx context.Context, sess *shellSession, endMark string, readerErr <-chan error, deadline time.Duration) (int, error) {
	re := regexp.MustCompile(`\x01\s*` + regexp.QuoteMeta(endMark) + `\s+(\d+)\r?\n`)
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return 1, ctx.Err()
		case <-timer.C:
			return 1, fmt.Errorf("timeout waiting for end marker")
		case r := <-readerErr:
			if r == io.EOF {
				return 1, fmt.Errorf("shell closed before end marker")
			}
			if r != nil {
				return 1, r
			}
		default:
			data := sess.snapshot()
			loc := re.FindSubmatchIndex(data)
			if loc == nil {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			before := data[:loc[0]]
			sess.appendOut(before)
			code, convErr := strconv.Atoi(string(data[loc[2]:loc[3]]))
			if convErr != nil {
				return 1, convErr
			}
			sess.consumePrefix(loc[1])
			return code, nil
		}
	}
}

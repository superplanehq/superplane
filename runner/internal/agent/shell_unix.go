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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// usePipeShell is true when RUNNER_SHELL_USE_PIPE is set: host directives run as a single non-PTY
// bash script bundle. Default (unset) is PTY + markers — what production EC2 runners use. The
// env var exists for broken-PTY environments and is covered by TestHostShellPipeBundleEcho.
func usePipeShell() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("RUNNER_SHELL_USE_PIPE")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func runHostShellDirectives(ctx context.Context, maxOut int, scripts []string) (int, string, error) {
	bash, err := resolveBash()
	if err != nil {
		return 1, "", err
	}
	if usePipeShell() {
		return runHostShellDirectivesPipe(ctx, maxOut, bash, scripts)
	}
	// Plain exec.Command (not CommandContext): attaching ctx to os/exec races with creack/pty on
	// some Darwin setups; cancellation is handled inside runShellPTYSession via ctx + Process.Kill().
	// Apple bash 3.2 mishandles `+m` together with `--noediting` (option parsing surfaces as
	// `/bin/bash: --: invalid option`). Keep job-control off (`+m`) for non-interactive scripts but
	// omit `--noediting`; readline editing is irrelevant on our PTY-driven line protocol anyway.
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	return runShellPTYSession(ctx, maxOut, cmd, scripts)
}

type cappedShellWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *cappedShellWriter) Write(p []byte) (int, error) {
	if w.max <= 0 {
		return w.buf.Write(p)
	}
	if w.buf.Len() >= w.max {
		return len(p), nil
	}
	room := w.max - w.buf.Len()
	if len(p) <= room {
		return w.buf.Write(p)
	}
	_, _ = w.buf.Write(p[:room])
	_, _ = w.buf.WriteString("\n…(truncated)\n")
	return len(p), nil
}

func writeDirectiveBundle(tmpRoot string, parts []string) (metaPath string, err error) {
	var meta strings.Builder
	meta.WriteString("set -e\nset +u\nset -o pipefail\n")
	for i, dir := range parts {
		hostPath := filepath.Join(tmpRoot, fmt.Sprintf("d%d.sh", i))
		if err := os.WriteFile(hostPath, []byte(dir+"\n"), 0600); err != nil {
			return "", err
		}
		meta.WriteString("source ")
		meta.WriteString(bashSingleQuotedPath(hostPath))
		meta.WriteString("\n")
	}
	metaPath = filepath.Join(tmpRoot, "_meta.sh")
	if err := os.WriteFile(metaPath, []byte(meta.String()), 0600); err != nil {
		return "", err
	}
	return metaPath, nil
}

// runHostShellDirectivesPipe runs directives in one bash process without a PTY (same source bundle
// semantics as the PTY path: cwd/env persist across sources).
func runHostShellDirectivesPipe(ctx context.Context, maxOut int, bash string, scripts []string) (int, string, error) {
	parts := normalizeDirectiveLines(scripts)
	if len(parts) == 0 {
		return 1, "", errEmptyCommands()
	}
	tmpRoot, err := os.MkdirTemp("", "runner-sh-*")
	if err != nil {
		return 1, "", err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	metaPath, err := writeDirectiveBundle(tmpRoot, parts)
	if err != nil {
		return 1, "", err
	}

	cmd := exec.CommandContext(ctx, bash, "--norc", "--noprofile", metaPath)
	max := maxOut
	if max <= 0 {
		max = 512 * 1024
	}
	var buf bytes.Buffer
	w := &cappedShellWriter{buf: &buf, max: max}
	cmd.Stdout = w
	cmd.Stderr = w

	if runErr := cmd.Run(); runErr != nil {
		outStr := truncateString(buf.String(), max)
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			c := ee.ExitCode()
			return c, outStr, fmt.Errorf("exit code %d", c)
		}
		return 1, outStr, runErr
	}
	return 0, truncateString(buf.String(), max), nil
}

func runDockerShellDirectives(ctx context.Context, maxOut int, image string, scripts []string) (int, string, error) {
	args := []string{"run", "-i", "--rm", strings.TrimSpace(image), "/bin/bash", "--norc", "--noprofile", "+m", "-i"}
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

// killShellProcess tears down the PTY bash process only (same as cmd/ptyprobe). We avoid
// syscall.Kill(-pid) here: negative PGID kills interact badly with creack/pty Setsid/session
// leadership on Darwin and some CI environments, yielding EOF/EIO on the PTY master mid-session.
// EC2/Linux workloads that spawn detached children should use RUNNER_SHELL_USE_PIPE or rely on
// bash job control; broader group-kill can be revisited if needed.
func killShellProcess(shellCmd *exec.Cmd) {
	if shellCmd == nil || shellCmd.Process == nil {
		return
	}
	_ = shellCmd.Process.Kill()
}

func runShellPTYSession(ctx context.Context, maxOut int, shellCmd *exec.Cmd, directives []string) (_ int, out string, err error) {
	parts := normalizeDirectiveLines(directives)
	if len(parts) == 0 {
		return 1, "", errEmptyCommands()
	}

	bootMarker := fmt.Sprintf("bootready-%d", time.Now().UnixNano())

	// github.com/creack/pty Start sets Setsid+Setctty on every non-Windows Unix. Combining that
	// with Setpgid has produced fork/exec EPERM on Linux EC2 and on macOS (Darwin).
	master, err := pty.Start(shellCmd)
	if err != nil {
		return 1, "", fmt.Errorf("pty start shell: %w", err)
	}

	ctxDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			killShellProcess(shellCmd)
		case <-ctxDone:
		}
	}()
	defer func() {
		close(ctxDone)
		if shellCmd.Process != nil {
			killShellProcess(shellCmd)
			_, _ = shellCmd.Process.Wait()
		}
		_ = master.Close()
	}()

	max := maxOut
	if max <= 0 {
		max = 512 * 1024
	}

	sess := &shellSession{master: master, maxOut: max}

	// Boot synchronously (no concurrent master reader): wait for a full line equal to bootMarker so
	// we do not treat the marker as a substring inside the echoed `echo '…'` line.
	bootDeadline := time.Now().Add(30 * time.Second)
	if err := sess.writeLine(fmt.Sprintf(`echo '%s'`, bootMarker)); err != nil {
		return 1, truncateString(sess.out.String(), max), err
	}
	buf := make([]byte, 4096)
	var post []byte
	for {
		if err := ctx.Err(); err != nil {
			return 1, truncateString(sess.out.String(), max), err
		}
		if time.Now().After(bootDeadline) {
			return 1, truncateString(sess.out.String(), max), fmt.Errorf("timeout waiting for boot marker (partial: %q)", post)
		}
		if cut := consumeThroughFirstExactLine(post, bootMarker); cut > 0 {
			post = post[cut:]
			break
		}
		n, rerr := master.Read(buf)
		if n > 0 {
			post = append(post, buf[:n]...)
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return 1, truncateString(sess.out.String(), max), fmt.Errorf("shell closed during boot (partial: %q)", post)
			}
			return 1, truncateString(sess.out.String(), max), rerr
		}
	}
	if len(post) > 0 {
		sess.push(post)
	}

	tmpRoot, err := os.MkdirTemp("", "runner-sh-*")
	if err != nil {
		return 1, truncateString(sess.out.String(), max), err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

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

	for i, dir := range parts {
		dPath := filepath.Join(tmpRoot, fmt.Sprintf("d%d.sh", i))
		if err := os.WriteFile(dPath, []byte(dir+"\n"), 0600); err != nil {
			return 1, truncateString(sess.out.String(), max), err
		}
		start := randomMark("s")
		end := randomMark("e")
		// ANSI-C $'…' emits SOH reliably on Bash 3.2 (macOS) and modern Linux; avoid echo -e (\001 via $').
		// No trailing `| sh`: under PTY+interactive bash that pipeline correlated with early slave close on Darwin.
		instr := fmt.Sprintf(
			`echo $'\001 %s\n'; source %s; AGENT_CMD_RESULT=$?; echo $'\001 %s '"$AGENT_CMD_RESULT"`,
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

func consumeThroughFirstExactLine(data []byte, want string) int {
	off := 0
	for off < len(data) {
		nl := bytes.IndexByte(data[off:], '\n')
		if nl < 0 {
			return 0
		}
		lineStart := off
		lineEnd := off + nl
		body := strings.TrimRight(string(data[lineStart:lineEnd]), "\r")
		if body == want {
			return lineEnd + 1
		}
		off = lineEnd + 1
	}
	return 0
}

// discardThroughStartMarker drops PTY noise up to and including the start-marker line.
// The echoed line is SOH, optional spaces, startMark, CRLF or LF (see instr). Consuming only
// startMark+"\n" leaves a stray \x01 prefix in raw; readThroughEndMarker then matches the wrong
// \x01 and never sees the end marker.
func discardThroughStartMarker(ctx context.Context, sess *shellSession, startMark string, readerErr <-chan error, deadline time.Duration) error {
	withSOH := regexp.MustCompile(`\x01\s*` + regexp.QuoteMeta(startMark) + `\r?\n`)
	plain := regexp.MustCompile(regexp.QuoteMeta(startMark) + `\r?\n`)
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
			var loc []int
			if loc = withSOH.FindIndex(data); loc == nil {
				loc = plain.FindIndex(data)
			}
			if loc != nil {
				sess.consumePrefix(loc[1])
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

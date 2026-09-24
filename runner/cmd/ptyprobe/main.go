// Package main is a minimal PTY sanity check: same rough shape as the runner (bash -i + creack/pty).
//
// Run twice and compare:
//
//	go run ./cmd/ptyprobe
//
// 1) From Cursor’s agent terminal (or whatever reproduces your failure).
// 2) From Terminal.app / iTerm on the same Mac.
//
// If (1) fails and (2) passes, PTY works on the machine but not in that execution context.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

func main() {
	wd, _ := os.Getwd()
	fmt.Printf("ptyprobe pid=%d ppid=%d cwd=%s\n", os.Getpid(), os.Getppid(), wd)

	bash, err := exec.LookPath("bash")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: no bash on PATH: %v\n", err)
		os.Exit(2)
	}

	// Match runner host bash argv: `+m` + `--noediting` breaks Apple bash 3.2 (`--: invalid option`).
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	m, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: pty.Start: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	marker := fmt.Sprintf("PTYPROBE_OK_%d", time.Now().UnixNano())
	line := "echo " + marker + "\n"
	if _, err := m.Write([]byte(line)); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: write to pty master: %v\n", err)
		os.Exit(1)
	}

	type readResult struct {
		found bool
		acc   []byte
		err   error
	}

	buf := make([]byte, 4096)
	resCh := make(chan readResult, 1)
	go func() {
		var acc []byte
		for {
			n, rerr := m.Read(buf)
			if n > 0 {
				acc = append(acc, buf[:n]...)
				if bytes.Contains(acc, []byte(marker)) {
					resCh <- readResult{true, append([]byte(nil), acc...), nil}
					return
				}
			}
			if rerr != nil {
				resCh <- readResult{false, append([]byte(nil), acc...), rerr}
				return
			}
		}
	}()

	select {
	case <-time.After(8 * time.Second):
		fmt.Fprintf(os.Stderr, "FAIL: timeout waiting for marker (PTY read produced no line containing echo output)\n")
		os.Exit(1)
	case r := <-resCh:
		if r.found {
			fmt.Printf("PASS: read PTY output containing %q (%d bytes total)\n", marker, len(r.acc))
			if len(os.Args) < 2 || os.Args[1] != "--two" {
				os.Exit(0)
			}
			// Second phase: same PTY session, runner-shaped boot line (quoted marker).
			boot := fmt.Sprintf("bootready-%d", time.Now().UnixNano())
			line2 := fmt.Sprintf("echo '%s'\n", boot)
			if _, err := m.Write([]byte(line2)); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL phase2 write: %v\n", err)
				os.Exit(1)
			}
			resCh2 := make(chan readResult, 1)
			go func() {
				var acc []byte
				for {
					n, rerr := m.Read(buf)
					if n > 0 {
						acc = append(acc, buf[:n]...)
						if bytes.Contains(acc, []byte(boot)) {
							resCh2 <- readResult{true, append([]byte(nil), acc...), nil}
							return
						}
					}
					if rerr != nil {
						resCh2 <- readResult{false, append([]byte(nil), acc...), rerr}
						return
					}
				}
			}()
			select {
			case <-time.After(8 * time.Second):
				fmt.Fprintf(os.Stderr, "FAIL phase2 timeout\n")
				os.Exit(1)
			case r2 := <-resCh2:
				if r2.found {
					fmt.Printf("PASS phase2: saw boot marker %q\n", boot)
					os.Exit(0)
				}
				fmt.Fprintf(os.Stderr, "FAIL phase2: err=%v partial=%s\n", r2.err, r2.acc)
				os.Exit(1)
			}
		}
		if r.err == io.EOF {
			fmt.Fprintf(os.Stderr, "FAIL: PTY master EOF before marker (slave side closed)\n--- partial (%d bytes) ---\n%s\n", len(r.acc), r.acc)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "FAIL: read pty master: %v\n--- partial (%d bytes) ---\n%s\n", r.err, len(r.acc), r.acc)
		os.Exit(1)
	}
}

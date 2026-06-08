package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
)

const bashProgramName = "script.sh"

// buildBashProgram wraps user script and calls main(payload) with the message chain.
func buildBashProgram(userScript string, messageChain json.RawMessage) ([]byte, error) {
	userScript = strings.TrimSpace(stripUserShebang(userScript))
	if userScript == "" {
		return nil, errors.New("empty script")
	}
	chain := bytes.TrimSpace(messageChain)
	if len(chain) == 0 {
		chain = []byte("{}")
	}
	if !json.Valid(chain) {
		return nil, errors.New("invalid message_chain JSON")
	}

	var buf bytes.Buffer
	buf.WriteString("#!/usr/bin/env bash\n")
	buf.WriteString("set -euo pipefail\n\n")
	buf.WriteString("payload=")
	buf.WriteString(bashSingleQuoted(string(chain)))
	buf.WriteString("\n\n")
	buf.WriteString(userScript)
	buf.WriteString("\n\n")
	buf.WriteString(`if ! declare -f main >/dev/null 2>&1; then
  echo "main(payload) is required" >&2
  exit 1
fi
result="$(main "$payload")"
printf '%s' "$result" > "${SUPERPLANE_RESULT_FILE:?SUPERPLANE_RESULT_FILE is required}"
`)
	return buf.Bytes(), nil
}

func stripUserShebang(script string) string {
	script = strings.TrimSpace(script)
	if !strings.HasPrefix(script, "#!") {
		return script
	}
	if idx := strings.IndexByte(script, '\n'); idx >= 0 {
		return strings.TrimSpace(script[idx+1:])
	}
	return ""
}

func bashSingleQuoted(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}

func writeBashProgram(dir, userScript string, messageChain json.RawMessage) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	program, err := buildBashProgram(userScript, messageChain)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, bashProgramName)
	if err := os.WriteFile(path, program, 0700); err != nil {
		return "", err
	}
	return path, nil
}

func runBashHost(
	ctx context.Context,
	max int,
	workDir string,
	task *api.TaskPayload,
	env []string,
	live io.Writer,
	resultHostPath string,
) (int, string, error) {
	setup := api.NormalizeCommandLines(task.SetupCommands)
	var combinedOut strings.Builder
	if len(setup) > 0 {
		exit, setupOut, err := runHostShellDirectives(ctx, max, workDir, setup, env, live, resultHostPath)
		combinedOut.WriteString(setupOut)
		if exit != 0 {
			return exit, truncateString(combinedOut.String(), max), err
		}
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), fmt.Errorf("bash not available on this runner: %w", err)
	}

	scriptDir := filepath.Join(workDir, ".superplane", task.ID)
	programPath, err := writeBashProgram(scriptDir, task.Script, task.MessageChain)
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), err
	}
	defer os.RemoveAll(scriptDir)

	cmd := exec.CommandContext(ctx, bash, programPath)
	cmd.Dir = workDir
	applyCmdEnv(cmd, env, resultHostPath)

	var buf bytes.Buffer
	if live != nil {
		mw := io.MultiWriter(&buf, live)
		cmd.Stdout = mw
		cmd.Stderr = mw
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}

	startedAt := time.Now()
	bashIndex := len(setup)
	writeLiveLogCommandStart(live, bashIndex, "bash "+bashProgramName, startedAt)
	runErr := cmd.Run()
	combinedOut.WriteString(buf.String())
	out := truncateString(combinedOut.String(), max)
	exit := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		writeLiveLogCommandEnd(live, bashIndex, exit, time.Since(startedAt))
		return exit, out, runErr
	}
	writeLiveLogCommandEnd(live, bashIndex, exit, time.Since(startedAt))
	return exit, out, nil
}

func prepareDockerBashWorkDir(task *api.TaskPayload) (string, error) {
	dir, err := os.MkdirTemp("", "superplane-bash-"+task.ID+"-*")
	if err != nil {
		return "", err
	}
	if _, err := writeBashProgram(dir, task.Script, task.MessageChain); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func dockerBashProgramPath() string {
	return dockerScriptWorkMount + "/" + bashProgramName
}

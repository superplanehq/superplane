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

const pythonProgramName = "script.py"

// buildPythonProgram wraps user script and calls main(payload) with the message chain.
func buildPythonProgram(userScript string, messageChain json.RawMessage) ([]byte, error) {
	userScript = strings.TrimSpace(userScript)
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
	chainLiteral, err := json.Marshal(string(chain))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("import json\n")
	buf.WriteString("import os\n")
	buf.WriteString("import sys\n")
	buf.WriteString("import traceback\n\n")
	buf.WriteString("payload = json.loads(")
	buf.Write(chainLiteral)
	buf.WriteString(")\n\n")
	buf.WriteString(userScript)
	buf.WriteString("\n\n")
	buf.WriteString(`if __name__ == "__main__":
    try:
        main_fn = globals().get("main")
        if not callable(main_fn):
            raise RuntimeError("main(payload) is required")
        result = main_fn(payload)
        with open(os.environ["SUPERPLANE_RESULT_FILE"], "w", encoding="utf-8") as f:
            json.dump(result, f)
    except Exception:
        traceback.print_exc()
        sys.exit(1)
`)
	return buf.Bytes(), nil
}

func writePythonProgram(dir, userScript string, messageChain json.RawMessage) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	program, err := buildPythonProgram(userScript, messageChain)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, pythonProgramName)
	if err := os.WriteFile(path, program, 0600); err != nil {
		return "", err
	}
	return path, nil
}

func runPythonHost(
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

	python, err := exec.LookPath("python3")
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), fmt.Errorf("python3 not available on this runner: %w", err)
	}

	scriptDir := filepath.Join(workDir, ".superplane", task.ID)
	programPath, err := writePythonProgram(scriptDir, task.Script, task.MessageChain)
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), err
	}
	defer os.RemoveAll(scriptDir)

	cmd := exec.CommandContext(ctx, python, programPath)
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
	pyIndex := len(setup)
	writeLiveLogCommandStart(live, pyIndex, "python3 "+pythonProgramName, startedAt)
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
		writeLiveLogCommandEnd(live, pyIndex, exit, time.Since(startedAt))
		return exit, out, runErr
	}
	writeLiveLogCommandEnd(live, pyIndex, exit, time.Since(startedAt))
	return exit, out, nil
}

func prepareDockerPythonWorkDir(task *api.TaskPayload) (string, error) {
	dir, err := os.MkdirTemp("", "superplane-py-"+task.ID+"-*")
	if err != nil {
		return "", err
	}
	if _, err := writePythonProgram(dir, task.Script, task.MessageChain); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func dockerPythonProgramPath() string {
	return dockerScriptWorkMount + "/" + pythonProgramName
}

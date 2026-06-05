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

const javaScriptProgramName = "script.js"

// buildJavaScriptProgram wraps user script with global $ and main() result handling.
func buildJavaScriptProgram(userScript string, messageChain json.RawMessage) ([]byte, error) {
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

	var buf bytes.Buffer
	buf.WriteString("'use strict';\n")
	buf.WriteString("const fs = require('fs');\n")
	buf.WriteString("globalThis.$ = ")
	buf.Write(chain)
	buf.WriteString(";\n\n")
	buf.WriteString(userScript)
	buf.WriteString("\n\n")
	buf.WriteString(`Promise.resolve(typeof main === 'function' ? main() : (() => { throw new Error('main() is required'); })())
  .then(result => {
    fs.writeFileSync(process.env.SUPERPLANE_RESULT_FILE, JSON.stringify(result ?? null));
  })
  .catch(err => {
    console.error(err && err.stack ? err.stack : err);
    process.exit(1);
  });
`)
	return buf.Bytes(), nil
}

func writeJavaScriptProgram(dir, userScript string, messageChain json.RawMessage) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	program, err := buildJavaScriptProgram(userScript, messageChain)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, javaScriptProgramName)
	if err := os.WriteFile(path, program, 0600); err != nil {
		return "", err
	}
	return path, nil
}

func runJavaScriptHost(
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

	node, err := exec.LookPath("node")
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), fmt.Errorf("node not available on this runner: %w", err)
	}

	scriptDir := filepath.Join(workDir, ".superplane", task.ID)
	programPath, err := writeJavaScriptProgram(scriptDir, task.Script, task.MessageChain)
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), err
	}
	defer os.RemoveAll(scriptDir)

	cmd := exec.CommandContext(ctx, node, programPath)
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
	jsIndex := len(setup)
	writeLiveLogCommandStart(live, jsIndex, "node "+javaScriptProgramName, startedAt)
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
		writeLiveLogCommandEnd(live, jsIndex, exit, time.Since(startedAt))
		return exit, out, runErr
	}
	writeLiveLogCommandEnd(live, jsIndex, exit, time.Since(startedAt))
	return exit, out, nil
}

const dockerJavaScriptWorkMount = "/superplane-work"

func prepareDockerJavaScriptWorkDir(task *api.TaskPayload) (string, error) {
	dir, err := os.MkdirTemp("", "superplane-js-"+task.ID+"-*")
	if err != nil {
		return "", err
	}
	if _, err := writeJavaScriptProgram(dir, task.Script, task.MessageChain); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func dockerJavaScriptProgramPath() string {
	return dockerJavaScriptWorkMount + "/" + javaScriptProgramName
}

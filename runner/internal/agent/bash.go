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

const (
	bashProgramName = "script.sh"
	bashPayloadName = "payload.json"
)

type bashTaskFiles struct {
	scriptPath  string
	payloadPath string
}

// writeBashTaskFiles stores the user script unchanged and message_chain in payload.json.
// Users read SUPERPLANE_PAYLOAD_FILE and write JSON to SUPERPLANE_RESULT_FILE.
func writeBashTaskFiles(dir, userScript string, messageChain json.RawMessage) (bashTaskFiles, error) {
	if strings.TrimSpace(userScript) == "" {
		return bashTaskFiles{}, errors.New("empty script")
	}
	chain := bytes.TrimSpace(messageChain)
	if len(chain) == 0 {
		chain = []byte("{}")
	}
	if !json.Valid(chain) {
		return bashTaskFiles{}, errors.New("invalid message_chain JSON")
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return bashTaskFiles{}, err
	}
	scriptPath := filepath.Join(dir, bashProgramName)
	payloadPath := filepath.Join(dir, bashPayloadName)
	if err := os.WriteFile(scriptPath, []byte(userScript), 0700); err != nil {
		return bashTaskFiles{}, err
	}
	if err := os.WriteFile(payloadPath, chain, 0600); err != nil {
		return bashTaskFiles{}, err
	}
	return bashTaskFiles{scriptPath: scriptPath, payloadPath: payloadPath}, nil
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
	files, err := writeBashTaskFiles(scriptDir, task.Script, task.MessageChain)
	if err != nil {
		return 1, truncateString(combinedOut.String(), max), err
	}
	defer os.RemoveAll(scriptDir)

	cmd := exec.CommandContext(ctx, bash, files.scriptPath)
	cmd.Dir = workDir
	applyCmdEnv(cmd, env, resultHostPath)
	setPayloadEnv(cmd, files.payloadPath)

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
	if _, err := writeBashTaskFiles(dir, task.Script, task.MessageChain); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func dockerBashProgramPath() string {
	return dockerScriptWorkMount + "/" + bashProgramName
}

func dockerBashPayloadPath() string {
	return dockerScriptWorkMount + "/" + bashPayloadName
}

package agent

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

const (
	envSuperplaneResultFile  = "SUPERPLANE_RESULT_FILE"
	envSuperplanePayloadFile = "SUPERPLANE_PAYLOAD_FILE"
)

// applyCmdEnv sets task environment first, then SUPERPLANE_RESULT_FILE when hostPath is set.
func applyCmdEnv(cmd *exec.Cmd, env []string, hostPath string) {
	if cmd == nil {
		return
	}
	if env != nil {
		cmd.Env = env
	}
	setResultEnv(cmd, hostPath)
}

func setResultEnv(cmd *exec.Cmd, hostPath string) {
	setProcessEnv(cmd, envSuperplaneResultFile, hostPath)
}

func setPayloadEnv(cmd *exec.Cmd, hostPath string) {
	setProcessEnv(cmd, envSuperplanePayloadFile, hostPath)
}

func setProcessEnv(cmd *exec.Cmd, name, hostPath string) {
	if cmd == nil || strings.TrimSpace(hostPath) == "" {
		return
	}
	pair := name + "=" + strings.TrimSpace(hostPath)
	if len(cmd.Env) > 0 {
		cmd.Env = append(cmd.Env, pair)
		return
	}
	cmd.Env = append(os.Environ(), pair)
}

func readTaskResultFile(path string, maxBytes int, log *slog.Logger) json.RawMessage {
	b, err := os.ReadFile(path)
	if err != nil || len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	b = bytes.TrimSpace(b)
	if maxBytes > 0 && len(b) > maxBytes {
		if log != nil {
			log.Warn("task_result_file_too_large",
				slog.String("path", path), slog.Int("bytes", len(b)), slog.Int("max", maxBytes))
		}
		return nil
	}
	if !json.Valid(b) {
		if log != nil {
			log.Warn("task_result_file_invalid_json", slog.String("path", path))
		}
		return nil
	}
	out := make(json.RawMessage, len(b))
	copy(out, b)
	return out
}

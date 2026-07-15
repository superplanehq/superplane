package ssh

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// sshConnectTimeoutSeconds bounds each individual TCP/handshake attempt. It is
// intentionally short so a stalled connect fails fast and the connection-retry
// loop can take over, instead of hanging until the whole task times out.
const sshConnectTimeoutSeconds = 15

// Reserved environment variables carrying the SSH credentials onto the runner.
// They are set on the broker task (never persisted in execution metadata) and
// consumed only by the generated wrapper script.
const (
	envPrivateKey = "SUPERPLANE_SSH_PRIVATE_KEY"
	envPassphrase = "SUPERPLANE_SSH_PASSPHRASE"
	envPassword   = "SUPERPLANE_SSH_PASSWORD"
)

// buildRemoteScript assembles the script executed on the remote host. Environment
// variables and the working-directory change are prepended so they apply to the
// whole script, and `set -e` makes a failing line abort the run with its exit
// code (mirroring the previous `&&`-chained behavior for multi-line inputs).
func buildRemoteScript(environment []EnvironmentVariable, workingDirectory, body string) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	for _, variable := range environment {
		b.WriteString(fmt.Sprintf("export %s=%s\n", variable.Name, shellQuote(variable.Value)))
	}
	if strings.TrimSpace(workingDirectory) != "" {
		b.WriteString(fmt.Sprintf("cd %s || exit 1\n", shellQuote(workingDirectory)))
	}
	script := normalizeScriptLineEndings(body)
	b.WriteString(script)
	if !strings.HasSuffix(script, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

// buildRunnerScript generates the Bash script the runner executes. It opens an
// SSH connection to the remote host, streams the remote script over stdin to a
// remote `bash -s`, and applies the configured connection/execution retries —
// all on the runner so stdout/stderr flow to live logs and there is no
// control-plane time limit.
func buildRunnerScript(spec Spec, remoteScript string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(remoteScript))

	connectRetries, connectInterval := retryValues(spec.ConnectionRetry)
	execRetries, execInterval := retryValues(spec.ExecutionRetry)

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -uo pipefail\n\n")

	if spec.Authentication.Method == AuthMethodSSHKey {
		b.WriteString(keyFileSetup())
	}

	b.WriteString("remote_script() {\n")
	b.WriteString("  base64 -d <<'SUPERPLANE_SSH_SCRIPT_EOF'\n")
	b.WriteString(encoded)
	b.WriteString("\nSUPERPLANE_SSH_SCRIPT_EOF\n")
	b.WriteString("}\n\n")

	b.WriteString("run_once() {\n")
	b.WriteString("  remote_script | " + sshInvocation(spec) + "\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "connect_retries=%d\n", connectRetries)
	fmt.Fprintf(&b, "connect_interval=%d\n", connectInterval)
	fmt.Fprintf(&b, "exec_retries=%d\n", execRetries)
	fmt.Fprintf(&b, "exec_interval=%d\n", execInterval)
	b.WriteString("connect_attempt=0\n")
	b.WriteString("exec_attempt=0\n")
	b.WriteString("code=0\n\n")

	b.WriteString("while true; do\n")
	b.WriteString("  run_once\n")
	b.WriteString("  code=$?\n")
	b.WriteString("  if [ \"$code\" -eq 255 ] && [ \"$connect_attempt\" -lt \"$connect_retries\" ]; then\n")
	b.WriteString("    connect_attempt=$((connect_attempt + 1))\n")
	b.WriteString("    echo \"superplane: ssh connection failed (exit 255), retry $connect_attempt/$connect_retries in ${connect_interval}s\" >&2\n")
	b.WriteString("    sleep \"$connect_interval\"\n")
	b.WriteString("    continue\n")
	b.WriteString("  fi\n")
	b.WriteString("  if [ \"$code\" -ne 0 ] && [ \"$code\" -ne 255 ] && [ \"$exec_attempt\" -lt \"$exec_retries\" ]; then\n")
	b.WriteString("    exec_attempt=$((exec_attempt + 1))\n")
	b.WriteString("    echo \"superplane: command exited $code, retry $exec_attempt/$exec_retries in ${exec_interval}s\" >&2\n")
	b.WriteString("    sleep \"$exec_interval\"\n")
	b.WriteString("    continue\n")
	b.WriteString("  fi\n")
	b.WriteString("  break\n")
	b.WriteString("done\n\n")
	b.WriteString("exit \"$code\"\n")

	return b.String()
}

func keyFileSetup() string {
	var b strings.Builder
	b.WriteString("key_file=\"$(mktemp)\"\n")
	b.WriteString("trap 'rm -f \"$key_file\"' EXIT\n")
	b.WriteString("chmod 600 \"$key_file\"\n")
	fmt.Fprintf(&b, "printf '%%s' \"${%s:-}\" > \"$key_file\"\n\n", envPrivateKey)
	return b.String()
}

// sshInvocation returns the shell snippet that runs `ssh ... bash -s`, wired up
// for the configured authentication method. Auth secrets are read from the
// reserved environment variables set on the runner task.
func sshInvocation(spec Spec) string {
	opts := fmt.Sprintf(
		"-p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=%d",
		sshPort(spec.Port), sshConnectTimeoutSeconds,
	)
	target := fmt.Sprintf("%s@%s", shellQuote(spec.User), shellQuote(spec.Host))

	if spec.Authentication.Method == AuthMethodPassword {
		return fmt.Sprintf(
			"SSHPASS=\"${%s:-}\" sshpass -e ssh -o PubkeyAuthentication=no %s %s bash -s",
			envPassword, opts, target,
		)
	}

	// SSH key auth. A passphrase-protected key is fed through sshpass answering
	// the "Enter passphrase" prompt; otherwise BatchMode avoids any interactive
	// hang when the key is rejected.
	if spec.Authentication.Passphrase.IsSet() {
		return fmt.Sprintf(
			"SSHPASS=\"${%s:-}\" sshpass -P assphrase -e ssh -i \"$key_file\" %s %s bash -s",
			envPassphrase, opts, target,
		)
	}

	return fmt.Sprintf("ssh -i \"$key_file\" -o BatchMode=yes %s %s bash -s", opts, target)
}

func sshPort(port int) int {
	if port <= 0 {
		return 22
	}
	return port
}

// retryValues returns the retry count and interval a retry spec should apply,
// collapsing a disabled or nil spec to zero retries.
func retryValues(spec *RetrySpec) (retries int, intervalSeconds int) {
	if spec == nil || !spec.Enabled {
		return 0, 0
	}
	interval := spec.IntervalSeconds
	if interval < 1 {
		interval = 1
	}
	if spec.Retries < 0 {
		return 0, interval
	}
	return spec.Retries, interval
}

func normalizeScriptLineEndings(script string) string {
	script = strings.ReplaceAll(script, "\r\n", "\n")
	return strings.ReplaceAll(script, "\r", "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

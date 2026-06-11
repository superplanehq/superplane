package ansible

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	binaryPlaybook = "ansible-playbook"
	binaryAdhoc    = "ansible"
)

// RecapStats mirrors the per-host counters Ansible reports in its play recap.
type RecapStats struct {
	Ok          int `json:"ok"`
	Changed     int `json:"changed"`
	Unreachable int `json:"unreachable"`
	Failures    int `json:"failures"`
	Skipped     int `json:"skipped"`
	Rescued     int `json:"rescued"`
	Ignored     int `json:"ignored"`
}

// RunResult is the outcome of a single Ansible invocation. It is the payload
// data emitted downstream, so it must not contain any inventory secrets.
type RunResult struct {
	ExitCode int                   `json:"exitCode"`
	Stdout   string                `json:"stdout"`
	Stderr   string                `json:"stderr"`
	Recap    map[string]RecapStats `json:"recap,omitempty"`
}

// ansibleRunner runs an Ansible invocation for a decoded Spec. It is an
// interface so Execute can be unit-tested with a stub that returns a fixed
// RunResult without a real ansible binary on the machine.
type ansibleRunner interface {
	Run(ctx context.Context, spec Spec, logger *log.Entry) (*RunResult, error)
}

// execRunner is the production runner: it writes the inventory (and playbook)
// to a temporary directory and shells out to the ansible binaries.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, spec Spec, logger *log.Entry) (*RunResult, error) {
	workDir, err := os.MkdirTemp("", "superplane-ansible-")
	if err != nil {
		return nil, fmt.Errorf("could not create working directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	inventoryPath := filepath.Join(workDir, "inventory")
	if err := os.WriteFile(inventoryPath, []byte(spec.Inventory), 0o600); err != nil {
		return nil, fmt.Errorf("could not write inventory: %w", err)
	}

	var binary string
	var args []string
	switch spec.Mode {
	case ModeAdhoc:
		binary = binaryAdhoc
		args = buildAdhocArgs(inventoryPath, spec)
	default:
		binary = binaryPlaybook
		playbookPath := filepath.Join(workDir, "playbook.yml")
		content := ""
		if spec.Playbook != nil {
			content = *spec.Playbook
		}
		if err := os.WriteFile(playbookPath, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("could not write playbook: %w", err)
		}
		args = buildPlaybookArgs(playbookPath, inventoryPath, spec)
	}

	if logger != nil {
		logger.Infof("Running %s %s", binary, strings.Join(args, " "))
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = ansibleEnv(spec.Mode)

	runErr := cmd.Run()

	// A deadline/cancellation means the component could not complete: surface
	// it as an execution error (error state), not a failed outcome.
	if ctx.Err() != nil {
		return nil, fmt.Errorf("ansible run timed out after %d seconds", spec.Timeout)
	}

	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			// Ansible ran and reported a non-zero status (e.g. a failed task);
			// this is a "failed" outcome, not an execution error.
			exitCode = exitErr.ExitCode()
		} else {
			// Could not start the process at all (e.g. binary missing).
			return nil, fmt.Errorf("could not run %s: %w", binary, runErr)
		}
	}

	result := &RunResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
	if spec.Mode != ModeAdhoc {
		result.Recap = parseRecap(stdout.Bytes())
	}

	return result, nil
}

// ansibleEnv returns the environment for an Ansible invocation. Host key
// checking is disabled so first-contact SSH does not block on a prompt, and
// the JSON stdout callback is used for playbooks so the recap can be parsed.
func ansibleEnv(mode string) []string {
	env := append(os.Environ(),
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"ANSIBLE_FORCE_COLOR=0",
	)
	if mode != ModeAdhoc {
		env = append(env, "ANSIBLE_STDOUT_CALLBACK=json")
	}
	return env
}

// buildPlaybookArgs builds the argv for `ansible-playbook`. Every value is a
// distinct argv element (no shell is involved), so values cannot break out
// into additional commands.
func buildPlaybookArgs(playbookPath, inventoryPath string, spec Spec) []string {
	args := []string{"-i", inventoryPath}
	args = append(args, commonArgs(spec)...)
	args = append(args, playbookPath)
	return args
}

// buildAdhocArgs builds the argv for `ansible <pattern> -m <module> -a <args>`.
func buildAdhocArgs(inventoryPath string, spec Spec) []string {
	module := ModuleDefault
	if spec.Module != nil && *spec.Module != "" {
		module = *spec.Module
	}

	pattern := ""
	if spec.HostPattern != nil {
		pattern = *spec.HostPattern
	}

	args := []string{pattern, "-i", inventoryPath, "-m", module}
	if spec.ModuleArgs != nil && *spec.ModuleArgs != "" {
		args = append(args, "-a", *spec.ModuleArgs)
	}
	args = append(args, commonArgs(spec)...)
	return args
}

// commonArgs builds the flags shared by both modes (limit, become, verbosity,
// extra vars).
func commonArgs(spec Spec) []string {
	args := []string{}
	if spec.Limit != nil && *spec.Limit != "" {
		args = append(args, "--limit", *spec.Limit)
	}
	if spec.Become {
		args = append(args, "--become")
	}
	if flag := verbosityFlag(spec.Verbosity); flag != "" {
		args = append(args, flag)
	}
	args = append(args, extraVarArgs(spec.ExtraVars)...)
	return args
}

// verbosityFlag maps a 0-4 verbosity level to the matching -v..-vvvv flag.
func verbosityFlag(level int) string {
	if level <= 0 {
		return ""
	}
	if level > 4 {
		level = 4
	}
	return "-" + strings.Repeat("v", level)
}

// extraVarArgs renders extra variables as repeated `-e name=value` argv pairs.
func extraVarArgs(vars []ExtraVar) []string {
	args := []string{}
	for _, v := range vars {
		if v.Name == "" {
			continue
		}
		args = append(args, "-e", fmt.Sprintf("%s=%s", v.Name, v.Value))
	}
	return args
}

// parseRecap extracts the per-host stats from the JSON stdout callback output.
// It is best-effort: if the output is not the expected JSON (e.g. a different
// callback is configured), it returns nil and the raw stdout is still emitted.
func parseRecap(stdout []byte) map[string]RecapStats {
	var payload struct {
		Stats map[string]RecapStats `json:"stats"`
	}
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return nil
	}
	return payload.Stats
}

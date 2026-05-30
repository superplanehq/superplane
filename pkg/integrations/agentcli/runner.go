package agentcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type Command struct {
	Name    string
	Args    []string
	Dir     string
	Env     map[string]string
	Timeout time.Duration
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Duration time.Duration
}

type Runner interface {
	Run(ctx context.Context, command Command) (Result, error)
}

type OSRunner struct{}

func (r OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	if command.Name == "" {
		return Result{}, fmt.Errorf("command name is required")
	}

	if command.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, command.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Env = commandEnvironment(command.Env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startedAt := time.Now()
	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(startedAt),
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		result.ExitCode = -1
		return result, nil
	}

	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, err
}

func commandEnvironment(overrides map[string]string) []string {
	env := os.Environ()
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

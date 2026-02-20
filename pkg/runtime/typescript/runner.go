package typescript

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultDenoBinary           = "deno"
	DefaultDenoExecutionTimeout = 30 * time.Second
)

func ExecuteComponentEntrypoint(
	binary string,
	timeout time.Duration,
	entrypoint string,
	request ComponentExecutionRequest,
) (*ComponentExecutionResponse, error) {
	if binary == "" {
		binary = DefaultDenoBinary
	}
	if timeout <= 0 {
		timeout = DefaultDenoExecutionTimeout
	}

	input, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime request: %w", err)
	}

	runCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(
		runCtx,
		binary,
		"run",
		"--quiet",
		"--no-prompt",
		"--allow-net",
		entrypoint,
	)
	cmd.Stdin = bytes.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("deno execution timed out")
		}

		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("deno execution failed: %s", message)
	}

	var response ComponentExecutionResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("invalid deno runtime output")
	}

	if err := response.Validate(); err != nil {
		return nil, err
	}

	return &response, nil
}

func ExecuteIntegrationEntrypoint(
	binary string,
	timeout time.Duration,
	entrypoint string,
	request IntegrationRuntimeRequest,
) (*IntegrationRuntimeResponse, error) {
	if binary == "" {
		binary = DefaultDenoBinary
	}
	if timeout <= 0 {
		timeout = DefaultDenoExecutionTimeout
	}

	input, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime request: %w", err)
	}

	runCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(
		runCtx,
		binary,
		"run",
		"--quiet",
		"--no-prompt",
		"--allow-net",
		entrypoint,
	)
	cmd.Stdin = bytes.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("deno execution timed out")
		}

		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("deno execution failed: %s", message)
	}

	var response IntegrationRuntimeResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("invalid deno runtime output")
	}

	if err := response.Validate(); err != nil {
		return nil, err
	}

	return &response, nil
}

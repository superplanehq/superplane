package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/runtime/runner"
)

func newTypeScriptRunner() (runner.Client, runner.Config, error) {
	cfg, err := runner.LoadConfigFromEnv()
	if err != nil {
		return nil, runner.Config{}, err
	}

	client, err := runner.NewClient(cfg)
	if err != nil {
		return nil, runner.Config{}, err
	}

	return client, cfg, nil
}

func newTypeScriptRunnerRequest(version string, timeout time.Duration, context runner.RuntimeContext, input any) runner.OperationRequest {
	return runner.OperationRequest{
		Request: runner.RequestEnvelope{
			RequestID: uuid.NewString(),
			Version:   version,
			TimeoutMS: timeout.Milliseconds(),
		},
		Context: context,
		Input:   input,
	}
}

func decodeTypeScriptRunnerOutput(output map[string]any, target any) error {
	if len(output) == 0 {
		return fmt.Errorf("runtime runner returned empty output")
	}

	data, err := json.Marshal(output)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

func typeScriptRunnerResponseError(response *runner.OperationResponse, fallback string) error {
	if response == nil {
		return fmt.Errorf("%s", fallback)
	}

	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return fmt.Errorf("%s", strings.TrimSpace(response.Error.Message))
	}

	return fmt.Errorf("%s", fallback)
}

func withTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(context.Background())
	}

	return context.WithTimeout(context.Background(), timeout)
}

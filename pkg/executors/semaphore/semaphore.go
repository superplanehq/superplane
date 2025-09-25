package semaphore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/executors"
)

type SemaphoreExecutor struct{}

func NewSemaphoreExecutor() executors.Executor {
	return &SemaphoreExecutor{}
}

type SemaphoreSpec struct {
	Task         string            `json:"task"`
	Ref          string            `json:"ref"`
	PipelineFile string            `json:"pipelineFile"`
	Parameters   map[string]string `json:"parameters"`
}

func (e *SemaphoreExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec SemaphoreSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.Ref == "" {
		return fmt.Errorf("ref is required")
	}

	return nil
}

func (e *SemaphoreExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (executors.Response, error) {
	var spec SemaphoreSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	// For standalone semaphore executor, this is a placeholder implementation
	// In a real scenario, this would interact with Semaphore CI directly
	return &SemaphoreResponse{
		successful: true,
		outputs:    map[string]any{},
	}, nil
}

type SemaphoreResponse struct {
	successful bool
	outputs    map[string]any
}

func (r *SemaphoreResponse) Successful() bool {
	return r.successful
}

func (r *SemaphoreResponse) Outputs() map[string]any {
	return r.outputs
}
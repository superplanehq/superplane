package noop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/manifest"
)

type NoOpExecutor struct{}

func NewNoOpExecutor() executors.Executor {
	return &NoOpExecutor{}
}

type NoOpSpec struct{}

func (e *NoOpExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec NoOpSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	return nil
}

func (e *NoOpExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (executors.Response, error) {
	var spec NoOpSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	outputMap := map[string]any{}
	for _, outputName := range parameters.OutputNames {
		v, err := crypto.Base64String(8)
		if err != nil {
			return nil, fmt.Errorf("error generating random output value: %v", err)
		}

		outputMap[outputName] = v
	}

	return &NoOpResponse{OutputMap: outputMap}, nil
}

type NoOpResponse struct {
	OutputMap map[string]any
}

func (r *NoOpResponse) Successful() bool {
	return true
}

func (r *NoOpResponse) Outputs() map[string]any {
	return r.OutputMap
}

func (e *NoOpExecutor) Manifest() *manifest.TypeManifest {
	return &manifest.TypeManifest{
		Type:        "noop",
		DisplayName: "No-Op",
		Description: "A test executor that does nothing and always succeeds",
		Category:    "executor",
		Icon:        "noop",
		Fields:      []manifest.FieldManifest{},
	}
}

package executors

import (
	"context"

	"github.com/superplanehq/superplane/pkg/manifest"
)

type Executor interface {
	Validate(context.Context, []byte) error
	Execute([]byte, ExecutionParameters) (Response, error)
	Manifest() *manifest.TypeManifest
}

type ExecutionParameters struct {
	ExecutionID string
	StageID     string
	Token       string
	OutputNames []string
}

type Response interface {
	Successful() bool
	Outputs() map[string]any
}

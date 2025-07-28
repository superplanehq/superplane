package executors

import (
	"context"
)

type Executor interface {
	Validate(context.Context, []byte) error
	Execute([]byte, ExecutionParameters) (Response, error)
}

type ExecutionParameters struct {
	ExecutionID string
	StageID     string
	Token       string
}

type Response interface {
	Finished() bool
	Successful() bool
	Outputs() map[string]any
	Id() string
}

package executors

import (
	"context"
	"regexp"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type Executor interface {
	Validate(context.Context, []byte) error
	Execute([]byte, ExecutionParameters) (Response, error)
	Check(string) (Response, error)
	HandleWebhook([]byte) (Response, error)
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

package executors

import (
	"context"
	"regexp"

	"github.com/superplanehq/superplane/pkg/integrations"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type BuildFn func(integrations.Integration, integrations.Resource) (Executor, error)

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

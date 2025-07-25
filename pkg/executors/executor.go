package executors

import (
	"regexp"

	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type Executor interface {
	Execute(models.ExecutorSpec, ExecutionParameters) (Response, error)
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

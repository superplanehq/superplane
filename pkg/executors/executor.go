package executors

import (
	"fmt"
	"regexp"

	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type Executor interface {
	Name() string
	Execute(spec models.ExecutorSpec, params ExecutionParameters) (Response, error)
	Check(id string) (Response, error)
	HandleWebhook(hook []byte) (Response, error)
}

type Response interface {
	Finished() bool
	Successful() bool
	Outputs() map[string]any
	Id() string
}

type ExecutionParameters struct {
	ExecutionID string
	StageID     string
	Token       string
}

func NewExecutorWithIntegration(integration integrations.Integration, resource integrations.Resource, executor *models.StageExecutor) (Executor, error) {
	switch executor.Type {
	case models.ExecutorSpecTypeSemaphore:
		return NewSemaphoreExecutor(integration, resource)

	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor()

	default:
		return nil, fmt.Errorf("executor type %s not supported", executor.Type)
	}
}

func NewExecutorWithoutIntegration(executor *models.StageExecutor) (Executor, error) {
	switch executor.Type {
	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor()

	default:
		return nil, fmt.Errorf("executor type %s not supported without integration", executor.Type)
	}
}

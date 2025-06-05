package executors

import (
	"fmt"
	"regexp"

	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type Executor interface {
	Name() string
	Execute(models.ExecutorSpec) (Response, error)
	Check(models.ExecutorSpec, string) (Response, error)
}

type Response interface {
	Finished() bool
	Successful() bool
	Outputs() map[string]any
	Id() string
}

func NewExecutor(specType string, execution models.StageExecution, jwtSigner *jwt.Signer) (Executor, error) {
	switch specType {
	case models.ExecutorSpecTypeSemaphore:
		return NewSemaphoreExecutor(execution, jwtSigner)
	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor(execution, jwtSigner)
	default:
		return nil, fmt.Errorf("executor type %s not supported", specType)
	}
}

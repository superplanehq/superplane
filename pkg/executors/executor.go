package executors

import (
	"context"
	"fmt"
	"regexp"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
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

func NewExecutor(executor *models.StageExecutor, execution models.StageExecution, jwtSigner *jwt.Signer, encryptor crypto.Encryptor) (Executor, error) {
	switch executor.Type {
	case models.ExecutorSpecTypeSemaphore:
		r, err := executor.FindIntegration()
		if err != nil {
			return nil, fmt.Errorf("error finding integration: %v", err)
		}

		integration, err := integrations.NewIntegration(context.Background(), r, encryptor)
		if err != nil {
			return nil, fmt.Errorf("error creating integration: %v", err)
		}

		return NewSemaphoreExecutor(integration, execution, jwtSigner)

	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor(execution, jwtSigner)

	default:
		return nil, fmt.Errorf("executor type %s not supported", executor.Type)
	}
}

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
	Execute(spec models.ExecutorSpec, resource integrations.Resource) (Response, error)
	Check(id string) (Response, error)
	HandleWebhook(hook []byte) (Response, error)
}

type Response interface {
	Finished() bool
	Successful() bool
	Outputs() map[string]any
	Id() string
}

func NewExecutor(integration *models.Integration, executor *models.StageExecutor, execution *models.StageExecution, jwtSigner *jwt.Signer, encryptor crypto.Encryptor) (Executor, error) {
	switch executor.Type {
	case models.ExecutorSpecTypeSemaphore:
		integrationImpl, err := integrations.NewIntegration(context.Background(), integration, encryptor)
		if err != nil {
			return nil, fmt.Errorf("error creating integration: %v", err)
		}

		return NewSemaphoreExecutor(integrationImpl, execution, jwtSigner)

	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor(execution, jwtSigner)

	default:
		return nil, fmt.Errorf("executor type %s not supported", executor.Type)
	}
}

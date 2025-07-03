package executors

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
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

func NewExecutor(spec models.ExecutorSpec, execution models.StageExecution, jwtSigner *jwt.Signer, encryptor crypto.Encryptor) (Executor, error) {
	switch spec.Type {
	case models.ExecutorSpecTypeSemaphore:
		//
		// If no integration is used for the executor,
		// we create one from the spec itself.
		//
		if spec.Integration == nil {
			integration, err := integrations.NewSemaphoreIntegration(spec.Semaphore.OrganizationURL, spec.Semaphore.APIToken)
			if err != nil {
				return nil, err
			}

			return NewSemaphoreExecutor(integration, execution, jwtSigner)
		}

		//
		// If an integration is used for the executor,
		// we use it to initialize our executor.
		//
		i, err := models.FindIntegrationByName(spec.Integration.DomainType, uuid.MustParse(spec.Integration.DomainID), *spec.Integration.Name)
		if err != nil {
			return nil, fmt.Errorf("error finding integration: %v", err)
		}

		integration, err := integrations.NewIntegration(context.Background(), i, encryptor)
		if err != nil {
			return nil, fmt.Errorf("error creating integration: %v", err)
		}

		return NewSemaphoreExecutor(integration, execution, jwtSigner)

	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor(execution, jwtSigner)

	default:
		return nil, fmt.Errorf("executor type %s not supported", spec.Type)
	}
}

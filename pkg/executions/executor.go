package executions

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type Executor interface {
	Execute() (Resource, error)
	AsyncCheck(string) (Status, error)
}

type Resource interface {

	//
	// Whether the final result of the execution is async or sync.
	//
	Async() bool

	//
	// For async executions, we need an identifier to monitor its status.
	//
	AsyncId() string

	//
	// Used for async resources.
	//
	Check() (Status, error)
}

type Status interface {
	Finished() bool
	Successful() bool
}

func NewExecutor(execution models.StageExecution, runTemplate models.RunTemplate, encryptor encryptor.Encryptor, jwtSigner *jwt.Signer) (Executor, error) {
	resolver := NewTemplateResolver(execution, runTemplate)
	template, err := resolver.Resolve()
	if err != nil {
		return nil, fmt.Errorf("error resolving run template: %v", err)
	}

	switch template.Type {
	case models.RunTemplateTypeSemaphore:
		return NewSemaphoreExecutor(execution, template.Semaphore, encryptor, jwtSigner)
	case models.RunTemplateTypeHTTP:
		return NewHTTPExecutor(execution, template.HTTP, encryptor, jwtSigner)
	default:
		return nil, fmt.Errorf("executor type %s not supported", template.Type)
	}
}

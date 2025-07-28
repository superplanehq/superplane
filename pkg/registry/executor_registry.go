package registry

import (
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/executors/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutorRegistry struct {
	Executors map[string]executors.BuildFn
	Encryptor crypto.Encryptor
}

func NewExecutorRegistry(encryptor crypto.Encryptor) *ExecutorRegistry {
	r := &ExecutorRegistry{
		Encryptor: encryptor,
		Executors: map[string]executors.BuildFn{},
	}

	r.Init()
	return r
}

func (r *ExecutorRegistry) Init() {
	r.Executors[models.IntegrationTypeSemaphore] = semaphore.NewSemaphoreExecutor
}

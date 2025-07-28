package registry

import (
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/executors/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutorRegistry struct {
	Executors map[string]executors.BuildFn
}

func NewExecutorRegistry() *ExecutorRegistry {
	r := &ExecutorRegistry{
		Executors: map[string]executors.BuildFn{},
	}

	r.Init()
	return r
}

func (r *ExecutorRegistry) Init() {
	r.Executors[models.IntegrationTypeSemaphore] = semaphore.NewSemaphoreExecutor
}

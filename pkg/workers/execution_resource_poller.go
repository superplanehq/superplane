package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ExecutionResourcePoller struct {
	Encryptor crypto.Encryptor
	Registry  *registry.Registry
}

func NewExecutionResourcePoller(encryptor crypto.Encryptor, registry *registry.Registry) *ExecutionResourcePoller {
	return &ExecutionResourcePoller{
		Encryptor: encryptor,
		Registry:  registry,
	}
}

func (w *ExecutionResourcePoller) Start() error {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing started executions: %v", err)
		}

		time.Sleep(15 * time.Minute)
	}
}

func (w *ExecutionResourcePoller) Tick() error {
	resources, err := models.PendingExecutionResources()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		err := w.ProcessResource(resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *ExecutionResourcePoller) ProcessResource(resource models.ExecutionResource) error {
	integration, err := resource.FindIntegration()
	if err != nil {
		return err
	}

	integrationImpl, err := w.Registry.NewIntegration(context.Background(), integration)
	if err != nil {
		return err
	}

	statefulResource, err := integrationImpl.Check(resource.Type, resource.ExternalID)
	if err != nil {
		return err
	}

	if !statefulResource.Finished() {
		log.Infof("Execution resource %s is not finished yet", resource.ExternalID)
		return nil
	}

	result := models.ResultPassed
	if !statefulResource.Successful() {
		result = models.ResultFailed
	}

	return resource.Finish(result)
}

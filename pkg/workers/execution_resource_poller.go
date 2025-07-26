package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutionResourcePoller struct {
	Encryptor crypto.Encryptor
}

func NewExecutionResourcePoller(encryptor crypto.Encryptor) *ExecutionResourcePoller {
	return &ExecutionResourcePoller{
		Encryptor: encryptor,
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
	stage, err := models.FindStageByID(resource.StageID.String())
	if err != nil {
		return err
	}

	stageExecutor, err := stage.GetExecutor()
	if err != nil {
		return err
	}

	executor, err := w.initExecutor(stageExecutor)
	if err != nil {
		return err
	}

	status, err := executor.Check(resource.ExternalID)
	if err != nil {
		return err
	}

	if !status.Finished() {
		log.Infof("Execution resource %s is not finished yet", resource.ExternalID)
		return nil
	}

	result := models.ResultPassed
	if !status.Successful() {
		result = models.ResultFailed
	}

	return resource.Finish(result)
}

func (w *ExecutionResourcePoller) initExecutor(stageExecutor *models.StageExecutor) (executors.Executor, error) {
	if stageExecutor.ResourceID == nil {
		return executors.NewExecutor(stageExecutor.Type, nil, nil, w.Encryptor)
	}

	integration, err := stageExecutor.FindIntegration()
	if err != nil {
		return nil, fmt.Errorf("error finding integration for stage executor: %v", err)
	}

	resource, err := stageExecutor.GetResource()
	if err != nil {
		return nil, fmt.Errorf("error finding resource for stage executor: %v", err)
	}

	return executors.NewExecutor(stageExecutor.Type, integration, resource, w.Encryptor)
}

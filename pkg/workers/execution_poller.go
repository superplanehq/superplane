package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	DefaultExecutionTimeout          = 3 * time.Hour
	ExecutionResourcePollingInterval = 5 * time.Minute
)

type ExecutionPoller struct {
	Encryptor        crypto.Encryptor
	Registry         *registry.Registry
	ExecutionTimeout time.Duration
	nowFunc          func() time.Time
}

func NewExecutionPoller(encryptor crypto.Encryptor, registry *registry.Registry) *ExecutionPoller {
	return &ExecutionPoller{
		Encryptor:        encryptor,
		Registry:         registry,
		ExecutionTimeout: DefaultExecutionTimeout,
		nowFunc:          time.Now,
	}
}

func (w *ExecutionPoller) Start() error {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing started executions: %v", err)
		}

		time.Sleep(time.Second)
	}
}

func (w *ExecutionPoller) Tick() error {
	executions, err := models.ListExecutionsInState(models.ExecutionStarted)
	if err != nil {
		return err
	}

	for _, execution := range executions {
		e := execution
		logger := logging.ForExecution(&e)
		err := w.ProcessExecution(logger, &e)
		if err != nil {
			logger.Errorf("Error processing execution: %v", err)
		}
	}

	return nil
}

func (w *ExecutionPoller) ProcessExecution(logger *log.Entry, execution *models.StageExecution) error {
	//
	// Handle execution cancellation.
	//
	if execution.CancelledAt != nil {
		logger.Info("Execution cancelled")
		return w.CancelExecution(logger, execution)
	}

	//
	// Handle execution timeouts.
	//
	if execution.IsTimedOut(w.nowFunc(), w.ExecutionTimeout) {
		logger.Info("Execution timed out")
		return w.CancelExecution(logger, execution)
	}

	//
	// Otherwise, check the state of the execution resources.
	//
	return w.CheckExecutionStatus(logger, execution)
}

func (w *ExecutionPoller) CheckExecutionStatus(logger *log.Entry, execution *models.StageExecution) error {
	resources, err := execution.Resources()
	if err != nil {
		return err
	}

	stage, err := models.FindStageByID(execution.CanvasID.String(), execution.StageID.String())
	if err != nil {
		return err
	}

	//
	// This is not a valid state.
	// If we have an execution in the started state, but no execution resources,
	// something went wrong, just finish the execution with an error.
	//
	if len(resources) == 0 {
		return w.finishExecution(logger, stage, execution, models.ResultFailed)
	}

	//
	// Poll the execution resources statuses, updating them in the database.
	//
	err = w.updateResourceStatuses(logger, resources)
	if err != nil {
		return err
	}

	//
	// If any of the resources is still not finished, return.
	//
	if !execution.Finished(resources) {
		return nil
	}

	result := execution.GetResult(stage, resources)
	return w.finishExecution(logger, stage, execution, result)
}

func (w *ExecutionPoller) updateResourceStatuses(logger *log.Entry, resources []models.ExecutionResource) error {
	for _, resource := range resources {
		err := w.updateResourceStatus(logger, &resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *ExecutionPoller) finishExecution(logger *log.Entry, stage *models.Stage, execution *models.StageExecution, result string) error {
	event, err := execution.Finish(stage, result)
	if err != nil {
		logger.Errorf("Error finishing execution: %v", err)
		return err
	}

	err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), execution).Publish()
	if err != nil {
		logger.Errorf("Error publishing execution finished message: %v", err)
	}

	err = messages.NewEventCreatedMessage(stage.CanvasID.String(), event).Publish()
	if err != nil {
		logger.Errorf("Error publishing event created message: %v", err)
	}

	return nil
}

func (w *ExecutionPoller) CancelExecution(logger *log.Entry, execution *models.StageExecution) error {
	resources, err := execution.Resources()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		err = w.cancelResource(logger, resource)
		if err != nil {
			return err
		}
	}

	stage, err := models.FindStageByID(execution.CanvasID.String(), execution.StageID.String())
	if err != nil {
		return err
	}

	return w.finishExecution(logger, stage, execution, models.ResultCancelled)
}

func (w *ExecutionPoller) cancelResource(logger *log.Entry, resource models.ExecutionResource) error {
	integration, err := resource.FindIntegration()
	if err != nil {
		return err
	}

	integrationImpl, err := w.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return err
	}

	parentResource, err := resource.FindParentResource()
	if err != nil {
		return err
	}

	if resource.Finished() {
		logger.Infof("Resource %s already finished", resource.Id())
		return nil
	}

	err = integrationImpl.Cancel(resource.Type(), resource.Id(), parentResource)
	if err != nil {
		logger.Errorf("Error canceling resource %s: %v", resource.Id(), err)
		return resource.Finish(models.ResultCancelled)
	}

	logger.Infof("Resource %s canceled", resource.Id())
	return resource.Finish(models.ResultCancelled)
}

func (w *ExecutionPoller) updateResourceStatus(logger *log.Entry, resource *models.ExecutionResource) error {
	//
	// If the resource is already finished, we don't need to check it again.
	//
	if resource.Finished() {
		logger.Infof("Resource %s already finished", resource.Id())
		return nil
	}

	//
	// We don't poll on every iteration to avoid rate limiting issues with third-party APIs.
	// If we shouldn't check the status yet, just return what we currently have.
	//
	if !resource.ShouldPoll(ExecutionResourcePollingInterval) {
		return nil
	}

	integration, err := resource.FindIntegration()
	if err != nil {
		return err
	}

	integrationImpl, err := w.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return err
	}

	parentResource, err := resource.FindParentResource()
	if err != nil {
		return err
	}

	updatedResource, err := integrationImpl.Status(resource.Type(), resource.Id(), parentResource)
	if err != nil {
		return err
	}

	//
	// Resource is not finished yet, no need to update anything in the database.
	//
	if !resource.Finished() {
		logger.Infof("Resource %s not finished yet", resource.Id())
		return resource.UpdatePollingTimestamp()
	}

	//
	// Resource is finished, update the database.
	//
	if updatedResource.Successful() {
		return resource.Finish(models.ResultPassed)
	}

	return resource.Finish(models.ResultFailed)
}

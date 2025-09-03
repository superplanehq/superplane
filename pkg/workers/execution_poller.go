package workers

import (
	"context"
	"math"
	"slices"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const (
	DefaultExecutionTimeout      = 30 * time.Minute
	DefaultExecutionFinalTimeout = 3 * time.Hour
	MaxRetryAttempts             = 5
	BaseRetryDelay               = 30 * time.Second
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

		if w.isExecutionStuck(&e) {
			logger.Warnf("Execution has been running for %v, attempting status polling", w.nowFunc().Sub(*e.StartedAt))
			err := w.ProcessStuckExecution(logger, &e)
			if err != nil {
				logger.Errorf("Error processing stuck execution: %v", err)
				continue
			}
		}

		err := w.ProcessExecution(logger, &e)
		logger.Errorf("Error processing execution: %v", err)
	}

	return nil
}

func (w *ExecutionPoller) ProcessExecution(logger *log.Entry, execution *models.StageExecution) error {
	stage, err := models.FindStageByID(execution.CanvasID.String(), execution.StageID.String())
	if err != nil {
		return err
	}

	resources, err := execution.Resources()
	if err != nil {
		return err
	}

	//
	// If the execution still has resources to finish, skip.
	//
	if slices.ContainsFunc(resources, func(resource models.ExecutionResource) bool {
		return resource.State == models.ExecutionResourcePending
	}) {
		return nil
	}

	//
	// If any resource failed, mark the execution as failed.
	//
	result := models.ResultPassed
	if slices.ContainsFunc(resources, func(resource models.ExecutionResource) bool {
		return resource.Result == models.ResultFailed
	}) {
		result = models.ResultFailed
	}

	var event *models.Event
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		outputs := execution.Outputs.Data()

		//
		// Check if all required outputs were pushed.
		// If any output wasn't pushed, mark the execution as failed.
		//
		missingOutputs := stage.MissingRequiredOutputs(outputs)
		if len(missingOutputs) > 0 {
			logger.Infof("Missing outputs %v - marking the execution as failed", missingOutputs)
			result = models.ResultFailed
		}

		if event, err = execution.FinishInTransaction(tx, stage, result); err != nil {
			logger.Errorf("Error updating execution state: %v", err)
			return err
		}

		logger.Infof("Execution state updated: %s", result)
		return nil
	})

	if err != nil {
		return err
	}

	logger.Infof("Finished with result: %s", result)
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

// isExecutionStuck checks if an execution has been running longer than the timeout
func (w *ExecutionPoller) isExecutionStuck(execution *models.StageExecution) bool {
	if execution.StartedAt == nil {
		return false
	}

	runningDuration := w.nowFunc().Sub(*execution.StartedAt)
	return runningDuration > w.ExecutionTimeout
}

func (w *ExecutionPoller) ProcessStuckExecution(logger *log.Entry, execution *models.StageExecution) error {
	resources, err := execution.Resources()
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		logger.Warnf("Execution has no resources and has been running for %v, marking as failed",
			w.nowFunc().Sub(*execution.StartedAt))
		return w.finishStuckExecution(execution, models.ResultFailed, models.StageEventStateReasonStuck)
	}

	allFinished := true

	for _, resource := range resources {
		if resource.State != models.ExecutionResourcePending {
			continue
		}

		status, err := w.pollResourceStatusWithRetry(logger, resource)
		if err != nil {
			logger.Errorf("Failed to poll status for resource %s after retries: %v", resource.ExternalID, err)
			err = resource.Finish(models.ResultFailed)
			if err != nil {
				logger.Errorf("Failed to mark resource as failed: %v", err)
			}
			continue
		}

		if status.Finished() {
			result := models.ResultPassed
			if !status.Successful() {
				result = models.ResultFailed
			}

			err = resource.Finish(result)
			if err != nil {
				logger.Errorf("Failed to finish resource: %v", err)
			}
			logger.Infof("Updated resource %s status to %s via polling", resource.ExternalID, result)
		} else {
			allFinished = false
			logger.Infof("Resource %s is still running according to third-party API", resource.ExternalID)
		}
	}

	if allFinished {
		logger.Infof("All resources finished via polling, processing execution normally")
		return w.ProcessExecution(logger, execution)
	}

	//
	// If some resources are still running but we've exceeded timeout,
	// we need to decide whether to fail the execution or wait longer
	//
	runningDuration := w.nowFunc().Sub(*execution.StartedAt)
	if runningDuration > DefaultExecutionFinalTimeout {
		logger.Warnf("Execution has been running for %v (2x timeout), force failing", runningDuration)
		return w.finishStuckExecution(execution, models.ResultFailed, models.StageEventStateReasonTimeout)
	}

	// Otherwise, let it continue for now
	logger.Infof("Some resources still running, will check again in next poll cycle")
	return nil
}

func (w *ExecutionPoller) pollResourceStatusWithRetry(logger *log.Entry, resource models.ExecutionResource) (integrations.StatefulResource, error) {
	integration, err := resource.FindIntegration()
	if err != nil {
		return nil, err
	}

	integrationImpl, err := w.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return nil, err
	}

	parentResource, err := resource.FindParentResource()
	if err != nil {
		return nil, err
	}

	//
	// Retry logic with exponential backoff
	//
	var lastErr error
	for attempt := 0; attempt < MaxRetryAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(BaseRetryDelay) * math.Pow(2, float64(attempt-1)))
			logger.Infof("Retry attempt %d/%d for resource %s, waiting %v", attempt+1, MaxRetryAttempts, resource.ExternalID, delay)
			time.Sleep(delay)
		}

		statefulResource, err := integrationImpl.Status(resource.Type, resource.ExternalID, parentResource)
		if err == nil {
			logger.Infof("Successfully polled status for resource %s on attempt %d", resource.ExternalID, attempt+1)
			return statefulResource, nil
		}

		lastErr = err
		logger.Warnf("Attempt %d failed for resource %s: %v", attempt+1, resource.ExternalID, err)
	}

	return nil, lastErr
}

func (w *ExecutionPoller) finishStuckExecution(execution *models.StageExecution, result string, reason string) error {
	stage, err := models.FindStageByID(execution.CanvasID.String(), execution.StageID.String())
	if err != nil {
		return err
	}

	var event *models.Event
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if event, err = execution.FinishInTransaction(tx, stage, result); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	log.Infof("Finished stuck execution %s with result: %s (reason: %s)", execution.ID, result, reason)

	// Publish messages
	err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), execution).Publish()
	if err != nil {
		log.Errorf("Error publishing execution finished message: %v", err)
	}

	err = messages.NewEventCreatedMessage(stage.CanvasID.String(), event).Publish()
	if err != nil {
		log.Errorf("Error publishing event created message: %v", err)
	}

	return nil
}

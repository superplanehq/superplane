package workers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionPoller struct {
	Encryptor encryptor.Encryptor
}

func NewExecutionPoller(encryptor encryptor.Encryptor) *ExecutionPoller {
	return &ExecutionPoller{Encryptor: encryptor}
}

func (w *ExecutionPoller) Start() error {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing started executions: %v", err)
		}

		time.Sleep(15 * time.Second)
	}
}

func (w *ExecutionPoller) Tick() error {
	executions, err := models.ListStageExecutionsInState(models.StageExecutionStarted)
	if err != nil {
		return err
	}

	for _, execution := range executions {
		e := execution
		logger := logging.ForExecution(&e)
		logger.Infof("Processing")
		err := w.ProcessExecution(logger, &e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *ExecutionPoller) ProcessExecution(logger *log.Entry, execution *models.StageExecution) error {
	stage, err := models.FindStageByID(execution.StageID.String())
	if err != nil {
		return err
	}

	result, err := w.resolveExecutionResult(logger, stage, execution)
	if err != nil {
		return err
	}

	if result == models.StageExecutionStarted {
		logger.Info("No change in state")
		return nil
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		tags, err := w.processExecutionTags(tx, logger, execution, result)
		if err != nil {
			logger.Errorf("Error processing execution tags: %v", err)
			return err
		}

		if err := execution.FinishInTransaction(tx, result); err != nil {
			logger.Errorf("Error updating execution state: %v", err)
			return err
		}

		err = models.UpdateStageEventsInTransaction(
			tx, []string{execution.StageEventID.String()}, models.StageEventStateProcessed, "",
		)

		if err != nil {
			logger.Errorf("Error updating stage event state: %v", err)
			return err
		}

		//
		// Lastly, since the stage for this execution might be connected to other stages,
		// we create a new event for the completion of this stage.
		//
		if err := w.createStageCompletionEvent(tx, execution, tags); err != nil {
			logger.Errorf("Error creating stage completion event: %v", err)
			return err
		}

		logger.Infof("Execution state updated: %s", result)
		return nil
	})

	if err == nil {
		stage, err := models.FindStageByID(execution.StageID.String())
		if err != nil {
			logger.Errorf("Error finding stage for execution: %v", err)
			return err
		}

		err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), execution).Publish()
		if err != nil {
			logger.Errorf("Error publishing execution finished message: %v", err)
		}
	}

	return err
}

func (w *ExecutionPoller) resolveExecutionResult(logger *log.Entry, stage *models.Stage, execution *models.StageExecution) (string, error) {
	executor := stage.ExecutorSpec.Data()
	switch executor.Type {
	case models.ExecutorSpecTypeSemaphore:
		t := executor.Semaphore
		decoded, err := base64.StdEncoding.DecodeString(t.APIToken)
		if err != nil {
			return "", err
		}

		token, err := w.Encryptor.Decrypt(context.Background(), decoded, []byte(t.OrganizationURL))
		if err != nil {
			return "", err
		}

		return w.resolveResultFromSemaphoreWorkflow(
			logger,
			execution,
			semaphore.NewSemaphoreAPI(
				executor.Semaphore.OrganizationURL,
				string(token),
			),
		)
	default:
		return "", fmt.Errorf("executor %s not supported", executor.Type)
	}
}

func (w *ExecutionPoller) resolveResultFromSemaphoreWorkflow(logger *log.Entry, execution *models.StageExecution, api *semaphore.Semaphore) (string, error) {
	pipeline, err := w.findPipeline(api, execution.ReferenceID)
	if err != nil {
		log.Errorf("Error finding pipeline: %v", err)
		return "", err
	}

	if pipeline.State != semaphore.PipelineStateDone {
		logger.Infof("Pipeline state is %s - skipping", pipeline.State)
		return models.StageExecutionStarted, nil
	}

	logger.Infof("Pipeline %s - state=%s, result=%s", pipeline.ID, pipeline.State, pipeline.Result)

	switch pipeline.Result {
	case semaphore.PipelineResultPassed:
		return models.StageExecutionResultPassed, nil
	default:
		return models.StageExecutionResultFailed, nil
	}
}

func (w *ExecutionPoller) findPipeline(api *semaphore.Semaphore, workflowID string) (*semaphore.Pipeline, error) {
	workflow, err := api.DescribeWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	pipeline, err := api.DescribePipeline(workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	return pipeline, nil
}

func (w *ExecutionPoller) createStageCompletionEvent(tx *gorm.DB, execution *models.StageExecution, tags map[string]string) error {
	stage, err := models.FindStageByIDInTransaction(tx, execution.StageID.String())
	if err != nil {
		return err
	}

	e, err := events.NewStageExecutionCompletion(execution, tags)
	if err != nil {
		return fmt.Errorf("error creating stage completion event: %v", err)
	}

	raw, err := json.Marshal(&e)
	if err != nil {
		return fmt.Errorf("error marshaling event: %v", err)
	}

	_, err = models.CreateEventInTransaction(tx, execution.StageID, stage.Name, models.SourceTypeStage, raw, []byte(`{}`))
	if err != nil {
		return fmt.Errorf("error creating event: %v", err)
	}

	return nil
}

func (w *ExecutionPoller) processExecutionTags(tx *gorm.DB, logger *log.Entry, execution *models.StageExecution, result string) (map[string]string, error) {
	tags := map[string]string{}

	//
	// Include extra tags from execution, if any.
	//
	if execution.Tags != nil {
		err := json.Unmarshal(execution.Tags, &tags)
		if err != nil {
			return nil, fmt.Errorf("error adding tags from execution: %v", err)
		}
	}

	return tags, nil
}

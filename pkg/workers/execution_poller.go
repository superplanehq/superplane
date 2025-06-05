package workers

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionPoller struct {
	Encryptor   crypto.Encryptor
	SpecBuilder executors.SpecBuilder
}

func NewExecutionPoller(encryptor crypto.Encryptor) *ExecutionPoller {
	return &ExecutionPoller{
		Encryptor:   encryptor,
		SpecBuilder: executors.SpecBuilder{},
	}
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

	inputMap, err := execution.GetInputs()
	if err != nil {
		return err
	}

	secrets, err := stage.FindSecrets(w.Encryptor)
	if err != nil {
		return err
	}

	spec, err := w.SpecBuilder.Build(stage.ExecutorSpec.Data(), inputMap, secrets)
	if err != nil {
		return err
	}

	executor, err := executors.NewExecutor(spec.Type, *execution, nil)
	if err != nil {
		return err
	}

	status, err := executor.Check(*spec, execution.ReferenceID)
	if err != nil {
		return err
	}

	if !status.Finished() {
		logger.Info("Not finished yet")
		return nil
	}

	result := models.StageExecutionResultFailed
	if status.Successful() {
		result = models.StageExecutionResultPassed
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		outputs := execution.Outputs.Data()

		//
		// Check if all required outputs were pushed.
		// If any output wasn't pushed, mark the execution as failed.
		//
		missingOutputs := stage.MissingRequiredOutputs(outputs)
		if len(missingOutputs) > 0 {
			logger.Infof("Missing outputs %v - marking the execution as failed", missingOutputs)
			result = models.StageExecutionResultFailed
		}

		if err := execution.FinishInTransaction(tx, stage, result); err != nil {
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

	return nil
}

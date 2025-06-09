package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

type PendingExecutionsWorker struct {
	JwtSigner   *jwt.Signer
	Encryptor   crypto.Encryptor
	SpecBuilder executors.SpecBuilder
}

func (w *PendingExecutionsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing pending executions: %v", err)
		}

		time.Sleep(time.Second)
	}
}

func (w *PendingExecutionsWorker) Tick() error {
	executions, err := models.ListStageExecutionsInState(models.StageExecutionPending)
	if err != nil {
		return fmt.Errorf("error listing pending stage executions: %v", err)
	}

	for _, execution := range executions {
		stage, err := models.FindStageByID(execution.StageID.String())
		if err != nil {
			return fmt.Errorf("error finding stage %s: %v", execution.StageID, err)
		}

		logger := logging.ForStage(stage)
		if err := w.ProcessExecution(logger, stage, execution); err != nil {
			return fmt.Errorf("error processing execution %s: %v", execution.ID, err)
		}
	}

	return nil
}

// TODO
// There is an issue here where, if we are having issues updating the state of the execution in the database,
// we might end up creating more executions than we should.
func (w *PendingExecutionsWorker) ProcessExecution(logger *log.Entry, stage *models.Stage, execution models.StageExecution) error {
	inputMap, err := execution.GetInputs()
	if err != nil {
		return fmt.Errorf("error finding inputs for execution: %v", err)
	}

	secrets, err := stage.FindSecrets(w.Encryptor)
	if err != nil {
		return fmt.Errorf("error finding secrets for execution: %v", err)
	}

	spec, err := w.SpecBuilder.Build(stage.ExecutorSpec.Data(), inputMap, secrets)
	if err != nil {
		return err
	}

	executor, err := executors.NewExecutor(spec.Type, execution, w.JwtSigner)
	if err != nil {
		return fmt.Errorf("error creating executor: %v", err)
	}

	err = execution.Start()
	if err != nil {
		return fmt.Errorf("error moving execution to started state: %v", err)
	}

	//
	// If we get an error calling the executor, we fail the execution.
	//
	response, err := executor.Execute(*spec)
	if err != nil {
		logger.Errorf("Error calling executor: %v - failing execution", err)
		err := execution.Finish(stage, models.StageExecutionResultFailed)
		if err != nil {
			return fmt.Errorf("error moving execution to failed state: %v", err)
		}

		return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()

	}

	if response.Finished() {
		return w.handleSyncResource(logger, response, execution, stage)
	}

	return w.handleAsyncResource(logger, response, stage, execution)
}

func (w *PendingExecutionsWorker) handleSyncResource(logger *log.Entry, response executors.Response, execution models.StageExecution, stage *models.Stage) error {
	outputs := response.Outputs()
	if len(outputs) > 0 {
		if err := execution.UpdateOutputs(outputs); err != nil {
			return fmt.Errorf("error setting outputs: %v", err)
		}
	}

	result := models.StageExecutionResultFailed
	if response.Successful() {
		result = models.StageExecutionResultPassed
	}

	//
	// Check if all required outputs were received.
	//
	missingOutputs := stage.MissingRequiredOutputs(outputs)
	if len(missingOutputs) > 0 {
		logger.Infof("Execution has missing outputs %v - marking the execution as failed", missingOutputs)
		result = models.StageExecutionResultFailed
	}

	err := execution.Finish(stage, result)
	if err != nil {
		return err
	}

	logger.Infof("Finished execution: %s", result)

	return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
}

func (w *PendingExecutionsWorker) handleAsyncResource(logger *log.Entry, response executors.Response, stage *models.Stage, execution models.StageExecution) error {
	err := execution.StartWithReferenceID(response.Id())
	if err != nil {
		return fmt.Errorf("error moving execution to started state: %v", err)
	}

	err = messages.NewExecutionStartedMessage(stage.CanvasID.String(), &execution).Publish()
	if err != nil {
		return fmt.Errorf("error publishing execution started message: %v", err)
	}

	logger.Infof("Started execution %s", response.Id())

	return nil
}

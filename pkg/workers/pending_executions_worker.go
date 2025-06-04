package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

type PendingExecutionsWorker struct {
	JwtSigner *jwt.Signer
	Encryptor crypto.Encryptor
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

	executor, err := executors.NewExecutor(stage.ExecutorSpec.Data().Type, execution, w.JwtSigner)
	if err != nil {
		return fmt.Errorf("error creating executor: %v", err)
	}

	spec, err := executor.BuildSpec(stage.ExecutorSpec.Data(), inputMap, secrets)
	if err != nil {
		return fmt.Errorf("error building spec: %v", err)
	}

	response, err := executor.Execute(*spec)
	if err != nil {
		return fmt.Errorf("executor Execute() error: %v", err)
	}

	if response.Finished() {
		return w.handleAsyncResource(logger, response, stage, execution)
	}

	err = w.handleSyncResource(response, execution)
	if err != nil {
		return err
	}

	return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
}

func (w *PendingExecutionsWorker) handleSyncResource(response executors.Response, execution models.StageExecution) error {
	if response.Successful() {
		return execution.FinishInTransaction(database.Conn(), models.StageExecutionResultPassed)
	}

	return execution.FinishInTransaction(database.Conn(), models.StageExecutionResultFailed)
}

func (w *PendingExecutionsWorker) handleAsyncResource(logger *log.Entry, response executors.Response, stage *models.Stage, execution models.StageExecution) error {
	err := execution.Start(response.Id())
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

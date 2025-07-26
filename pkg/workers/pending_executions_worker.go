package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
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
	executions, err := models.ListExecutionsInState(models.ExecutionPending)
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

	secrets, err := w.FindSecrets(stage, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error finding secrets for execution: %v", err)
	}

	stageExecutor, err := stage.GetExecutor()
	if err != nil {
		return fmt.Errorf("error getting executor for stage: %v", err)
	}

	spec, err := w.SpecBuilder.Build(stageExecutor.Spec, inputMap, secrets)
	if err != nil {
		return err
	}

	executor, err := w.initExecutor(stageExecutor)
	if err != nil {
		return fmt.Errorf("error initializing executor: %v", err)
	}

	token, err := w.JwtSigner.Generate(execution.ID.String(), 24*time.Hour)
	if err != nil {
		return fmt.Errorf("error generating token: %v", err)
	}

	//
	// If we get an error calling the executor, we fail the execution.
	//
	response, err := executor.Execute(spec, executors.ExecutionParameters{
		ExecutionID: execution.ID.String(),
		StageID:     stage.ID.String(),
		Token:       token,
	})

	if err != nil {
		logger.Errorf("Error calling executor: %v - failing execution", err)
		err := execution.Finish(stage, models.ResultFailed)
		if err != nil {
			return fmt.Errorf("error moving execution to failed state: %v", err)
		}

		return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()

	}

	if response.Finished() {
		return w.handleSyncResource(logger, response, execution, stage)
	}

	return w.handleAsyncResource(logger, response, stageExecutor, stage, execution)
}

func (w *PendingExecutionsWorker) initExecutor(stageExecutor *models.StageExecutor) (executors.Executor, error) {
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

func (w *PendingExecutionsWorker) FindSecrets(stage *models.Stage, encryptor crypto.Encryptor) (map[string]string, error) {
	secretMap := map[string]string{}
	for _, def := range stage.Secrets {
		secretDef := def.ValueFrom.Secret
		provider, err := secretProvider(encryptor, secretDef, stage)
		if err != nil {
			return nil, fmt.Errorf("error initializing secret provider for %s: %v", secretDef.Name, err)
		}

		values, err := provider.Load(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error loading values for secret %s: %v", secretDef.Name, err)
		}

		value, ok := values[secretDef.Key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in secret %s", secretDef.Key, secretDef.Name)
		}

		secretMap[secretDef.Name] = value
	}

	return secretMap, nil
}

func (w *PendingExecutionsWorker) handleSyncResource(logger *log.Entry, response executors.Response, execution models.StageExecution, stage *models.Stage) error {
	outputs := response.Outputs()
	if len(outputs) > 0 {
		if err := execution.UpdateOutputs(outputs); err != nil {
			return fmt.Errorf("error setting outputs: %v", err)
		}
	}

	result := models.ResultFailed
	if response.Successful() {
		result = models.ResultPassed
	}

	//
	// Check if all required outputs were received.
	//
	missingOutputs := stage.MissingRequiredOutputs(outputs)
	if len(missingOutputs) > 0 {
		logger.Infof("Execution has missing outputs %v - marking the execution as failed", missingOutputs)
		result = models.ResultFailed
	}

	err := execution.Finish(stage, result)
	if err != nil {
		return err
	}

	logger.Infof("Finished execution: %s", result)

	return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
}

func (w *PendingExecutionsWorker) handleAsyncResource(logger *log.Entry, response executors.Response, executor *models.StageExecutor, stage *models.Stage, execution models.StageExecution) error {
	_, err := execution.AddResource(response.Id(), *executor.ResourceID)
	if err != nil {
		return fmt.Errorf("error adding resource to execution: %v", err)
	}

	err = execution.Start()
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

func secretProvider(encryptor crypto.Encryptor, secretDef *models.ValueDefinitionFromSecret, stage *models.Stage) (secrets.Provider, error) {
	if secretDef.DomainType == models.DomainTypeCanvas {
		return secrets.NewProvider(encryptor, secretDef.Name, secretDef.DomainType, stage.CanvasID)
	}

	canvas, err := models.FindCanvasByID(stage.CanvasID.String())
	if err != nil {
		return nil, fmt.Errorf("error finding canvas %s: %v", stage.CanvasID, err)
	}

	return secrets.NewProvider(encryptor, secretDef.Name, secretDef.DomainType, canvas.OrganizationID)
}

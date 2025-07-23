package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
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
// we might end up creating more execution resources than we should.
func (w *PendingExecutionsWorker) ProcessExecution(logger *log.Entry, stage *models.Stage, execution models.StageExecution) error {
	inputMap, err := execution.GetInputs()
	if err != nil {
		return fmt.Errorf("error finding inputs for execution: %v", err)
	}

	stageExecutor, err := stage.GetExecutor()
	if err != nil {
		return fmt.Errorf("error getting executor for stage: %v", err)
	}

	integration, executor, err := w.getExecutor(stageExecutor)
	if err != nil {
		return fmt.Errorf("error getting executor: %v", err)
	}

	secrets, err := w.FindSecrets(stage, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error finding secrets for execution: %v", err)
	}

	executorSpec := stageExecutor.Spec.Data()
	spec, err := w.SpecBuilder.Build(executorSpec, inputMap, secrets)
	if err != nil {
		return err
	}

	parameters, err := w.buildExecutionParameters(&execution, integration)
	if err != nil {
		return fmt.Errorf("error building execution parameters: %v", err)
	}

	//
	// If we get an error calling the executor, we fail the execution.
	//
	response, err := executor.Execute(*spec, *parameters)
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

func (w *PendingExecutionsWorker) getExecutor(stageExecutor *models.StageExecutor) (integrations.Integration, executors.Executor, error) {
	if stageExecutor.ResourceID == nil {
		executor, err := executors.NewExecutorWithoutIntegration(stageExecutor)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating executor: %v", err)
		}

		return nil, executor, nil
	}

	resource, err := stageExecutor.GetResource()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting resource for stage executor: %v", err)
	}

	integration, err := stageExecutor.FindIntegration()
	if err != nil {
		return nil, nil, fmt.Errorf("error finding integration: %v", err)
	}

	integrationImpl, err := integrations.NewIntegration(context.Background(), integration, w.Encryptor)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating integration: %v", err)
	}

	executor, err := executors.NewExecutorWithIntegration(integrationImpl, resource, stageExecutor)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating executor: %v", err)
	}

	return integrationImpl, executor, nil
}

func (w *PendingExecutionsWorker) buildExecutionParameters(execution *models.StageExecution, integration integrations.Integration) (*executors.ExecutionParameters, error) {
	parameters := executors.ExecutionParameters{
		StageID:     execution.StageID.String(),
		ExecutionID: execution.ID.String(),
	}

	if integration != nil && !integration.HasSupportFor(integrations.FeatureOpenIdConnectToken) {
		token, err := w.JwtSigner.Generate(execution.ID.String(), time.Hour)
		if err != nil {
			return nil, fmt.Errorf("error generating token: %v", err)
		}

		parameters.Token = token
	}

	return &parameters, nil
}

func (w *PendingExecutionsWorker) FindSecrets(stage *models.Stage, encryptor crypto.Encryptor) (map[string]string, error) {
	secretMap := map[string]string{}
	for _, secretDef := range stage.Secrets {
		secretName := secretDef.ValueFrom.Secret.Name

		// TODO: it should be possible for organization secrets to be used here too.
		provider, err := secrets.NewProvider(encryptor, secretName, models.DomainTypeCanvas, stage.CanvasID)
		if err != nil {
			return nil, fmt.Errorf("error initializing secret provider for %s: %v", secretName, err)
		}

		values, err := provider.Load(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error loading values for secret %s: %v", secretName, err)
		}

		key := secretDef.ValueFrom.Secret.Key
		value, ok := values[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in secret %s", key, secretName)
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

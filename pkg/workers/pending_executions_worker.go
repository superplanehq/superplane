package workers

import (
	"context"
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
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/secrets"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

const (
	ExecutionTokenDuration  = 24 * time.Hour
	MaxConcurrentExecutions = 25
)

type PendingExecutionsWorker struct {
	JwtSigner   *jwt.Signer
	Encryptor   crypto.Encryptor
	SpecBuilder executors.SpecBuilder
	Registry    *registry.Registry
	semaphore   *semaphore.Weighted
}

func NewPendingExecutionsWorker(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, specBuilder executors.SpecBuilder, registry *registry.Registry) *PendingExecutionsWorker {
	return &PendingExecutionsWorker{
		JwtSigner:   jwtSigner,
		Encryptor:   encryptor,
		SpecBuilder: specBuilder,
		Registry:    registry,
		semaphore:   semaphore.NewWeighted(MaxConcurrentExecutions),
	}
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
	executions, err := models.ListExecutionsInState(models.ExecutionPending, MaxConcurrentExecutions)
	if err != nil {
		return fmt.Errorf("error listing pending stage executions: %v", err)
	}

	if len(executions) == 0 {
		return nil
	}

	for _, execution := range executions {
		if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
			log.Errorf("Error acquiring semaphore: %v", err)
			continue
		}

		go func(exec models.StageExecution) {
			defer w.semaphore.Release(1)

			if err := w.LockAndProcessExecution(exec); err != nil {
				log.Errorf("Error processing execution %s: %v", exec.ID, err)
			}
		}(execution)
	}

	return nil
}

func (w *PendingExecutionsWorker) LockAndProcessExecution(execution models.StageExecution) error {
	logger := logging.ForExecution(&execution)

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		e, err := models.LockExecution(tx, execution.ID)
		if err != nil {
			logger.Info("Execution already being processed - skipping")
			return nil
		}

		stage, err := models.FindStageByIDInTransaction(tx, e.CanvasID.String(), e.StageID.String())
		if err != nil {
			return fmt.Errorf("error finding stage %s: %v", e.StageID, err)
		}

		return w.ProcessExecution(tx, logger, stage, *e)
	})
}

// TODO
// There is an issue here where, if we are having issues updating the state of the execution in the database,
// we might end up creating more execution resources than we should.
func (w *PendingExecutionsWorker) ProcessExecution(tx *gorm.DB, logger *log.Entry, stage *models.Stage, execution models.StageExecution) error {
	logger.Info("Processing")

	inputMap, err := execution.GetInputsInTransaction(tx)
	if err != nil {
		return fmt.Errorf("error finding inputs for execution: %v", err)
	}

	secrets, err := w.FindSecrets(tx, logger, stage, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error finding secrets for execution: %v", err)
	}

	logger.Info("Building executor spec")
	spec, err := w.SpecBuilder.Build(stage.ExecutorSpec, inputMap, secrets)
	if err != nil {
		return err
	}

	//
	// If the stage is in dry run mode, we use the no-op executor.
	//
	if stage.DryRun {
		logger.Info("Stage is in dry run mode, using no-op executor")
		executor, err := w.Registry.NewExecutor(models.ExecutorTypeNoOp)
		if err != nil {
			return err
		}

		return w.handleExecutor(tx, logger, executor, spec, execution, stage)
	}

	//
	// If the stage is not connected to integration,
	// we use an executor unrelated to integrations.
	//
	if stage.ResourceID == nil {
		executor, err := w.Registry.NewExecutor(stage.ExecutorType)
		if err != nil {
			return err
		}

		return w.handleExecutor(tx, logger, executor, spec, execution, stage)
	}

	//
	// If the stage is connected to integration,
	// we use an executor related to integrations.
	//
	return w.handleIntegrationExecutor(tx, logger, spec, stage, execution)
}

func (w *PendingExecutionsWorker) FindSecrets(tx *gorm.DB, logger *log.Entry, stage *models.Stage, encryptor crypto.Encryptor) (map[string]string, error) {
	logger.Info("Loading secrets")
	secretMap := map[string]string{}
	for _, def := range stage.Secrets {
		secretDef := def.ValueFrom.Secret
		provider, err := secretProvider(tx, encryptor, secretDef, stage)
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

func (w *PendingExecutionsWorker) handleExecutor(tx *gorm.DB, logger *log.Entry, executor executors.Executor, spec []byte, execution models.StageExecution, stage *models.Stage) error {
	logger.Info("Calling executor")
	response, err := executor.Execute(spec, executors.ExecutionParameters{
		ExecutionID: execution.ID.String(),
		StageID:     stage.ID.String(),
		OutputNames: stage.OutputNames(),
	})

	if err != nil {
		logger.Errorf("Error calling executor: %v - failing execution", err)
		newEvent, err := execution.FinishInTransaction(tx, stage, models.ResultFailed, models.ResultReasonError, fmt.Sprintf("Error calling executor: %v", err))
		if err != nil {
			return fmt.Errorf("error moving execution to failed state: %v", err)
		}

		err = messages.NewEventCreatedMessage(stage.CanvasID.String(), newEvent).Publish()
		if err != nil {
			return fmt.Errorf("error publishing event created message: %v", err)
		}

		return messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
	}

	outputs := response.Outputs()
	if len(outputs) > 0 {
		if err := execution.UpdateOutputsInTransaction(tx, outputs); err != nil {
			return fmt.Errorf("error setting outputs: %v", err)
		}
	}

	result := models.ResultFailed
	resultReason := ""
	resultMessage := ""

	if response.Successful() {
		result = models.ResultPassed
	}

	//
	// Check if all required outputs were received.
	//
	missingOutputs := stage.MissingRequiredOutputs(outputs)
	if len(missingOutputs) > 0 {
		result = models.ResultFailed
		resultReason = models.ResultReasonMissingOutputs
		resultMessage = fmt.Sprintf("missing outputs: %v", missingOutputs)
	}

	logger.Infof("Finishing execution. Result: %s, Reason: %s, Message: %s", result, resultReason, resultMessage)
	newEvent, err := execution.FinishInTransaction(tx, stage, result, resultReason, resultMessage)
	if err != nil {
		return err
	}

	err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
	if err != nil {
		return fmt.Errorf("error publishing execution finished message: %v", err)
	}

	err = messages.NewEventCreatedMessage(stage.CanvasID.String(), newEvent).Publish()
	if err != nil {
		return fmt.Errorf("error publishing event created message: %v", err)
	}

	return nil
}

func (w *PendingExecutionsWorker) handleIntegrationExecutor(tx *gorm.DB, logger *log.Entry, spec []byte, stage *models.Stage, execution models.StageExecution) error {
	integration, err := stage.FindIntegrationInTransaction(tx)
	if err != nil {
		return err
	}

	resource, err := stage.GetResourceInTransaction(tx)
	if err != nil {
		return err
	}

	logger.Info("Building execution parameters")
	parameters, err := w.buildExecutionParameters(&execution, integration)
	if err != nil {
		return err
	}

	integrationExecutor, err := w.Registry.NewIntegrationExecutorWithTx(tx, integration, resource)
	if err != nil {
		return err
	}

	logger.Info("Calling integration executor")
	statefulResource, err := integrationExecutor.Execute(spec, *parameters)
	if err != nil {
		logger.Errorf("Error calling executor: %v - failing execution", err)
		newEvent, err := execution.FinishInTransaction(tx, stage, models.ResultFailed, models.ResultReasonError, fmt.Sprintf("Error calling executor: %v", err))
		if err != nil {
			return fmt.Errorf("error moving execution to failed state: %v", err)
		}

		err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), &execution).Publish()
		if err != nil {
			return fmt.Errorf("error publishing execution finished message: %v", err)
		}

		err = messages.NewEventCreatedMessage(stage.CanvasID.String(), newEvent).Publish()
		if err != nil {
			return fmt.Errorf("error publishing event created message: %v", err)
		}

		return nil
	}

	_, err = execution.AddResourceInTransaction(
		tx,
		statefulResource.Id(),
		statefulResource.Type(),
		statefulResource.URL(),
		*stage.ResourceID,
	)

	if err != nil {
		return fmt.Errorf("error adding resource to execution: %v", err)
	}

	err = execution.StartInTransaction(tx)
	if err != nil {
		return fmt.Errorf("error moving execution to started state: %v", err)
	}

	err = messages.NewExecutionStartedMessage(stage.CanvasID.String(), &execution).Publish()
	if err != nil {
		return fmt.Errorf("error publishing execution started message: %v", err)
	}

	logger.Infof("Created %s %s: %s", integration.Type, statefulResource.Type(), statefulResource.Id())

	return nil
}

func (w *PendingExecutionsWorker) buildExecutionParameters(execution *models.StageExecution, integration *models.Integration) (*executors.ExecutionParameters, error) {
	parameters := executors.ExecutionParameters{
		ExecutionID: execution.ID.String(),
		StageID:     execution.StageID.String(),
	}

	//
	// If the integration has an OIDC verifier,
	// it can push outputs using the OIDC token generated by itself.
	//
	if w.Registry.HasOIDCVerifier(integration.Type) {
		return &parameters, nil
	}

	//
	// Otherwise, we need to inject a token to allow the execution to push outputs.
	//
	token, err := w.JwtSigner.Generate(execution.ID.String(), ExecutionTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("error generating token: %v", err)
	}

	parameters.Token = token
	return &parameters, nil
}

func secretProvider(tx *gorm.DB, encryptor crypto.Encryptor, secretDef *models.ValueDefinitionFromSecret, stage *models.Stage) (secrets.Provider, error) {
	if secretDef.DomainType == models.DomainTypeCanvas {
		return secrets.NewProvider(tx, encryptor, secretDef.Name, secretDef.DomainType, stage.CanvasID)
	}

	canvas, err := models.FindUnscopedCanvasByIDInTransaction(tx, stage.CanvasID.String())
	if err != nil {
		return nil, fmt.Errorf("error finding canvas %s: %v", stage.CanvasID, err)
	}

	return secrets.NewProvider(tx, encryptor, secretDef.Name, secretDef.DomainType, canvas.OrganizationID)
}

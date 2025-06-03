package workers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

type PendingExecutionsWorker struct {
	JwtSigner *jwt.Signer
	Encryptor encryptor.Encryptor
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

	specBuilder := inputs.NewExecutorSpecBuilder(stage.ExecutorSpec.Data(), inputMap)
	spec, err := specBuilder.Build()
	if err != nil {
		return fmt.Errorf("error resolving executor spec: %v", err)
	}

	executionID, err := w.StartExecution(logger, stage, execution, *spec)
	if err != nil {
		return fmt.Errorf("error starting execution: %v", err)
	}

	err = execution.Start(executionID)
	if err != nil {
		return fmt.Errorf("error moving execution to started state: %v", err)
	}

	err = messages.NewExecutionStartedMessage(stage.CanvasID.String(), &execution).Publish()
	if err != nil {
		return fmt.Errorf("error publishing execution started message: %v", err)
	}

	logger.Infof("Started execution %s", executionID)

	return nil
}

// TODO: implement some retry and give up mechanism
func (w *PendingExecutionsWorker) StartExecution(logger *log.Entry, stage *models.Stage, execution models.StageExecution, spec models.ExecutorSpec) (string, error) {
	switch spec.Type {
	case models.ExecutorSpecTypeSemaphore:
		//
		// For now, only task runs are supported,
		// until the workflow API is updated to support parameters.
		//
		if spec.Semaphore.TaskID == "" {
			return "", fmt.Errorf("only task runs are supported")
		}

		return w.TriggerSemaphoreTask(logger, stage, execution, spec.Semaphore)
	default:
		return "", fmt.Errorf("unknown executor spec type")
	}
}

func (w *PendingExecutionsWorker) TriggerSemaphoreTask(logger *log.Entry, stage *models.Stage, execution models.StageExecution, spec *models.SemaphoreExecutorSpec) (string, error) {
	api, err := w.newSemaphoreAPI(spec)
	if err != nil {
		return "", err
	}

	parameters, err := w.buildParameters(execution, spec.Parameters)
	if err != nil {
		return "", fmt.Errorf("error building parameters: %v", err)
	}

	workflowID, err := api.TriggerTask(spec.ProjectID, spec.TaskID, semaphore.TaskTriggerSpec{
		Branch:       spec.Branch,
		PipelineFile: spec.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return "", err
	}

	logger.Infof("Semaphore task triggered - workflow=%s", workflowID)
	return workflowID, nil
}

func (w *PendingExecutionsWorker) newSemaphoreAPI(spec *models.SemaphoreExecutorSpec) (*semaphore.Semaphore, error) {
	token, err := base64.StdEncoding.DecodeString(spec.APIToken)
	if err != nil {
		return nil, err
	}

	t, err := w.Encryptor.Decrypt(context.Background(), token, []byte(spec.OrganizationURL))
	if err != nil {
		return nil, err
	}

	return semaphore.NewSemaphoreAPI(spec.OrganizationURL, string(t)), nil
}

// TODO
// How should we pass these SEMAPHORE_* parameters to the job?
// SEMAPHORE_STAGE_ID and SEMAPHORE_STAGE_EXECUTION_ID are not sensitive values,
// but currently, if the task does not define a parameter, it is ignored.
//
// Additionally, SEMAPHORE_STAGE_EXECUTION_TOKEN is sensitive,
// so if we pass it here, it will be visible in UI / API responses.
func (w *PendingExecutionsWorker) buildParameters(execution models.StageExecution, parameters map[string]string) ([]semaphore.TaskTriggerParameter, error) {
	parameterValues := []semaphore.TaskTriggerParameter{
		{Name: "SEMAPHORE_STAGE_ID", Value: execution.StageID.String()},
		{Name: "SEMAPHORE_STAGE_EXECUTION_ID", Value: execution.ID.String()},
	}

	token, err := w.JwtSigner.Generate(execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating outputs token: %v", err)
	}

	parameterValues = append(parameterValues, semaphore.TaskTriggerParameter{
		Name:  "SEMAPHORE_STAGE_EXECUTION_TOKEN",
		Value: token,
	})

	for key, value := range parameters {
		parameterValues = append(parameterValues, semaphore.TaskTriggerParameter{
			Name:  key,
			Value: value,
		})
	}

	return parameterValues, nil
}

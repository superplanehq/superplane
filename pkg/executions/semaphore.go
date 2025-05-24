package executions

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	execution models.StageExecution
	template  *models.SemaphoreRunTemplate
	encryptor encryptor.Encryptor
	jwtSigner *jwt.Signer
}

type SemaphoreResource struct {
	api  *semaphore.Semaphore
	wfID string
}

// Since a Semaphore execution creates a Semaphore pipeline,
// and a Semaphore pipeline is not finished after the HTTP API call completes,
// we need to monitor the state of the created pipeline.
// That makes the Semaphore executor type async.
func (r *SemaphoreResource) Async() bool {
	return true
}

// The API call to run a pipeline gives me back a workflow ID,
// so we use that ID as the unique identifier here.
func (r *SemaphoreResource) AsyncId() string {
	return r.wfID
}

func (r *SemaphoreResource) Check() (Status, error) {
	workflow, err := r.api.DescribeWorkflow(r.wfID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", r.wfID)
	}

	pipeline, err := r.api.DescribePipeline(workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	return &SemaphoreStatus{pipeline: pipeline}, nil
}

type SemaphoreStatus struct {
	pipeline *semaphore.Pipeline
}

func (s *SemaphoreStatus) Finished() bool {
	return s.pipeline.State == semaphore.PipelineStateDone
}

func (s *SemaphoreStatus) Successful() bool {
	return s.pipeline.Result == semaphore.PipelineResultPassed
}

func NewSemaphoreExecutor(execution models.StageExecution, template *models.SemaphoreRunTemplate, encryptor encryptor.Encryptor, jwtSigner *jwt.Signer) (*SemaphoreExecutor, error) {
	return &SemaphoreExecutor{
		execution: execution,
		template:  template,
		encryptor: encryptor,
		jwtSigner: jwtSigner,
	}, nil
}

func (e *SemaphoreExecutor) Execute() (Resource, error) {
	//
	// For now, only task runs are supported,
	// until the workflow API is updated to support parameters.
	//
	if e.template.TaskID == "" {
		return nil, fmt.Errorf("only task runs are supported")
	}

	return e.triggerSemaphoreTask()
}

func (e *SemaphoreExecutor) AsyncCheck(id string) (Status, error) {
	api, err := e.newAPI()
	if err != nil {
		return nil, err
	}

	r := &SemaphoreResource{api: api, wfID: id}
	return r.Check()
}

func (e *SemaphoreExecutor) triggerSemaphoreTask() (Resource, error) {
	api, err := e.newAPI()
	if err != nil {
		return nil, err
	}

	parameters, err := e.buildParameters()
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	workflowID, err := api.TriggerTask(e.template.ProjectID, e.template.TaskID, semaphore.TaskTriggerSpec{
		Branch:       e.template.Branch,
		PipelineFile: e.template.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResource{api: api, wfID: workflowID}, nil
}

func (e *SemaphoreExecutor) newAPI() (*semaphore.Semaphore, error) {
	token, err := base64.StdEncoding.DecodeString(e.template.APIToken)
	if err != nil {
		return nil, err
	}

	t, err := e.encryptor.Decrypt(context.Background(), token, []byte(e.template.OrganizationURL))
	if err != nil {
		return nil, err
	}

	return semaphore.NewSemaphoreAPI(e.template.OrganizationURL, string(t)), nil
}

// TODO
// How should we pass these SEMAPHORE_* parameters to the job?
// SEMAPHORE_STAGE_ID and SEMAPHORE_STAGE_EXECUTION_ID are not sensitive values,
// but currently, if the task does not define a parameter, it is ignored.
//
// Additionally, SEMAPHORE_STAGE_EXECUTION_TOKEN is sensitive,
// so if we pass it here, it will be visible in UI / API responses.
func (e *SemaphoreExecutor) buildParameters() ([]semaphore.TaskTriggerParameter, error) {
	parameterValues := []semaphore.TaskTriggerParameter{
		{Name: "SEMAPHORE_STAGE_ID", Value: e.execution.StageID.String()},
		{Name: "SEMAPHORE_STAGE_EXECUTION_ID", Value: e.execution.ID.String()},
	}

	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	parameterValues = append(parameterValues, semaphore.TaskTriggerParameter{
		Name:  "SEMAPHORE_STAGE_EXECUTION_TOKEN",
		Value: token,
	})

	for key, value := range e.template.Parameters {
		parameterValues = append(parameterValues, semaphore.TaskTriggerParameter{
			Name:  key,
			Value: value,
		})
	}

	return parameterValues, nil
}

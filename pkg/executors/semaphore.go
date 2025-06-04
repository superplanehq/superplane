package executors

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	execution models.StageExecution
	jwtSigner *jwt.Signer
}

type SemaphoreResponse struct {
	api  *semaphore.Semaphore
	wfID string
}

// Since a Semaphore execution creates a Semaphore pipeline,
// and a Semaphore pipeline is not finished after the HTTP API call completes,
// we need to monitor the state of the created pipeline.
// That makes the Semaphore executor type async.
func (r *SemaphoreResponse) Async() bool {
	return true
}

// The API call to run a pipeline gives me back a workflow ID,
// so we use that ID as the unique identifier here.
func (r *SemaphoreResponse) AsyncId() string {
	return r.wfID
}

func (r *SemaphoreResponse) Check() (Status, error) {
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

func NewSemaphoreExecutor(execution models.StageExecution, jwtSigner *jwt.Signer) (*SemaphoreExecutor, error) {
	return &SemaphoreExecutor{
		execution: execution,
		jwtSigner: jwtSigner,
	}, nil
}

func (e *SemaphoreExecutor) Name() string {
	return models.ExecutorSpecTypeSemaphore
}

func (e *SemaphoreExecutor) BuildSpec(spec models.ExecutorSpec, inputs map[string]any, secrets map[string]string) (*models.ExecutorSpec, error) {
	orgURL, err := resolveExpression(spec.Semaphore.OrganizationURL, inputs, secrets)
	if err != nil {
		return nil, err
	}

	token, err := resolveExpression(spec.Semaphore.APIToken, inputs, secrets)
	if err != nil {
		return nil, err
	}

	projectID, err := resolveExpression(spec.Semaphore.ProjectID, inputs, secrets)
	if err != nil {
		return nil, err
	}

	branch, err := resolveExpression(spec.Semaphore.Branch, inputs, secrets)
	if err != nil {
		return nil, err
	}

	pipelineFile, err := resolveExpression(spec.Semaphore.PipelineFile, inputs, secrets)
	if err != nil {
		return nil, err
	}

	taskID, err := resolveExpression(spec.Semaphore.TaskID, inputs, secrets)
	if err != nil {
		return nil, err
	}

	parameters := make(map[string]string, len(spec.Semaphore.Parameters))
	for k, v := range spec.Semaphore.Parameters {
		value, err := resolveExpression(v, inputs, secrets)
		if err != nil {
			return nil, err
		}

		parameters[k] = value.(string)
	}

	return &models.ExecutorSpec{
		Type: models.ExecutorSpecTypeSemaphore,
		Semaphore: &models.SemaphoreExecutorSpec{
			OrganizationURL: orgURL.(string),
			APIToken:        token.(string),
			ProjectID:       projectID.(string),
			Branch:          branch.(string),
			PipelineFile:    pipelineFile.(string),
			TaskID:          taskID.(string),
			Parameters:      parameters,
		},
	}, nil
}

func (e *SemaphoreExecutor) Execute(spec models.ExecutorSpec) (Response, error) {
	//
	// For now, only task runs are supported,
	// until the workflow API is updated to support parameters.
	//
	if spec.Semaphore.TaskID == "" {
		return nil, fmt.Errorf("only task runs are supported")
	}

	return e.triggerSemaphoreTask(spec)
}

func (e *SemaphoreExecutor) triggerSemaphoreTask(spec models.ExecutorSpec) (Response, error) {
	api := semaphore.NewSemaphoreAPI(spec.Semaphore.OrganizationURL, string(spec.Semaphore.APIToken))
	parameters, err := e.buildParameters(spec.Semaphore.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	workflowID, err := api.TriggerTask(spec.Semaphore.ProjectID, spec.Semaphore.TaskID, semaphore.TaskTriggerSpec{
		Branch:       spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{api: api, wfID: workflowID}, nil
}

// TODO
// How should we pass these SEMAPHORE_* parameters to the job?
// SEMAPHORE_STAGE_ID and SEMAPHORE_STAGE_EXECUTION_ID are not sensitive values,
// but currently, if the task does not define a parameter, it is ignored.
//
// Additionally, SEMAPHORE_STAGE_EXECUTION_TOKEN is sensitive,
// so if we pass it here, it will be visible in UI / API responses.
func (e *SemaphoreExecutor) buildParameters(parameters map[string]string) ([]semaphore.TaskTriggerParameter, error) {
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

	for key, value := range parameters {
		parameterValues = append(parameterValues, semaphore.TaskTriggerParameter{
			Name:  key,
			Value: value,
		})
	}

	return parameterValues, nil
}

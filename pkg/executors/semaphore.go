package executors

import (
	"fmt"
	"maps"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	integration integrations.Integration
	execution   models.StageExecution
	jwtSigner   *jwt.Signer
}

type SemaphoreResponse struct {
	wfID     string
	pipeline *integrations.SemaphorePipeline
}

// Since a Semaphore execution creates a Semaphore pipeline,
// and a Semaphore pipeline is not finished after the HTTP API call completes,
// we need to monitor the state of the created pipeline.
// That makes the Semaphore executor type async.
func (r *SemaphoreResponse) Finished() bool {
	if r.pipeline == nil {
		return false
	}

	return r.pipeline.State == integrations.SemaphorePipelineStateDone
}

// The API call to run a pipeline gives me back a workflow ID,
// so we use that ID as the unique identifier here.
func (r *SemaphoreResponse) Id() string {
	return r.wfID
}

func (r *SemaphoreResponse) Successful() bool {
	if r.pipeline == nil {
		return false
	}

	return r.pipeline.Result == integrations.SemaphorePipelineResultPassed
}

// Outputs for Semaphore executions are sent via the /outputs API.
func (r *SemaphoreResponse) Outputs() map[string]any {
	return nil
}

func NewSemaphoreExecutor(integration integrations.Integration, execution models.StageExecution, jwtSigner *jwt.Signer) (*SemaphoreExecutor, error) {
	return &SemaphoreExecutor{
		integration: integration,
		execution:   execution,
		jwtSigner:   jwtSigner,
	}, nil
}

func (e *SemaphoreExecutor) Name() string {
	return models.ExecutorSpecTypeSemaphore
}

func (e *SemaphoreExecutor) Execute(spec models.ExecutorSpec) (Response, error) {
	if spec.Semaphore.TaskID == "" {
		return e.runWorkflow(spec)
	}

	return e.triggerTask(spec)
}

func (e *SemaphoreExecutor) Check(spec models.ExecutorSpec, id string) (Response, error) {
	resource, err := e.integration.GetResource(integrations.ResourceTypeWorkflow, id)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", id)
	}

	workflow := resource.(*integrations.SemaphoreWorkflow)
	resource, err = e.integration.GetResource(integrations.ResourceTypePipeline, workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	pipeline := resource.(*integrations.SemaphorePipeline)
	return &SemaphoreResponse{wfID: id, pipeline: pipeline}, nil
}

func (e *SemaphoreExecutor) runWorkflow(spec models.ExecutorSpec) (Response, error) {
	parameters, err := e.runWorkflowParameters(spec.Semaphore.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	resource, err := e.integration.CreateResource(integrations.ResourceTypeWorkflow, &integrations.CreateWorkflowParams{
		ProjectID:    spec.Semaphore.ProjectID,
		Reference:    "refs/heads/" + spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: resource.ID(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) triggerTask(spec models.ExecutorSpec) (Response, error) {
	parameters, err := e.taskTriggerParameters(spec.Semaphore.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	resource, err := e.integration.CreateResource(integrations.ResourceTypeTaskTrigger, &integrations.TaskTriggerParams{
		ProjectID: spec.Semaphore.ProjectID,
		TaskID:    spec.Semaphore.TaskID,
		TaskTrigger: &integrations.TaskTrigger{
			Spec: integrations.TaskTriggerSpec{
				Branch:       spec.Semaphore.Branch,
				PipelineFile: spec.Semaphore.PipelineFile,
				Parameters:   parameters,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: resource.ID(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) taskTriggerParameters(parameters map[string]string) ([]integrations.TaskTriggerParameter, error) {
	parameterValues := []integrations.TaskTriggerParameter{
		{Name: "SEMAPHORE_STAGE_ID", Value: e.execution.StageID.String()},
		{Name: "SEMAPHORE_STAGE_EXECUTION_ID", Value: e.execution.ID.String()},
	}

	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	parameterValues = append(parameterValues, integrations.TaskTriggerParameter{
		Name:  "SEMAPHORE_STAGE_EXECUTION_TOKEN",
		Value: token,
	})

	for key, value := range parameters {
		parameterValues = append(parameterValues, integrations.TaskTriggerParameter{
			Name:  key,
			Value: value,
		})
	}

	return parameterValues, nil
}

func (e *SemaphoreExecutor) runWorkflowParameters(fromSpec map[string]string) (map[string]string, error) {
	parameters := maps.Clone(fromSpec)
	parameters["SEMAPHORE_STAGE_ID"] = e.execution.StageID.String()
	parameters["SEMAPHORE_STAGE_EXECUTION_ID"] = e.execution.ID.String()

	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"] = token
	return parameters, nil
}

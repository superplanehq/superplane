package executors

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	integration integrations.Integration
	resource    integrations.Resource
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

type SemaphoreHook struct {
	Workflow SemaphoreHookWorkflow
	Pipeline SemaphoreHookPipeline
}

type SemaphoreHookWorkflow struct {
	ID string `json:"id"`
}

type SemaphoreHookPipeline struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Result string `json:"result"`
}

func NewSemaphoreExecutor(integration integrations.Integration, resource integrations.Resource) (*SemaphoreExecutor, error) {
	return &SemaphoreExecutor{integration: integration, resource: resource}, nil
}

func (e *SemaphoreExecutor) Name() string {
	return models.ExecutorSpecTypeSemaphore
}

func (e *SemaphoreExecutor) Execute(spec models.ExecutorSpec, parameters ExecutionParameters) (Response, error) {
	if spec.Semaphore.TaskId != nil {
		return e.runTask(spec, parameters)
	}

	return e.runWorkflow(spec, parameters)
}

func (e *SemaphoreExecutor) Check(id string) (Response, error) {
	resource, err := e.integration.Get(integrations.ResourceTypeWorkflow, id)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", id)
	}

	workflow := resource.(*integrations.SemaphoreWorkflow)
	resource, err = e.integration.Get(integrations.ResourceTypePipeline, workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	pipeline := resource.(*integrations.SemaphorePipeline)
	return &SemaphoreResponse{wfID: id, pipeline: pipeline}, nil
}

func (e *SemaphoreExecutor) HandleWebhook(data []byte) (Response, error) {
	var hook SemaphoreHook
	err := json.Unmarshal(data, &hook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling webhook data: %v", err)
	}

	return &SemaphoreResponse{
		wfID: hook.Workflow.ID,
		pipeline: &integrations.SemaphorePipeline{
			PipelineID: hook.Pipeline.ID,
			State:      hook.Pipeline.State,
			Result:     hook.Pipeline.Result,
		},
	}, nil
}

func (e *SemaphoreExecutor) runWorkflow(spec models.ExecutorSpec, parameters ExecutionParameters) (Response, error) {
	workflow, err := e.integration.Create(integrations.ResourceTypeWorkflow, &integrations.CreateWorkflowRequest{
		ProjectID:    e.resource.Id(),
		Reference:    "refs/heads/" + spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   e.workflowParameters(spec.Semaphore.Parameters, parameters),
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) runTask(spec models.ExecutorSpec, parameters ExecutionParameters) (Response, error) {
	workflow, err := e.integration.Create(integrations.ResourceTypeTaskTrigger, &integrations.RunTaskRequest{
		TaskID:       *spec.Semaphore.TaskId,
		Branch:       spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   e.workflowParameters(spec.Semaphore.Parameters, parameters),
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) workflowParameters(paramsFromSpec map[string]string, paramsFromExecution ExecutionParameters) map[string]string {
	parameters := maps.Clone(paramsFromSpec)
	parameters["SEMAPHORE_STAGE_ID"] = paramsFromExecution.StageID
	parameters["SEMAPHORE_STAGE_EXECUTION_ID"] = paramsFromExecution.ExecutionID

	if paramsFromExecution.Token != "" {
		parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"] = paramsFromExecution.Token
	}

	return parameters
}

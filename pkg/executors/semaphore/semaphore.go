package semaphore

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	Integration integrations.Integration
	Resource    integrations.Resource
}

type SemaphoreResponse struct {
	wfID     string
	pipeline *semaphore.SemaphorePipeline
}

// Since a Semaphore execution creates a Semaphore pipeline,
// and a Semaphore pipeline is not finished after the HTTP API call completes,
// we need to monitor the state of the created pipeline.
// That makes the Semaphore executor type async.
func (r *SemaphoreResponse) Finished() bool {
	if r.pipeline == nil {
		return false
	}

	return r.pipeline.State == semaphore.SemaphorePipelineStateDone
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

	return r.pipeline.Result == semaphore.SemaphorePipelineResultPassed
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

func init() {
	executors.Register(models.ExecutorSpecTypeSemaphore, NewSemaphoreExecutor)
}

func NewSemaphoreExecutor(integration integrations.Integration, resource integrations.Resource) (executors.Executor, error) {
	return &SemaphoreExecutor{
		Integration: integration,
		Resource:    resource,
	}, nil
}

func (e *SemaphoreExecutor) Execute(spec models.ExecutorSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	if spec.Semaphore.TaskId != nil {
		return e.runTask(spec, parameters)
	}

	return e.runWorkflow(spec, parameters)
}

func (e *SemaphoreExecutor) Check(id string) (executors.Response, error) {
	resource, err := e.Integration.Get(semaphore.ResourceTypeWorkflow, id)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", id)
	}

	workflow := resource.(*semaphore.SemaphoreWorkflow)
	resource, err = e.Integration.Get(semaphore.ResourceTypePipeline, workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	pipeline := resource.(*semaphore.SemaphorePipeline)
	return &SemaphoreResponse{wfID: id, pipeline: pipeline}, nil
}

func (e *SemaphoreExecutor) HandleWebhook(data []byte) (executors.Response, error) {
	var hook SemaphoreHook
	err := json.Unmarshal(data, &hook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling webhook data: %v", err)
	}

	return &SemaphoreResponse{
		wfID: hook.Workflow.ID,
		pipeline: &semaphore.SemaphorePipeline{
			PipelineID: hook.Pipeline.ID,
			State:      hook.Pipeline.State,
			Result:     hook.Pipeline.Result,
		},
	}, nil
}

func (e *SemaphoreExecutor) runWorkflow(spec models.ExecutorSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	workflow, err := e.Integration.Create(semaphore.ResourceTypeWorkflow, &semaphore.CreateWorkflowRequest{
		ProjectID:    e.Resource.Id(),
		Reference:    "refs/heads/" + spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   e.workflowParameters(spec.Semaphore.Parameters, parameters),
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) runTask(spec models.ExecutorSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	workflow, err := e.Integration.Create(semaphore.ResourceTypeTaskTrigger, &semaphore.RunTaskRequest{
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

func (e *SemaphoreExecutor) workflowParameters(fromSpec map[string]string, fromExecution executors.ExecutionParameters) map[string]string {
	parameters := maps.Clone(fromSpec)
	parameters["SEMAPHORE_STAGE_ID"] = fromExecution.StageID
	parameters["SEMAPHORE_STAGE_EXECUTION_ID"] = fromExecution.ExecutionID

	if fromExecution.Token != "" {
		parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"] = fromExecution.Token
	}

	return parameters
}

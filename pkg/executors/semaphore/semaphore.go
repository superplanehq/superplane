package semaphore

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/google/uuid"
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

func (e *SemaphoreExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec SemaphoreSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if e.Integration == nil {
		return fmt.Errorf("invalid semaphore spec: missing integration")
	}

	return e.validateSemaphoreTask(spec)
}

func (e *SemaphoreExecutor) validateSemaphoreTask(spec SemaphoreSpec) error {
	if spec.Task == "" {
		return nil
	}

	//
	// If task is a UUID, we describe to validate that it exists.
	//
	_, err := uuid.Parse(spec.Task)
	if err == nil {
		_, err := e.Integration.Get(semaphore.ResourceTypeTask, spec.Task, e.Resource.Id())
		if err != nil {
			return fmt.Errorf("task %s not found: %v", spec.Task, err)
		}

		return nil
	}

	//
	// If task is a string, we have to list tasks and find the one that matches.
	//
	tasks, err := e.Integration.List(semaphore.ResourceTypeTask, e.Resource.Id())
	if err != nil {
		return fmt.Errorf("error listing tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Name() == spec.Task {
			return nil
		}
	}

	return fmt.Errorf("task %s not found", spec.Task)
}

type SemaphoreSpec struct {
	Task         string            `json:"task"`
	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipelineFile"`
	Parameters   map[string]string `json:"parameters"`
}

func (e *SemaphoreExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (executors.Response, error) {
	var spec SemaphoreSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.Task != "" {
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

func (e *SemaphoreExecutor) runWorkflow(spec SemaphoreSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	workflow, err := e.Integration.Create(semaphore.ResourceTypeWorkflow, &semaphore.CreateWorkflowRequest{
		ProjectID:    e.Resource.Id(),
		Reference:    "refs/heads/" + spec.Branch,
		PipelineFile: spec.PipelineFile,
		Parameters:   e.workflowParameters(spec.Parameters, parameters),
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) runTask(spec SemaphoreSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	workflow, err := e.Integration.Create(semaphore.ResourceTypeTaskTrigger, &semaphore.RunTaskRequest{
		TaskID:       spec.Task,
		Branch:       spec.Branch,
		PipelineFile: spec.PipelineFile,
		Parameters:   e.workflowParameters(spec.Parameters, parameters),
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

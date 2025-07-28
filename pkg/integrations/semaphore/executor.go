package semaphore

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
)

type SemaphoreExecutor struct {
	Integration integrations.Integration
	Resource    integrations.Resource
}

func NewSemaphoreExecutor(integration integrations.Integration, resource integrations.Resource) (integrations.Executor, error) {
	return &SemaphoreExecutor{
		Integration: integration,
		Resource:    resource,
	}, nil
}

func (e *SemaphoreExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec ExecutorSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.Branch == "" {
		return fmt.Errorf("branch is required")
	}

	return e.validateSemaphoreTask(spec)
}

func (e *SemaphoreExecutor) validateSemaphoreTask(spec ExecutorSpec) error {
	if spec.Task == "" {
		return nil
	}

	//
	// If task is a UUID, we describe to validate that it exists.
	//
	_, err := uuid.Parse(spec.Task)
	if err == nil {
		_, err := e.Integration.Get(ResourceTypeTask, spec.Task, e.Resource.Id())
		if err != nil {
			return fmt.Errorf("task %s not found: %v", spec.Task, err)
		}

		return nil
	}

	//
	// If task is a string, we have to list tasks and find the one that matches.
	//
	tasks, err := e.Integration.List(ResourceTypeTask, e.Resource.Id())
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

type ExecutorSpec struct {
	Task         string            `json:"task"`
	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipelineFile"`
	Parameters   map[string]string `json:"parameters"`
}

func (e *SemaphoreExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (integrations.Resource, error) {
	var spec ExecutorSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.Task != "" {
		return e.runTask(spec, parameters)
	}

	return e.runWorkflow(spec, parameters)
}

func (e *SemaphoreExecutor) runWorkflow(spec ExecutorSpec, parameters executors.ExecutionParameters) (integrations.Resource, error) {
	return e.Integration.Create(ResourceTypeWorkflow, &CreateWorkflowRequest{
		ProjectID:    e.Resource.Id(),
		Reference:    "refs/heads/" + spec.Branch,
		PipelineFile: spec.PipelineFile,
		Parameters:   e.workflowParameters(spec.Parameters, parameters),
	})
}

func (e *SemaphoreExecutor) runTask(spec ExecutorSpec, parameters executors.ExecutionParameters) (integrations.Resource, error) {
	return e.Integration.Create(ResourceTypeTaskTrigger, &RunTaskRequest{
		TaskID:       spec.Task,
		Branch:       spec.Branch,
		PipelineFile: spec.PipelineFile,
		Parameters:   e.workflowParameters(spec.Parameters, parameters),
	})
}

func (e *SemaphoreExecutor) workflowParameters(fromSpec map[string]string, fromExecution executors.ExecutionParameters) map[string]string {
	parameters := maps.Clone(fromSpec)
	parameters["SUPERPLANE_STAGE_ID"] = fromExecution.StageID
	parameters["SUPERPLANE_STAGE_EXECUTION_ID"] = fromExecution.ExecutionID

	if fromExecution.Token != "" {
		parameters["SUPERPLANE_STAGE_EXECUTION_TOKEN"] = fromExecution.Token
	}

	return parameters
}

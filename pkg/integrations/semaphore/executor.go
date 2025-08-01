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
	ResourceManager integrations.ResourceManager
	Resource        integrations.Resource
}

type ExecutorSpec struct {
	Task         string            `json:"task"`
	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipelineFile"`
	Parameters   map[string]string `json:"parameters"`
}

func NewSemaphoreExecutor(resourceManager integrations.ResourceManager, resource integrations.Resource) (integrations.Executor, error) {
	return &SemaphoreExecutor{
		ResourceManager: resourceManager,
		Resource:        resource,
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

	return e.validateTask(spec)
}

func (e *SemaphoreExecutor) validateTask(spec ExecutorSpec) error {
	if spec.Task == "" {
		return nil
	}

	semaphore := e.ResourceManager.(*SemaphoreResourceManager)

	//
	// If task is a UUID, we describe to validate that it exists.
	//
	_, err := uuid.Parse(spec.Task)
	if err == nil {
		_, err := semaphore.getTask(spec.Task, e.Resource.Id())
		if err != nil {
			return fmt.Errorf("task %s not found: %v", spec.Task, err)
		}

		return nil
	}

	//
	// If task is a string, we have to list tasks and find the one that matches.
	//
	tasks, err := semaphore.listTasks(e.Resource.Id())
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

func (e *SemaphoreExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (integrations.StatefulResource, error) {
	var spec ExecutorSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	semaphore := e.ResourceManager.(*SemaphoreResourceManager)
	if spec.Task != "" {
		semaphore.runTask(&RunTaskRequest{
			TaskID:       spec.Task,
			Branch:       spec.Branch,
			PipelineFile: spec.PipelineFile,
			Parameters:   e.workflowParameters(spec.Parameters, parameters),
		})
	}

	return semaphore.runWorkflow(CreateWorkflowRequest{
		ProjectID:    e.Resource.Id(),
		Reference:    "refs/heads/" + spec.Branch,
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

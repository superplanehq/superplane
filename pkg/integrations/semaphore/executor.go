package semaphore

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/manifest"
)

type SemaphoreExecutor struct {
	ResourceManager integrations.ResourceManager
	Resource        integrations.Resource
}

type ExecutorSpec struct {
	Task         string            `json:"task"`
	Ref          string            `json:"ref"`
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

	if spec.Ref == "" {
		return fmt.Errorf("ref is required")
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
			Branch:       strings.TrimPrefix(spec.Ref, "refs/heads/"),
			PipelineFile: spec.PipelineFile,
			Parameters:   e.workflowParameters(spec.Parameters, parameters),
		})
	}

	return semaphore.runWorkflow(CreateWorkflowRequest{
		ProjectID:    e.Resource.Id(),
		Reference:    spec.Ref,
		PipelineFile: spec.PipelineFile,
		Parameters:   e.workflowParameters(spec.Parameters, parameters),
	})
}

func (e *SemaphoreExecutor) workflowParameters(fromSpec map[string]string, fromExecution executors.ExecutionParameters) map[string]string {
	if fromSpec == nil {
		fromSpec = make(map[string]string)
	}
	parameters := maps.Clone(fromSpec)
	parameters["SUPERPLANE_STAGE_ID"] = fromExecution.StageID
	parameters["SUPERPLANE_STAGE_EXECUTION_ID"] = fromExecution.ExecutionID

	if fromExecution.Token != "" {
		parameters["SUPERPLANE_STAGE_EXECUTION_TOKEN"] = fromExecution.Token
	}

	return parameters
}

func (e *SemaphoreExecutor) Manifest() *manifest.TypeManifest {
	return &manifest.TypeManifest{
		Type:            "semaphore",
		DisplayName:     "Semaphore CI",
		Description:     "Execute Semaphore CI workflows and tasks",
		Category:        "executor",
		IntegrationType: "semaphore",
		Icon:            "semaphore",
		Fields: []manifest.FieldManifest{
			{
				Name:         "resource",
				DisplayName:  "Project",
				Type:         manifest.FieldTypeResource,
				Required:     true,
				Description:  "The Semaphore project to execute in",
				ResourceType: "project",
			},
			{
				Name:        "task",
				DisplayName: "Task",
				Type:        manifest.FieldTypeString,
				Required:    false,
				Description: "Optional task name or UUID to trigger",
				Placeholder: "Task name or UUID",
				DependsOn:   "resource",
			},
			{
				Name:        "ref",
				DisplayName: "Git Reference",
				Type:        manifest.FieldTypeString,
				Required:    true,
				Description: "Git branch or tag reference to execute",
				Placeholder: "refs/heads/main",
			},
			{
				Name:        "pipelineFile",
				DisplayName: "Pipeline File",
				Type:        manifest.FieldTypeString,
				Required:    false,
				Description: "Path to the pipeline configuration file",
				Placeholder: ".semaphore/semaphore.yml",
			},
			{
				Name:        "parameters",
				DisplayName: "Parameters",
				Type:        manifest.FieldTypeMap,
				Required:    false,
				Description: "Environment parameters to pass to the workflow",
				Placeholder: "Add workflow parameters",
			},
		},
	}
}

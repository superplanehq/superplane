package executors

import (
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type SemaphoreExecutor struct {
	integration integrations.Integration
	execution   *models.StageExecution
	jwtSigner   *jwt.Signer
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

func NewSemaphoreExecutor(integration integrations.Integration, execution *models.StageExecution, jwtSigner *jwt.Signer) (*SemaphoreExecutor, error) {
	return &SemaphoreExecutor{
		integration: integration,
		execution:   execution,
		jwtSigner:   jwtSigner,
	}, nil
}

func (e *SemaphoreExecutor) Name() string {
	return models.ExecutorSpecTypeSemaphore
}

func (e *SemaphoreExecutor) Execute(spec models.ExecutorSpec, resource integrations.Resource) (Response, error) {
	if spec.Semaphore.TaskId != nil {
		return e.runTask(spec)
	}

	return e.runWorkflow(spec, resource)
}

func (e *SemaphoreExecutor) Check(id string) (Response, error) {
	resource, err := e.integration.Get(semaphore.ResourceTypeWorkflow, id)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found", id)
	}

	workflow := resource.(*semaphore.SemaphoreWorkflow)
	resource, err = e.integration.Get(semaphore.ResourceTypePipeline, workflow.InitialPplID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
	}

	pipeline := resource.(*semaphore.SemaphorePipeline)
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
		pipeline: &semaphore.SemaphorePipeline{
			PipelineID: hook.Pipeline.ID,
			State:      hook.Pipeline.State,
			Result:     hook.Pipeline.Result,
		},
	}, nil
}

func (e *SemaphoreExecutor) runWorkflow(spec models.ExecutorSpec, resource integrations.Resource) (Response, error) {
	parameters, err := e.workflowParameters(spec.Semaphore.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	workflow, err := e.integration.Create(semaphore.ResourceTypeWorkflow, &semaphore.CreateWorkflowRequest{
		ProjectID:    resource.Id(),
		Reference:    "refs/heads/" + spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) runTask(spec models.ExecutorSpec) (Response, error) {
	parameters, err := e.workflowParameters(spec.Semaphore.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	workflow, err := e.integration.Create(semaphore.ResourceTypeTaskTrigger, &semaphore.RunTaskRequest{
		TaskID:       *spec.Semaphore.TaskId,
		Branch:       spec.Semaphore.Branch,
		PipelineFile: spec.Semaphore.PipelineFile,
		Parameters:   parameters,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreResponse{wfID: workflow.Id(), pipeline: nil}, nil
}

func (e *SemaphoreExecutor) workflowParameters(fromSpec map[string]string) (map[string]string, error) {
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

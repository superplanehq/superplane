package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	SemaphorePipelineStateDone    = "done"
	SemaphorePipelineResultPassed = "passed"
	SemaphorePipelineResultFailed = "failed"
)

type SemaphoreIntegration struct {
	URL   string
	Token string
}

func NewSemaphoreIntegration(URL, token string) (Integration, error) {
	return &SemaphoreIntegration{
		URL:   URL,
		Token: token,
	}, nil
}

func (s *SemaphoreIntegration) ListResources(resourceType string) ([]IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeTask:
		return s.listTasks()
	case ResourceTypeProject:
		return s.listProjects()
	default:
		return nil, fmt.Errorf("unsupported resource type %s for list", resourceType)
	}
}

func (s *SemaphoreIntegration) GetResource(resourceType, id string) (IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		return s.getWorkflow(id)
	case ResourceTypePipeline:
		return s.getPipeline(id)
	case ResourceTypeTask:
		return s.getTask(id)
	case ResourceTypeProject:
		return s.getProject(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s for get", resourceType)
	}
}

type CreateWorkflowParams struct {
	ProjectID    string            `json:"project_id"`
	Reference    string            `json:"reference"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
}

func (p *CreateWorkflowParams) For() string {
	return ResourceTypeWorkflow
}

type CreateWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
}

func (s *SemaphoreIntegration) CreateResource(resourceType string, params CreateParams) (IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		return s.createWorkflow(params)
	case ResourceTypeTaskTrigger:
		return s.createTaskTrigger(params)
	default:
		return nil, fmt.Errorf("unsupported resource type %s for create", resourceType)
	}
}

type TaskTriggerParams struct {
	ProjectID   string
	TaskID      string
	TaskTrigger *TaskTrigger
}

func (p *TaskTriggerParams) For() string {
	return ResourceTypeTaskTrigger
}

type TaskTrigger struct {
	Kind       string              `json:"kind"`
	APIVersion string              `json:"apiVersion"`
	Metadata   TaskTriggerMetadata `json:"metadata,omitempty"`
	Spec       TaskTriggerSpec     `json:"spec"`
}

type TaskTriggerMetadata struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
}

type TaskTriggerSpec struct {
	Branch       string                 `json:"branch"`
	PipelineFile string                 `json:"pipeline_file"`
	Parameters   []TaskTriggerParameter `json:"parameters"`
}

type TaskTriggerParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *SemaphoreIntegration) createTaskTrigger(createParams CreateParams) (IntegrationResource, error) {
	params := createParams.(*TaskTriggerParams)
	URL := fmt.Sprintf("%s/api/v2/projects/%s/tasks/%s/triggers", s.URL, params.ProjectID, params.TaskID)

	body, err := json.Marshal(&TaskTrigger{
		APIVersion: "v2",
		Kind:       "TaskTrigger",
		Spec:       params.TaskTrigger.Spec,
	})

	if err != nil {
		return nil, fmt.Errorf("error marshaling task trigger: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	var trigger TaskTrigger
	err = json.Unmarshal(responseBody, &trigger)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if trigger.Metadata.Status != "PASSED" {
		return nil, fmt.Errorf("trigger status was %s", trigger.Metadata.Status)
	}

	return &SemaphoreWorkflow{WfID: trigger.Metadata.WorkflowID}, nil
}

func (s *SemaphoreIntegration) createWorkflow(params CreateParams) (IntegrationResource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows", s.URL)

	body, err := json.Marshal(&params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create workflow params: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building create workflow request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	var response CreateWorkflowResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &SemaphoreWorkflow{WfID: response.WorkflowID}, nil
}

// TODO
func (s *SemaphoreIntegration) listTasks() ([]IntegrationResource, error) {
	return nil, nil
}

// TODO
func (s *SemaphoreIntegration) listProjects() ([]IntegrationResource, error) {
	return nil, nil
}

func (s *SemaphoreIntegration) getWorkflow(id string) (IntegrationResource, error) {
	URL := fmt.Sprintf("%s/api/v2/workflows/%s", s.URL, id)
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	var workflow SemaphoreWorkflow
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &workflow, nil
}

func (s *SemaphoreIntegration) getPipeline(id string) (IntegrationResource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/pipelines/%s", s.URL, id)
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	var pipelineResponse SemaphorePipelineResponse
	err = json.Unmarshal(responseBody, &pipelineResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return pipelineResponse.Pipeline, nil
}

func (s *SemaphoreIntegration) getProject(id string) (IntegrationResource, error) {
	URL := fmt.Sprintf("%s/api/v2/projects/%s", s.URL, id)
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	var project SemaphoreProject
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &project, nil
}

func (s *SemaphoreIntegration) getTask(id string) (IntegrationResource, error) {
	// TODO: task resource is a child of the project resource.
	// How do I get the project ID to use here?
	_ = fmt.Sprintf("%s/api/v2/projects/%s/tasks/%s", s.URL, "???", id)
	return nil, nil
}

type SemaphoreWorkflow struct {
	WfID         string `json:"wf_id"`
	InitialPplID string `json:"initial_ppl_id"`
}

func (s *SemaphoreWorkflow) ID() string {
	return s.WfID
}

func (s *SemaphoreWorkflow) Name() string {
	return ""
}

func (s *SemaphoreWorkflow) Type() string {
	return ResourceTypeWorkflow
}

type SemaphoreProject struct {
	Metadata *SemaphoreProjectMetadata `json:"metadata"`
}

func (s *SemaphoreProject) ID() string {
	return s.Metadata.ProjectID
}

func (s *SemaphoreProject) Name() string {
	return s.Metadata.ProjectName
}

func (s *SemaphoreProject) Type() string {
	return ResourceTypeProject
}

type SemaphoreProjectMetadata struct {
	ProjectName string `json:"name"`
	ProjectID   string `json:"id"`
}

type SemaphorePipelineResponse struct {
	Pipeline *SemaphorePipeline `json:"pipeline"`
}

type SemaphorePipeline struct {
	PipelineName string `json:"name"`
	PipelineID   string `json:"ppl_id"`
	WorkflowID   string `json:"wf_id"`
	State        string `json:"state"`
	Result       string `json:"result"`
}

func (s *SemaphorePipeline) ID() string {
	return s.PipelineID
}

func (s *SemaphorePipeline) Name() string {
	return s.PipelineName
}

func (s *SemaphorePipeline) Type() string {
	return ResourceTypePipeline
}

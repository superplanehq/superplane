package semaphore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/integrations"
)

const (
	PipelineStateDone    = "done"
	PipelineResultPassed = "passed"
	PipelineResultFailed = "failed"

	ResourceTypeTask         = "task"
	ResourceTypeProject      = "project"
	ResourceTypeWorkflow     = "workflow"
	ResourceTypeNotification = "notification"
	ResourceTypeSecret       = "secret"
	ResourceTypePipeline     = "pipeline"
)

type SemaphoreIntegration struct {
	URL   string
	Token string
}

func NewSemaphoreIntegration(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.Integration, error) {
	token, err := authenticate()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %v", err)
	}

	return &SemaphoreIntegration{
		URL:   URL,
		Token: token,
	}, nil
}

func (i *SemaphoreIntegration) Check(resourceType, id string) (integrations.StatefulResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		resource, err := i.getWorkflow(id)
		if err != nil {
			return nil, fmt.Errorf("workflow %s not found", id)
		}

		workflow := resource.(*Workflow)
		resource, err = i.getPipeline(workflow.InitialPplID)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s not found", workflow.InitialPplID)
		}

		pipeline := resource.(*Pipeline)
		return pipeline, nil

	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *SemaphoreIntegration) Get(resourceType, id string) (integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return i.getProject(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

type CreateWorkflowRequest struct {
	ProjectID    string            `json:"project_id"`
	Reference    string            `json:"reference"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
}

type CreateWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
}

func (i *SemaphoreIntegration) Executor(resource integrations.Resource) (integrations.Executor, error) {
	return &SemaphoreExecutor{
		Integration: i,
		Resource:    resource,
	}, nil
}

func (i *SemaphoreIntegration) createSecret(params any) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1beta/secrets", i.URL)

	secret, ok := params.(*Secret)
	if !ok {
		return nil, fmt.Errorf("invalid params type %T", params)
	}

	secret.APIVersion = "v1beta"
	secret.Kind = "Secret"
	body, err := json.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("error marshaling secret: %v", err)
	}

	responseBody, err := i.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response Secret
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

func (i *SemaphoreIntegration) getSecret(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1beta/secrets/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var response Secret
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type Secret struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   SecretMetadata `json:"metadata"`
	Data       SecretSpecData `json:"data"`
}

func (p *Secret) Id() string {
	return p.Metadata.ID
}

func (p *Secret) Name() string {
	return p.Metadata.Name
}

func (p *Secret) Type() string {
	return ResourceTypeSecret
}

type SecretMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SecretSpecData struct {
	EnvVars []SecretSpecDataEnvVar `json:"env_vars"`
}

type SecretSpecDataEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (i *SemaphoreIntegration) getNotification(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/notifications/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var response Notification
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

func (i *SemaphoreIntegration) createNotification(params any) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/notifications", i.URL)

	notification, ok := params.(*Notification)
	if !ok {
		return nil, fmt.Errorf("invalid params type %T", params)
	}

	notification.APIVersion = "v1alpha"
	notification.Kind = "Notification"
	body, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("error marshaling notification: %v", err)
	}

	responseBody, err := i.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response Notification
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type Notification struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Metadata   NotificationMetadata `json:"metadata"`
	Spec       NotificationSpec     `json:"spec"`
}

func (p *Notification) Id() string {
	return p.Metadata.ID
}

func (p *Notification) Name() string {
	return p.Metadata.Name
}

func (p *Notification) Type() string {
	return ResourceTypeNotification
}

type NotificationMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotificationRule struct {
	Name   string                 `json:"name"`
	Filter NotificationRuleFilter `json:"filter"`
	Notify NotificationRuleNotify `json:"notify"`
}

type NotificationRuleNotify struct {
	Webhook NotificationNotifyWebhook `json:"webhook"`

	// TODO
	// we don't really need this, but if it's not in the request,
	// the API does not work properly.
	// Once it's fixed, or we migrate to v2 API, we can remove it from here.
	//
	Slack NotificationNotifySlack `json:"slack"`
}

type NotificationNotifySlack struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Channels []string `json:"channels,omitempty"`
}

type NotificationNotifyWebhook struct {
	Endpoint string `json:"endpoint"`
	Secret   string `json:"secret"`
}

type NotificationRuleFilter struct {
	Branches  []string `json:"branches"`
	Pipelines []string `json:"pipelines"`
	Projects  []string `json:"projects"`
	Results   []string `json:"results"`
}

type NotificationSpec struct {
	Rules []NotificationRule `json:"rules"`
}

type RunTaskRequest struct {
	TaskID       string
	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
}

type RunTaskResponse struct {
	WorkflowID string `json:"workflow_id"`
}

func (i *SemaphoreIntegration) runTask(params any) (integrations.Resource, error) {
	p, ok := params.(*RunTaskRequest)
	if !ok {
		return nil, fmt.Errorf("invalid params type %T", params)
	}

	URL := fmt.Sprintf("%s/api/v1alpha/tasks/%s/run_now", i.URL, p.TaskID)
	body, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("error marshaling task trigger: %v", err)
	}

	responseBody, err := i.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response RunTaskResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &Workflow{WfID: response.WorkflowID}, nil
}

func (i *SemaphoreIntegration) runWorkflow(params any) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows", i.URL)

	body, err := json.Marshal(&params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create workflow params: %v", err)
	}

	responseBody, err := i.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateWorkflowResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &Workflow{WfID: response.WorkflowID}, nil
}

func (i *SemaphoreIntegration) listTasks(parentIDs ...string) ([]integrations.Resource, error) {
	if len(parentIDs) != 1 {
		return nil, fmt.Errorf("expected 1 parent ID, got %d: %v", len(parentIDs), parentIDs)
	}

	URL := fmt.Sprintf("%s/api/v1alpha/tasks?project_id=%s", i.URL, parentIDs[0])
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	err = json.Unmarshal(responseBody, &tasks)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	resources := make([]integrations.Resource, len(tasks))
	for i := range tasks {
		resources[i] = &tasks[i]
	}

	return resources, nil
}

func (i *SemaphoreIntegration) listProjects() ([]integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/projects", i.URL)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	err = json.Unmarshal(responseBody, &projects)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	resources := make([]integrations.Resource, len(projects))
	for i := range projects {
		resources[i] = &projects[i]
	}

	return resources, nil
}

func (i *SemaphoreIntegration) getWorkflow(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v2/workflows/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &workflow, nil
}

func (i *SemaphoreIntegration) getPipeline(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/pipelines/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var pipelineResponse PipelineResponse
	err = json.Unmarshal(responseBody, &pipelineResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return pipelineResponse.Pipeline, nil
}

func (i *SemaphoreIntegration) getProject(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/projects/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &project, nil
}

func (i *SemaphoreIntegration) getTask(id string, parentIDs ...string) (integrations.Resource, error) {
	if len(parentIDs) != 1 {
		return nil, fmt.Errorf("expected 1 parent ID, got %d: %v", len(parentIDs), parentIDs)
	}

	URL := fmt.Sprintf("%s/api/v1alpha/tasks/%s?project_id=%s", i.URL, id, parentIDs[0])
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	type TaskDescribeResponse struct {
		Task *Task `json:"schedule"`
	}

	var task TaskDescribeResponse
	err = json.Unmarshal(responseBody, &task)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return task.Task, nil
}

func (i *SemaphoreIntegration) execRequest(method string, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+i.Token)
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

	return responseBody, nil
}

type Workflow struct {
	WfID         string    `json:"wf_id"`
	InitialPplID string    `json:"initial_ppl_id"`
	Pipeline     *Pipeline `json:"pipeline"`
}

func (w *Workflow) Id() string {
	return w.WfID
}

func (w *Workflow) Name() string {
	return ""
}

func (w *Workflow) Type() string {
	return ResourceTypeWorkflow
}

func (w *Workflow) Finished() bool {
	return w.Pipeline != nil && w.Pipeline.Finished()
}

func (w *Workflow) Successful() bool {
	return w.Pipeline != nil && w.Pipeline.Successful()
}

type Project struct {
	Metadata *ProjectMetadata `json:"metadata"`
}

func (p *Project) Id() string {
	return p.Metadata.ProjectID
}

func (p *Project) Name() string {
	return p.Metadata.ProjectName
}

func (p *Project) Type() string {
	return ResourceTypeProject
}

type ProjectMetadata struct {
	ProjectName string `json:"name"`
	ProjectID   string `json:"id"`
}

type Task struct {
	ID       string `json:"id"`
	TaskName string `json:"name"`
}

func (t *Task) Id() string {
	return t.ID
}

func (t *Task) Name() string {
	return t.TaskName
}

func (t *Task) Type() string {
	return ResourceTypeTask
}

type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

type Pipeline struct {
	PipelineName string `json:"name"`
	PipelineID   string `json:"ppl_id"`
	WorkflowID   string `json:"wf_id"`
	State        string `json:"state"`
	Result       string `json:"result"`
}

func (p *Pipeline) Id() string {
	return p.PipelineID
}

func (p *Pipeline) Name() string {
	return p.PipelineName
}

func (p *Pipeline) Type() string {
	return ResourceTypePipeline
}

func (p *Pipeline) Finished() bool {
	return p.State == PipelineStateDone
}

func (p *Pipeline) Successful() bool {
	return p.Result == PipelineResultPassed
}

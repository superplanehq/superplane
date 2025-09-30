package semaphore

import (
	"bytes"
	"context"
	"crypto/sha256"
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

type SemaphoreResourceManager struct {
	URL   string
	Token string
}

func NewSemaphoreResourceManager(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.ResourceManager, error) {
	token, err := authenticate()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %v", err)
	}

	return &SemaphoreResourceManager{
		URL:   URL,
		Token: token,
	}, nil
}

func (i *SemaphoreResourceManager) Status(resourceType, id string, _ integrations.Resource) (integrations.StatefulResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		resource, err := i.getWorkflow(id)
		if err != nil {
			return nil, fmt.Errorf("workflow %s not found: %v", id, err)
		}

		workflow := resource.(*Workflow)
		resource, err = i.getPipeline(workflow.InitialPplID)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s not found: %v", workflow.InitialPplID, err)
		}

		pipeline := resource.(*Pipeline)
		return pipeline, nil

	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *SemaphoreResourceManager) Cancel(resourceType, id string, _ integrations.Resource) error {
	switch resourceType {
	case ResourceTypeWorkflow:
		return i.stopWorkflow(id)

	default:
		return fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *SemaphoreResourceManager) Get(resourceType, id string) (integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return i.getProject(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *SemaphoreResourceManager) List(ctx context.Context, resourceType string) ([]integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return i.listProjects()
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

func (i *SemaphoreResourceManager) createWebhookSecret(name, key string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1beta/secrets", i.URL)

	secret := &Secret{
		APIVersion: "v1beta",
		Kind:       "Secret",
		Metadata:   SecretMetadata{Name: name},
		Data: SecretSpecData{
			EnvVars: []SecretSpecDataEnvVar{
				{
					Name:  "WEBHOOK_SECRET",
					Value: string(key),
				},
			},
		},
	}

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

func (i *SemaphoreResourceManager) getSecret(id string) (integrations.Resource, error) {
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

// TODO: do we need that here?
func (p *Secret) URL() string {
	return ""
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

func (i *SemaphoreResourceManager) getNotification(id string) (integrations.Resource, error) {
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

func (i *SemaphoreResourceManager) createNotification(params any) (integrations.Resource, error) {
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

// TODO: do we need that here?
func (p *Notification) URL() string {
	return ""
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

func (i *SemaphoreResourceManager) runTask(params any) (integrations.StatefulResource, error) {
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

func (i *SemaphoreResourceManager) runWorkflow(params any) (integrations.StatefulResource, error) {
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

	return &Workflow{
		WfID:            response.WorkflowID,
		OrganizationURL: i.URL,
	}, nil
}

func (i *SemaphoreResourceManager) listTasks(parentIDs ...string) ([]integrations.Resource, error) {
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

func (i *SemaphoreResourceManager) listProjects() ([]integrations.Resource, error) {
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

func (i *SemaphoreResourceManager) getWorkflow(id string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows/%s", i.URL, id)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var workflow WorkflowResponse
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	workflow.Workflow.OrganizationURL = i.URL

	return workflow.Workflow, nil
}

func (i *SemaphoreResourceManager) stopWorkflow(id string) error {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows/%s/terminate", i.URL, id)
	_, err := i.execRequest(http.MethodPost, URL, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *SemaphoreResourceManager) getPipeline(id string) (integrations.Resource, error) {
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

	pipelineResponse.Pipeline.OrganizationURL = i.URL

	return pipelineResponse.Pipeline, nil
}

func (i *SemaphoreResourceManager) getProject(id string) (integrations.Resource, error) {
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

	project.OrganizationURL = i.URL

	return &project, nil
}

func (i *SemaphoreResourceManager) getTask(id string, parentIDs ...string) (integrations.Resource, error) {
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

	task.Task.OrganizationURL = i.URL

	return task.Task, nil
}

func (i *SemaphoreResourceManager) execRequest(method string, url string, body io.Reader) ([]byte, error) {
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

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type WorkflowResponse struct {
	Workflow *Workflow `json:"workflow"`
}

type Workflow struct {
	WfID            string    `json:"wf_id"`
	InitialPplID    string    `json:"initial_ppl_id"`
	Pipeline        *Pipeline `json:"pipeline"`
	ProjectID       string    `json:"project_id"`
	OrganizationURL string
}

func (w *Workflow) Id() string {
	return w.WfID
}

func (w *Workflow) Name() string {
	return ""
}

func (w *Workflow) URL() string {
	return fmt.Sprintf("%s/workflows/%s", w.OrganizationURL, w.WfID)
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
	Metadata        *ProjectMetadata `json:"metadata"`
	OrganizationURL string
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

func (p *Project) URL() string {
	return fmt.Sprintf("%s/projects/%s", p.OrganizationURL, p.Id())
}

type ProjectMetadata struct {
	ProjectName string `json:"name"`
	ProjectID   string `json:"id"`
}

type Task struct {
	ID              string `json:"id"`
	TaskName        string `json:"name"`
	ProjectId       string `json:"project_id"`
	OrganizationURL string
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

func (t *Task) URL() string {
	return fmt.Sprintf("%s/projects/%s/schedulers/%s", t.OrganizationURL, t.ProjectId, t.Id())
}

type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

type Pipeline struct {
	PipelineName    string `json:"name"`
	PipelineID      string `json:"ppl_id"`
	WorkflowID      string `json:"wf_id"`
	State           string `json:"state"`
	Result          string `json:"result"`
	ProjectID       string `json:"project_id"`
	OrganizationURL string
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

func (p *Pipeline) URL() string {
	return fmt.Sprintf("%s/workflows/%s?pipeline_id=%s", p.OrganizationURL, p.WorkflowID, p.Id())
}

func (p *Pipeline) Finished() bool {
	return p.State == PipelineStateDone
}

func (p *Pipeline) Successful() bool {
	return p.Result == PipelineResultPassed
}

func (i *SemaphoreResourceManager) SetupWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	//
	// Semaphore doesn't let us use UUIDs in secret names,
	// so we sha256 the ID before creating the secret.
	//
	hash := sha256.New()
	hash.Write([]byte(options.ID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	resourceName := fmt.Sprintf("superplane-webhook-%x", suffix[:16])

	//
	// Create Semaphore secret to store the event source key.
	//
	secret, err := i.createSemaphoreSecret(resourceName, options.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore secret: %v", err)
	}

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := i.createSemaphoreNotification(resourceName, options)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore notification: %v", err)
	}

	return []integrations.Resource{secret, notification}, nil
}

func (i *SemaphoreResourceManager) createSemaphoreSecret(name string, key []byte) (integrations.Resource, error) {
	//
	// Check if secret already exists.
	//
	secret, err := i.getSecret(name)
	if err == nil {
		return secret, nil
	}

	//
	// Secret does not exist, create it.
	//
	secret, err = i.createWebhookSecret(name, string(key))
	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	return secret, nil
}

func (i *SemaphoreResourceManager) createSemaphoreNotification(name string, options integrations.WebhookOptions) (integrations.Resource, error) {
	//
	// Check if notification already exists.
	//
	notification, err := i.getNotification(name)
	if err == nil {
		return notification, nil
	}

	//
	// Notification does not exist, create it.
	//
	notification, err = i.createNotification(&Notification{
		Metadata: NotificationMetadata{
			Name: name,
		},
		Spec: NotificationSpec{
			Rules: []NotificationRule{
				{
					Name: fmt.Sprintf("webhook-for-%s", options.Parent.Name()),
					Filter: NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{options.Parent.Name()},
						Results:   []string{},
					},
					Notify: NotificationRuleNotify{
						Webhook: NotificationNotifyWebhook{
							Endpoint: options.URL,
							Secret:   name,
						},
					},
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating notification: %v", err)
	}

	return notification, nil
}

func (i *SemaphoreResourceManager) CleanupWebhook(parentResource integrations.Resource, webhook integrations.Resource) error {
	// For Semaphore, we need to delete both the notification and the associated secret
	// We'll use DELETE HTTP method to clean up the resources

	// Delete notification
	if webhook.Type() == ResourceTypeNotification {
		notificationURL := fmt.Sprintf("%s/api/v1alpha/notifications/%s", i.URL, webhook.Id())
		_, err := i.execRequest(http.MethodDelete, notificationURL, nil)
		if err != nil {
			return fmt.Errorf("error deleting notification: %v", err)
		}
	}

	// For secrets, we can attempt to delete them by name pattern
	// Since we created secrets with a specific naming convention
	if webhook.Type() == ResourceTypeSecret {
		secretURL := fmt.Sprintf("%s/api/v1beta/secrets/%s", i.URL, webhook.Name())
		_, err := i.execRequest(http.MethodDelete, secretURL, nil)
		if err != nil {
			return fmt.Errorf("error deleting secret: %v", err)
		}
	}

	return nil
}

package semaphore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/integrations"
)

const (
	PipelineStateDone    = "done"
	PipelineResultPassed = "passed"
	PipelineResultFailed = "failed"

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

func (i *SemaphoreResourceManager) List(resourceTypes string) ([]integrations.Resource, error) {
	switch resourceTypes {
	case ResourceTypeProject:
		return i.listProjects()
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceTypes)
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
	PipelineID string `json:"pipeline_id"`
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

func (p *Secret) URL() string {
	return ""
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

func (p *Notification) URL() string {
	return ""
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

func (i *SemaphoreResourceManager) RunWorkflow(params any) (*Workflow, error) {
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
		WfID:         response.WorkflowID,
		InitialPplID: response.PipelineID,
		WorkflowURL:  fmt.Sprintf("%s/workflows/%s", i.URL, response.WorkflowID),
		Pipeline: &Pipeline{
			ID: response.PipelineID,
		},
	}, nil
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
	for idx, project := range projects {
		projects[idx].ProjectURL = fmt.Sprintf("%s/projects/%s", i.URL, project.Id())
		resources[idx] = &projects[idx]
	}

	return resources, nil
}

func (i *SemaphoreResourceManager) stopWorkflow(id string) error {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows/%s/terminate", i.URL, id)
	_, err := i.execRequest(http.MethodPost, URL, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *SemaphoreResourceManager) GetPipeline(id string) (*Pipeline, error) {
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

func (i *SemaphoreResourceManager) getProject(idOrName string) (integrations.Resource, error) {
	_, err := uuid.Parse(idOrName)
	if err != nil {
		return i.getProjectByName(idOrName)
	}

	projects, err := i.listProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Id() == idOrName {
			return project, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", idOrName)
}

func (i *SemaphoreResourceManager) getProjectByName(name string) (integrations.Resource, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/projects/%s", i.URL, name)
	responseBody, err := i.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	project.ProjectURL = fmt.Sprintf("%s/projects/%s", i.URL, name)
	return &project, nil
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
	WorkflowURL  string    `json:"-"`
	WfID         string    `json:"wf_id" mapstructure:"wf_id"`
	InitialPplID string    `json:"initial_ppl_id" mapstructure:"initial_ppl_id"`
	Pipeline     *Pipeline `json:"pipeline" mapstructure:"pipeline"`
}

type Project struct {
	ProjectURL string           `json:"-"`
	Metadata   *ProjectMetadata `json:"metadata"`
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
	return p.ProjectURL
}

type ProjectMetadata struct {
	ProjectName string `json:"name"`
	ProjectID   string `json:"id"`
}

type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

type Pipeline struct {
	URL              string `json:"-"`
	ID               string `json:"ppl_id" mapstructure:"ppl_id"`
	Name             string `json:"name" mapstructure:"name"`
	State            string `json:"state" mapstructure:"state"`
	Result           string `json:"result" mapstructure:"result"`
	ResultReason     string `json:"result_reason" mapstructure:"result_reason"`
	CreatedAt        string `json:"created_at" mapstructure:"created_at"`
	PendingAt        string `json:"pending_at" mapstructure:"pending_at"`
	QueuingAt        string `json:"queuing_at" mapstructure:"queuing_at"`
	RunningAt        string `json:"running_at" mapstructure:"running_at"`
	StoppingAt       string `json:"stopping_at" mapstructure:"stopping_at"`
	DoneAt           string `json:"done_at" mapstructure:"done_at"`
	TerminatedBy     string `json:"terminated_by" mapstructure:"terminated_by"`
	TerminateRequest string `json:"terminate_request" mapstructure:"terminate_request"`
}

type WebhookMetadata struct {
	Secret       WebhookSecretMetadata       `json:"secret"`
	Notification WebhookNotificationMetadata `json:"notification"`
}

type WebhookSecretMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WebhookNotificationMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (i *SemaphoreResourceManager) SetupWebhook(options integrations.WebhookOptions) (any, error) {
	//
	// Semaphore doesn't let us use UUIDs in secret names,
	// so we sha256 the ID before creating the secret.
	//
	hash := sha256.New()
	hash.Write([]byte(options.ID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%x", suffix[:16])

	//
	// Create Semaphore secret to store the event source key.
	//
	secret, err := i.createSemaphoreSecret(name, options.Secret)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore secret: %v", err)
	}

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := i.createSemaphoreNotification(name, options.URL, options.Resource)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore notification: %v", err)
	}

	return WebhookMetadata{
		Secret:       WebhookSecretMetadata{ID: secret.Id(), Name: secret.Name()},
		Notification: WebhookNotificationMetadata{ID: notification.Id(), Name: notification.Name()},
	}, nil
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

func (i *SemaphoreResourceManager) createSemaphoreNotification(name string, URL string, resource integrations.Resource) (integrations.Resource, error) {
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
					Name: fmt.Sprintf("webhook-for-%s", resource.Name()),
					Filter: NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{resource.Name()},
						Results:   []string{},
					},
					Notify: NotificationRuleNotify{
						Webhook: NotificationNotifyWebhook{
							Endpoint: URL,
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

func (i *SemaphoreResourceManager) CleanupWebhook(options integrations.WebhookOptions) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(options.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	// Delete notification
	notificationURL := fmt.Sprintf("%s/api/v1alpha/notifications/%s", i.URL, metadata.Notification.ID)
	_, err = i.execRequest(http.MethodDelete, notificationURL, nil)
	if err != nil {
		return fmt.Errorf("error deleting notification: %v", err)
	}

	// For secrets, we can attempt to delete them by name pattern
	// Since we created secrets with a specific naming convention
	secretURL := fmt.Sprintf("%s/api/v1beta/secrets/%s", i.URL, metadata.Secret.Name)
	_, err = i.execRequest(http.MethodDelete, secretURL, nil)
	if err != nil {
		return fmt.Errorf("error deleting secret: %v", err)
	}

	return nil
}

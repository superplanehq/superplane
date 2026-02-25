package semaphore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	OrgURL   string
	APIToken string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	orgURL, err := ctx.GetConfig("organizationUrl")
	if err != nil {
		return nil, err
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		OrgURL:   string(orgURL),
		APIToken: string(apiToken),
		http:     http,
	}, nil
}

type ProjectResponse struct {
	Metadata *ProjectMetadata `json:"metadata"`
}

type ProjectMetadata struct {
	ProjectName string `json:"name"`
	ProjectID   string `json:"id"`
}

func (c *Client) GetProject(idOrName string) (*ProjectResponse, error) {
	_, err := uuid.Parse(idOrName)
	if err != nil {
		return c.getProjectByName(idOrName)
	}

	projects, err := c.listProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Metadata.ProjectID == idOrName {
			return &project, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", idOrName)
}

func (c *Client) getProjectByName(name string) (*ProjectResponse, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/projects/%s", c.OrgURL, name)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var project ProjectResponse
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &project, nil
}

func (c *Client) listProjects() ([]ProjectResponse, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/projects", c.OrgURL)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var projects []ProjectResponse
	err = json.Unmarshal(responseBody, &projects)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return projects, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+c.APIToken)
	res, err := c.http.Do(req)
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

type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

type Pipeline struct {
	PipelineName     string `json:"name"`
	PipelineID       string `json:"ppl_id"`
	WorkflowID       string `json:"wf_id"`
	State            string `json:"state"`
	Result           string `json:"result"`
	ResultReason     string `json:"result_reason"`
	BranchName       string `json:"branch_name"`
	CommitSHA        string `json:"commit_sha"`
	CommitMessage    string `json:"commit_message"`
	YAMLFileName     string `json:"yaml_file_name"`
	WorkingDirectory string `json:"working_directory"`
	ProjectID        string `json:"project_id"`
	CreatedAt        string `json:"created_at"`
	DoneAt           string `json:"done_at"`
	RunningAt        string `json:"running_at"`
	ErrorDescription string `json:"error_description"`
	TerminatedBy     string `json:"terminated_by"`
	PromotionOf      string `json:"promotion_of"`
}

func (c *Client) GetPipeline(id string) (*Pipeline, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/pipelines/%s", c.OrgURL, id)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
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

func (c *Client) ListPipelines(projectID string) ([]any, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/pipelines?project_id=%s", c.OrgURL, projectID)
	response, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var pipelines []any
	err = json.Unmarshal(response, &pipelines)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return pipelines, nil
}

type CreateWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	PipelineID string `json:"pipeline_id"`
}

func (c *Client) RunWorkflow(params any) (*CreateWorkflowResponse, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/plumber-workflows", c.OrgURL)
	body, err := json.Marshal(&params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create workflow params: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateWorkflowResponse
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

func (c *Client) GetNotification(id string) (*Notification, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/notifications/%s", c.OrgURL, id)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
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

func (c *Client) CreateNotification(params any) (*Notification, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/notifications", c.OrgURL)

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

	responseBody, err := c.execRequest(http.MethodPost, URL, bytes.NewReader(body))
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

func (c *Client) DeleteNotification(id string) error {
	notificationURL := fmt.Sprintf("%s/api/v1alpha/notifications/%s", c.OrgURL, id)
	_, err := c.execRequest(http.MethodDelete, notificationURL, nil)
	if err != nil {
		return fmt.Errorf("error deleting notification: %v", err)
	}

	return nil
}

type Secret struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   SecretMetadata `json:"metadata"`
	Data       SecretSpecData `json:"data"`
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

func (c *Client) GetSecret(id string) (*Secret, error) {
	URL := fmt.Sprintf("%s/api/v1beta/secrets/%s", c.OrgURL, id)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
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

func (c *Client) DeleteSecret(name string) error {
	secretURL := fmt.Sprintf("%s/api/v1beta/secrets/%s", c.OrgURL, name)
	_, err := c.execRequest(http.MethodDelete, secretURL, nil)
	if err != nil {
		return fmt.Errorf("error deleting secret: %v", err)
	}

	return nil
}

func (c *Client) CreateWebhookSecret(name, key string) (*Secret, error) {
	URL := fmt.Sprintf("%s/api/v1beta/secrets", c.OrgURL)

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

	responseBody, err := c.execRequest(http.MethodPost, URL, bytes.NewReader(body))
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

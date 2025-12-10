package semaphore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
)

type Client struct {
	OrgURL   string
	APIToken string
}

func NewClient(ctx components.AppInstallationContext) (*Client, error) {
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

type PipelineResponse struct {
	Pipeline *PipelineX `json:"pipeline"`
}

type PipelineX struct {
	PipelineName string `json:"name"`
	PipelineID   string `json:"ppl_id"`
	WorkflowID   string `json:"wf_id"`
	State        string `json:"state"`
	Result       string `json:"result"`
}

func (c *Client) GetPipeline(id string) (*PipelineX, error) {
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

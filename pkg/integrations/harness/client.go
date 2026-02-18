package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://app.harness.io"

type Client struct {
	AccountID string
	APIToken  string
	http      core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	accountID, err := ctx.GetConfig("accountId")
	if err != nil {
		return nil, err
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		AccountID: string(accountID),
		APIToken:  string(apiToken),
		http:      http,
	}, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIToken)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// ValidateCredentials verifies that the API token and account ID are valid
// by making a lightweight API call.
// Uses the account settings endpoint which works with both user and service account tokens.
func (c *Client) ValidateCredentials() error {
	URL := fmt.Sprintf("%s/ng/api/accounts/%s?accountIdentifier=%s", BaseURL, c.AccountID, c.AccountID)
	_, err := c.execRequest(http.MethodGet, URL, nil)
	return err
}

// ExecutePipelineResponse represents the response from executing a pipeline.
type ExecutePipelineResponse struct {
	Status string                        `json:"status"`
	Data   *ExecutePipelineResponseData   `json:"data"`
}

type ExecutePipelineResponseData struct {
	PlanExecution *PlanExecution `json:"planExecution"`
}

type PlanExecution struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

// ExecutePipeline triggers a pipeline execution in Harness.
func (c *Client) ExecutePipeline(org, project, pipeline, module string) (*ExecutePipelineResponse, error) {
	URL := fmt.Sprintf(
		"%s/pipeline/api/pipeline/execute/%s?accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s",
		BaseURL, pipeline, c.AccountID, org, project,
	)

	if module != "" {
		URL += "&module=" + module
	}

	// Empty body — no runtime inputs for basic execution
	body := bytes.NewReader([]byte(""))
	req, err := http.NewRequest(http.MethodPost, URL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("x-api-key", c.APIToken)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("execute pipeline got %d code: %s", res.StatusCode, string(responseBody))
	}

	var response ExecutePipelineResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

// PipelineExecutionDetails represents the execution details response.
type PipelineExecutionDetails struct {
	Status string                          `json:"status"`
	Data   *PipelineExecutionDetailsData   `json:"data"`
}

type PipelineExecutionDetailsData struct {
	PipelineExecution *PipelineExecutionSummary `json:"pipelineExecutionSummary"`
}

type PipelineExecutionSummary struct {
	PipelineIdentifier string `json:"pipelineIdentifier"`
	PlanExecutionID    string `json:"planExecutionId"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	StartTs            int64  `json:"startTs"`
	EndTs              int64  `json:"endTs"`
	ExecutionTriggerInfo *ExecutionTriggerInfo `json:"executionTriggerInfo"`
}

type ExecutionTriggerInfo struct {
	TriggerType string `json:"triggerType"`
	TriggeredBy *TriggeredBy `json:"triggeredBy"`
}

type TriggeredBy struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

// GetPipelineExecution retrieves the execution details for a pipeline run.
func (c *Client) GetPipelineExecution(org, project, planExecutionID string) (*PipelineExecutionDetails, error) {
	URL := fmt.Sprintf(
		"%s/pipeline/api/pipelines/execution/v2/%s?accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s",
		BaseURL, planExecutionID, c.AccountID, org, project,
	)

	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var response PipelineExecutionDetails
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

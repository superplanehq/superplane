package codepipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const TargetPrefix = "CodePipeline_20150709."

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type GetPipelineResponseBody struct {
	Pipeline map[string]any             `json:"pipeline"`
	Metadata PipelineDefinitionMetadata `json:"metadata"`
}

type PipelineDefinitionMetadata struct {
	PipelineARN       string           `json:"pipelineArn"`
	Created           common.FloatTime `json:"created"`
	Updated           common.FloatTime `json:"updated"`
	PollingDisabledAt common.FloatTime `json:"pollingDisabledAt,omitempty"`
}

func (c *Client) GetPipeline(name string) (*GetPipelineResponseBody, error) {
	payload := map[string]any{
		"name": name,
	}

	var response GetPipelineResponseBody
	if err := c.postJSON("GetPipeline", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

type StartPipelineExecutionResponse struct {
	PipelineExecutionID string `json:"pipelineExecutionId"`
}

func (c *Client) StartPipelineExecution(pipelineName string) (*StartPipelineExecutionResponse, error) {
	payload := map[string]any{
		"name": pipelineName,
	}

	var response StartPipelineExecutionResponse
	if err := c.postJSON("StartPipelineExecution", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

type PipelineExecution struct {
	PipelineExecutionID string `json:"pipelineExecutionId"`
	Status              string `json:"status"`
	PipelineName        string `json:"pipelineName"`
}

type GetPipelineExecutionResponse struct {
	PipelineExecution PipelineExecution `json:"pipelineExecution"`
}

func (c *Client) GetPipelineExecution(pipelineName, executionID string) (*PipelineExecution, error) {
	payload := map[string]any{
		"pipelineName":        pipelineName,
		"pipelineExecutionId": executionID,
	}

	var response GetPipelineExecutionResponse
	if err := c.postJSON("GetPipelineExecution", payload, &response); err != nil {
		return nil, err
	}

	return &response.PipelineExecution, nil
}

type GetPipelineExecutionDetailsResponse struct {
	PipelineExecution map[string]any `json:"pipelineExecution"`
}

func (c *Client) GetPipelineExecutionDetails(pipelineName, executionID string) (*GetPipelineExecutionDetailsResponse, error) {
	payload := map[string]any{
		"pipelineName":        pipelineName,
		"pipelineExecutionId": executionID,
	}

	var response GetPipelineExecutionDetailsResponse
	if err := c.postJSON("GetPipelineExecution", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) StopPipelineExecution(pipelineName, executionID, reason string, abandon bool) error {
	payload := map[string]any{
		"pipelineName":        pipelineName,
		"pipelineExecutionId": executionID,
		"abandon":             abandon,
		"reason":              reason,
	}

	return c.postJSON("StopPipelineExecution", payload, nil)
}

// PipelineSummary uses Name as the identifier because AWS ListPipelines
// does not return ARN in the response.
type PipelineSummary struct {
	Name string `json:"name"`
}

type ListPipelinesResponse struct {
	Pipelines []PipelineSummary `json:"pipelines"`
	NextToken string            `json:"nextToken"`
}

func (c *Client) ListPipelines() ([]PipelineSummary, error) {
	pipelines := []PipelineSummary{}
	nextToken := ""

	for {
		payload := map[string]any{}
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		var response ListPipelinesResponse
		if err := c.postJSON("ListPipelines", payload, &response); err != nil {
			return nil, err
		}

		pipelines = append(pipelines, response.Pipelines...)
		if response.NextToken == "" {
			break
		}
		nextToken = response.NextToken
	}

	return pipelines, nil
}

func (c *Client) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://codepipeline.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", TargetPrefix+action)

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("CodePipeline API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "codepipeline", c.region, time.Now())
}

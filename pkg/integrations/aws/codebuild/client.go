package codebuild

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

const TargetPrefix = "CodeBuild_20161006."

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

type StartBuildResponse struct {
	Build BuildDetail `json:"build"`
}

type StopBuildResponse struct {
	Build BuildDetail `json:"build"`
}

type BatchGetBuildsResponse struct {
	Builds         []BuildDetail `json:"builds"`
	BuildsNotFound []string      `json:"buildsNotFound"`
}

type BuildDetail struct {
	ID            string         `json:"id"`
	ARN           string         `json:"arn"`
	BuildNumber   int            `json:"buildNumber"`
	BuildStatus   string         `json:"buildStatus"`
	ProjectName   string         `json:"projectName"`
	CurrentPhase  string         `json:"currentPhase"`
	StartTime     float64        `json:"startTime"`
	EndTime       float64        `json:"endTime"`
	Initiator     string         `json:"initiator"`
	Source        map[string]any `json:"source"`
	Environment   map[string]any `json:"environment"`
	Logs          map[string]any `json:"logs"`
	BuildComplete bool           `json:"buildComplete"`
}

func (c *Client) StartBuild(projectName string, envOverrides []EnvironmentVariable) (*BuildDetail, error) {
	payload := map[string]any{
		"projectName": projectName,
	}

	if len(envOverrides) > 0 {
		overrides := make([]map[string]string, len(envOverrides))
		for i, v := range envOverrides {
			overrides[i] = map[string]string{
				"name":  v.Name,
				"value": v.Value,
				"type":  "PLAINTEXT",
			}
		}
		payload["environmentVariablesOverride"] = overrides
	}

	var response StartBuildResponse
	if err := c.postJSON("StartBuild", payload, &response); err != nil {
		return nil, err
	}

	return &response.Build, nil
}

type EnvironmentVariable struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

func (c *Client) StopBuild(buildID string) (*BuildDetail, error) {
	payload := map[string]any{
		"id": buildID,
	}

	var response StopBuildResponse
	if err := c.postJSON("StopBuild", payload, &response); err != nil {
		return nil, err
	}

	return &response.Build, nil
}

func (c *Client) BatchGetBuilds(buildIDs []string) ([]BuildDetail, error) {
	payload := map[string]any{
		"ids": buildIDs,
	}

	var response BatchGetBuildsResponse
	if err := c.postJSON("BatchGetBuilds", payload, &response); err != nil {
		return nil, err
	}

	return response.Builds, nil
}

type ProjectSummary struct {
	Name string `json:"name"`
	ARN  string `json:"arn"`
}

type ListProjectsResponse struct {
	Projects  []string `json:"projects"`
	NextToken string   `json:"nextToken"`
}

type BatchGetProjectsResponse struct {
	Projects []ProjectSummary `json:"projects"`
}

func (c *Client) ListProjects() ([]string, error) {
	projects := []string{}
	nextToken := ""

	for {
		payload := map[string]any{}
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		var response ListProjectsResponse
		if err := c.postJSON("ListProjects", payload, &response); err != nil {
			return nil, err
		}

		projects = append(projects, response.Projects...)
		if response.NextToken == "" {
			break
		}
		nextToken = response.NextToken
	}

	return projects, nil
}

func (c *Client) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://codebuild.%s.amazonaws.com/", c.region)
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
		return fmt.Errorf("CodeBuild API request failed with %d: %s", res.StatusCode, string(responseBody))
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
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "codebuild", c.region, time.Now())
}

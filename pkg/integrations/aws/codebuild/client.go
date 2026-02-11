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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const targetPrefix = "CodeBuild_20161006."

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type Build struct {
	ID           string           `json:"id"`
	Arn          string           `json:"arn"`
	BuildNumber  int64            `json:"buildNumber"`
	CurrentPhase string           `json:"currentPhase"`
	BuildStatus  string           `json:"buildStatus"`
	ProjectName  string           `json:"projectName"`
	SourceVersion string          `json:"sourceVersion"`
	Initiator    string           `json:"initiator"`
	StartTime    common.FloatTime `json:"startTime,omitempty"`
	EndTime      common.FloatTime `json:"endTime,omitempty"`
	Logs         *BuildLogs       `json:"logs,omitempty"`
}

type BuildLogs struct {
	DeepLink         string `json:"deepLink"`
	CloudWatchLogsArn string `json:"cloudWatchLogsArn"`
	GroupName        string `json:"groupName"`
	StreamName       string `json:"streamName"`
	Status           string `json:"status"`
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) ListProjects() ([]string, error) {
	projects := []string{}
	nextToken := ""

	for {
		payload := map[string]any{}
		if strings.TrimSpace(nextToken) != "" {
			payload["nextToken"] = strings.TrimSpace(nextToken)
		}

		var response struct {
			Projects  []string `json:"projects"`
			NextToken string   `json:"nextToken"`
		}

		if err := c.postJSON("ListProjects", payload, &response); err != nil {
			return nil, err
		}

		projects = append(projects, response.Projects...)
		if strings.TrimSpace(response.NextToken) == "" {
			break
		}

		nextToken = response.NextToken
	}

	return projects, nil
}

func (c *Client) DescribeProject(projectName string) (*Project, error) {
	projects, err := c.BatchGetProjects([]string{projectName})
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	return &projects[0], nil
}

func (c *Client) BatchGetProjects(projectNames []string) ([]Project, error) {
	if len(projectNames) == 0 {
		return []Project{}, nil
	}

	payload := map[string]any{
		"names": projectNames,
	}

	var response struct {
		Projects []struct {
			Name string `json:"name"`
			Arn  string `json:"arn"`
		} `json:"projects"`
	}

	if err := c.postJSON("BatchGetProjects", payload, &response); err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(response.Projects))
	for _, project := range response.Projects {
		projects = append(projects, Project{
			ProjectName: project.Name,
			ProjectArn:  project.Arn,
		})
	}

	return projects, nil
}

func (c *Client) StartBuild(projectName string, sourceVersion string) (*Build, error) {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	payload := map[string]any{
		"projectName": projectName,
	}
	if strings.TrimSpace(sourceVersion) != "" {
		payload["sourceVersion"] = strings.TrimSpace(sourceVersion)
	}

	var response struct {
		Build Build `json:"build"`
	}

	if err := c.postJSON("StartBuild", payload, &response); err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.Build.ID) == "" {
		return nil, fmt.Errorf("start build response missing build ID")
	}

	return &response.Build, nil
}

func (c *Client) BatchGetBuilds(buildIDs []string) ([]Build, error) {
	if len(buildIDs) == 0 {
		return nil, fmt.Errorf("build IDs are required")
	}

	payload := map[string]any{
		"ids": buildIDs,
	}

	var response struct {
		Builds         []Build  `json:"builds"`
		BuildsNotFound []string `json:"buildsNotFound"`
	}

	if err := c.postJSON("BatchGetBuilds", payload, &response); err != nil {
		return nil, err
	}

	if len(response.Builds) == 0 && len(response.BuildsNotFound) > 0 {
		return nil, fmt.Errorf("build not found: %s", strings.Join(response.BuildsNotFound, ", "))
	}

	return response.Builds, nil
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
	req.Header.Set("X-Amz-Target", targetPrefix+action)

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

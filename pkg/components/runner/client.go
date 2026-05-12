package runner

import (
	"bytes"
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
	awscommon "github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	codeBuildTargetPrefix      = "CodeBuild_20161006."
	cloudWatchLogsTargetPrefix = "Logs_20140328."
)

type codeBuildClient struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func newCodeBuildClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *codeBuildClient {
	return &codeBuildClient{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type environmentVariableOverride struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type startBuildInput struct {
	ProjectName                  string                        `json:"projectName"`
	BuildspecOverride            string                        `json:"buildspecOverride"`
	SourceTypeOverride           string                        `json:"sourceTypeOverride"`
	EnvironmentVariablesOverride []environmentVariableOverride `json:"environmentVariablesOverride,omitempty"`
	TimeoutInMinutesOverride     int                           `json:"timeoutInMinutesOverride,omitempty"`
}

type startBuildResponse struct {
	Build build `json:"build"`
}

type batchGetBuildsResponse struct {
	Builds []build `json:"builds"`
}

type build struct {
	ID          string              `json:"id"`
	ARN         string              `json:"arn"`
	BuildStatus string              `json:"buildStatus"`
	StartTime   awscommon.FloatTime `json:"startTime"`
	EndTime     awscommon.FloatTime `json:"endTime"`
	Logs        buildLogs           `json:"logs"`
}

type buildLogs struct {
	GroupName  string `json:"groupName"`
	StreamName string `json:"streamName"`
	DeepLink   string `json:"deepLink"`
}

func (c *codeBuildClient) startBuild(input startBuildInput) (*build, error) {
	var response startBuildResponse
	if err := c.postCodeBuildJSON("StartBuild", input, &response); err != nil {
		return nil, err
	}

	return &response.Build, nil
}

func (c *codeBuildClient) getBuild(id string) (*build, error) {
	var response batchGetBuildsResponse
	if err := c.postCodeBuildJSON("BatchGetBuilds", map[string]any{"ids": []string{id}}, &response); err != nil {
		return nil, err
	}

	if len(response.Builds) == 0 {
		return nil, fmt.Errorf("run not found: %s", id)
	}

	return &response.Builds[0], nil
}

func (c *codeBuildClient) stopBuild(id string) (*build, error) {
	var response startBuildResponse
	if err := c.postCodeBuildJSON("StopBuild", map[string]any{"id": id}, &response); err != nil {
		return nil, err
	}

	return &response.Build, nil
}

func (c *codeBuildClient) getLogEvents(groupName, streamName string) ([]logEvent, error) {
	events, _, err := c.getLogEventsPaged(groupName, streamName, "")
	return events, err
}

const cloudWatchLogEventsPageLimit = 10000

func (c *codeBuildClient) getLogEventsPaged(groupName, streamName, nextForwardToken string) ([]logEvent, string, error) {
	if groupName == "" || streamName == "" {
		return nil, "", nil
	}

	payload := map[string]any{
		"logGroupName":  groupName,
		"logStreamName": streamName,
		"limit":         cloudWatchLogEventsPageLimit,
	}
	if strings.TrimSpace(nextForwardToken) != "" {
		payload["nextToken"] = nextForwardToken
	} else {
		payload["startFromHead"] = true
	}

	var response getLogEventsResponse
	err := c.postCloudWatchLogsJSON("GetLogEvents", payload, &response)
	if err != nil {
		return nil, "", err
	}

	return response.Events, response.NextForwardToken, nil
}

type getLogEventsResponse struct {
	Events           []logEvent `json:"events"`
	NextForwardToken string     `json:"nextForwardToken,omitempty"`
}

type logEvent struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func (c *codeBuildClient) postCodeBuildJSON(action string, payload any, out any) error {
	endpoint := fmt.Sprintf("https://codebuild.%s.amazonaws.com/", c.region)
	return c.postJSON(endpoint, "codebuild", codeBuildTargetPrefix+action, payload, out)
}

func (c *codeBuildClient) postCloudWatchLogsJSON(action string, payload any, out any) error {
	endpoint := fmt.Sprintf("https://logs.%s.amazonaws.com/", c.region)
	return c.postJSON(endpoint, "logs", cloudWatchLogsTargetPrefix+action, payload, out)
}

func (c *codeBuildClient) postJSON(endpoint, service, target string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)

	if err := c.signRequest(req, body, service); err != nil {
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
		if awsErr := awscommon.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("AWS API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *codeBuildClient) signRequest(req *http.Request, body []byte, service string) error {
	hash := sha256.Sum256(body)
	return c.signer.SignHTTP(
		req.Context(),
		*c.credentials,
		req,
		hex.EncodeToString(hash[:]),
		service,
		c.region,
		time.Now(),
	)
}

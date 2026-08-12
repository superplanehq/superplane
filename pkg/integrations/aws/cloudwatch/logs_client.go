package cloudwatch

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

const (
	logsServiceName  = "logs"
	logsTargetPrefix = "Logs_20140328."
)

// LogsClient talks to the CloudWatch Logs API (a separate service/protocol from
// the CloudWatch alarms and metrics API used by Client in client.go).
type LogsClient struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewLogsClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *LogsClient {
	return &LogsClient{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type LogGroup struct {
	Name string
	Arn  string
}

type StartQueryInput struct {
	LogGroupNames []string
	QueryString   string
	StartTime     time.Time
	EndTime       time.Time
	Limit         int
}

type ResultField struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type QueryStatistics struct {
	BytesScanned   float64 `json:"bytesScanned"`
	RecordsMatched float64 `json:"recordsMatched"`
	RecordsScanned float64 `json:"recordsScanned"`
}

type QueryResults struct {
	Status     string          `json:"status"`
	Results    [][]ResultField `json:"results"`
	Statistics QueryStatistics `json:"statistics"`
}

type LogEvent struct {
	Message   string
	Timestamp time.Time
}

// DescribeLogGroups lists log groups whose name starts with namePrefix (all log groups when empty).
func (c *LogsClient) DescribeLogGroups(namePrefix string) ([]LogGroup, error) {
	payload := map[string]any{}
	if namePrefix != "" {
		payload["logGroupNamePrefix"] = namePrefix
	}

	var response struct {
		LogGroups []struct {
			LogGroupName string `json:"logGroupName"`
			Arn          string `json:"arn"`
		} `json:"logGroups"`
	}

	if err := c.postJSON("DescribeLogGroups", payload, &response); err != nil {
		return nil, err
	}

	logGroups := make([]LogGroup, 0, len(response.LogGroups))
	for _, lg := range response.LogGroups {
		logGroups = append(logGroups, LogGroup{Name: lg.LogGroupName, Arn: lg.Arn})
	}

	return logGroups, nil
}

// StartQuery starts a CloudWatch Logs Insights query and returns its query ID.
func (c *LogsClient) StartQuery(input StartQueryInput) (string, error) {
	payload := map[string]any{
		"logGroupNames": input.LogGroupNames,
		"queryString":   input.QueryString,
		"startTime":     input.StartTime.Unix(),
		"endTime":       input.EndTime.Unix(),
	}

	if input.Limit > 0 {
		payload["limit"] = input.Limit
	}

	var response struct {
		QueryID string `json:"queryId"`
	}

	if err := c.postJSON("StartQuery", payload, &response); err != nil {
		return "", err
	}

	return response.QueryID, nil
}

// GetQueryResults retrieves the current status and rows for a Logs Insights query.
func (c *LogsClient) GetQueryResults(queryID string) (*QueryResults, error) {
	payload := map[string]any{"queryId": queryID}

	results := &QueryResults{}
	if err := c.postJSON("GetQueryResults", payload, results); err != nil {
		return nil, err
	}

	return results, nil
}

// CreateLogStream creates a log stream in a log group. Callers should treat
// common.IsAlreadyExistsErr(err) as success since streams only need to exist once.
func (c *LogsClient) CreateLogStream(logGroupName, logStreamName string) error {
	payload := map[string]any{
		"logGroupName":  logGroupName,
		"logStreamName": logStreamName,
	}

	return c.postJSON("CreateLogStream", payload, nil)
}

// PutLogEvents uploads a batch of log events to a log stream. Events must be
// in chronological order, per the CloudWatch Logs API contract.
func (c *LogsClient) PutLogEvents(logGroupName, logStreamName string, events []LogEvent) error {
	logEvents := make([]map[string]any, 0, len(events))
	for _, event := range events {
		logEvents = append(logEvents, map[string]any{
			"message":   event.Message,
			"timestamp": event.Timestamp.UnixMilli(),
		})
	}

	payload := map[string]any{
		"logGroupName":  logGroupName,
		"logStreamName": logStreamName,
		"logEvents":     logEvents,
	}

	return c.postJSON("PutLogEvents", payload, nil)
}

func (c *LogsClient) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://logs.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", logsTargetPrefix+action)

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
		return fmt.Errorf("CloudWatch Logs API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *LogsClient) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, logsServiceName, c.region, time.Now())
}

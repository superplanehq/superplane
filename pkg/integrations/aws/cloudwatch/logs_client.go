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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	logsServiceName  = "logs"
	logsTargetPrefix = "Logs_20140328."

	// DescribeLogGroups is paginated. The cap keeps the log group picker
	// responsive on accounts with very large numbers of log groups.
	maxLogGroupPages = 20
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
	logGroups := []LogGroup{}
	nextToken := ""

	for page := 0; page < maxLogGroupPages; page++ {
		payload := map[string]any{}
		if namePrefix != "" {
			payload["logGroupNamePrefix"] = namePrefix
		}
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		var response struct {
			LogGroups []struct {
				LogGroupName string `json:"logGroupName"`
				Arn          string `json:"arn"`
			} `json:"logGroups"`
			NextToken string `json:"nextToken"`
		}

		if err := c.postJSON("DescribeLogGroups", payload, &response); err != nil {
			return nil, err
		}

		for _, lg := range response.LogGroups {
			logGroups = append(logGroups, LogGroup{Name: lg.LogGroupName, Arn: lg.Arn})
		}

		nextToken = response.NextToken
		if nextToken == "" {
			break
		}
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

// rejectedLogEventsInfo mirrors PutLogEvents' response field of the same name.
// Fields are pointers because CloudWatch omits each one unless that specific
// rejection happened, and a present index of 0 is a valid, meaningful value.
type rejectedLogEventsInfo struct {
	ExpiredLogEventEndIndex  *int `json:"expiredLogEventEndIndex"`
	TooNewLogEventStartIndex *int `json:"tooNewLogEventStartIndex"`
	TooOldLogEventEndIndex   *int `json:"tooOldLogEventEndIndex"`
}

func (r *rejectedLogEventsInfo) String() string {
	reasons := []string{}
	if r.ExpiredLogEventEndIndex != nil {
		reasons = append(reasons, "events at or before the log group's retention period expired")
	}
	if r.TooNewLogEventStartIndex != nil {
		reasons = append(reasons, "events more than 2 hours in the future")
	}
	if r.TooOldLogEventEndIndex != nil {
		reasons = append(reasons, "events older than 14 days")
	}

	return strings.Join(reasons, "; ")
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

	var response struct {
		RejectedLogEventsInfo *rejectedLogEventsInfo `json:"rejectedLogEventsInfo"`
	}

	if err := c.postJSON("PutLogEvents", payload, &response); err != nil {
		return err
	}

	if response.RejectedLogEventsInfo != nil {
		return fmt.Errorf("CloudWatch Logs rejected the log event(s): %s", response.RejectedLogEventsInfo)
	}

	return nil
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

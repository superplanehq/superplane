package ecr

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
)

const (
	eventBridgeTargetPrefix = "AWSEvents."
)

type EventBridgeClient struct {
	http   core.HTTPContext
	region string
	creds  aws.Credentials
	signer *v4.Signer
}

type Target struct {
	ID  string `json:"Id"`
	Arn string `json:"Arn"`
}

type awsError struct {
	Code    string
	Message string
}

func (e *awsError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Code
}

func NewEventBridgeClient(httpCtx core.HTTPContext, creds aws.Credentials, region string) *EventBridgeClient {
	return &EventBridgeClient{
		http:   httpCtx,
		region: region,
		creds:  creds,
		signer: v4.NewSigner(),
	}
}

func (c *EventBridgeClient) CreateConnection(name, apiKeyHeader, apiKeyValue string) (string, error) {
	payload := map[string]any{
		"Name":              name,
		"AuthorizationType": "API_KEY",
		"AuthParameters": map[string]any{
			"ApiKeyAuthParameters": map[string]any{
				"ApiKeyName":  apiKeyHeader,
				"ApiKeyValue": apiKeyValue,
			},
		},
	}

	var response struct {
		ConnectionArn string `json:"ConnectionArn"`
	}

	if err := c.postJSON("CreateConnection", payload, &response); err != nil {
		return "", err
	}

	return response.ConnectionArn, nil
}

func (c *EventBridgeClient) DescribeConnection(name string) (string, error) {
	payload := map[string]any{"Name": name}
	var response struct {
		ConnectionArn string `json:"ConnectionArn"`
	}

	if err := c.postJSON("DescribeConnection", payload, &response); err != nil {
		return "", err
	}

	return response.ConnectionArn, nil
}

func (c *EventBridgeClient) DeleteConnection(name string) error {
	payload := map[string]any{"Name": name}
	return c.postJSON("DeleteConnection", payload, nil)
}

func (c *EventBridgeClient) CreateApiDestination(name, connectionArn, url string) (string, error) {
	payload := map[string]any{
		"Name":                         name,
		"ConnectionArn":                connectionArn,
		"InvocationEndpoint":           url,
		"HttpMethod":                   http.MethodPost,
		"InvocationRateLimitPerSecond": 10,
	}

	var response struct {
		ApiDestinationArn string `json:"ApiDestinationArn"`
	}

	if err := c.postJSON("CreateApiDestination", payload, &response); err != nil {
		return "", err
	}

	return response.ApiDestinationArn, nil
}

func (c *EventBridgeClient) DescribeApiDestination(name string) (string, error) {
	payload := map[string]any{"Name": name}
	var response struct {
		ApiDestinationArn string `json:"ApiDestinationArn"`
	}

	if err := c.postJSON("DescribeApiDestination", payload, &response); err != nil {
		return "", err
	}

	return response.ApiDestinationArn, nil
}

func (c *EventBridgeClient) DeleteApiDestination(name string) error {
	payload := map[string]any{"Name": name}
	return c.postJSON("DeleteApiDestination", payload, nil)
}

func (c *EventBridgeClient) PutRule(name, pattern, description string) (string, error) {
	payload := map[string]any{
		"Name":         name,
		"EventPattern": pattern,
		"State":        "ENABLED",
		"Description":  description,
	}

	var response struct {
		RuleArn string `json:"RuleArn"`
	}

	if err := c.postJSON("PutRule", payload, &response); err != nil {
		return "", err
	}

	return response.RuleArn, nil
}

func (c *EventBridgeClient) PutTargets(rule string, targets []Target) error {
	payload := map[string]any{
		"Rule":    rule,
		"Targets": targets,
	}

	return c.postJSON("PutTargets", payload, nil)
}

func (c *EventBridgeClient) RemoveTargets(rule string, targetIDs []string) error {
	payload := map[string]any{
		"Rule": rule,
		"Ids":  targetIDs,
	}

	return c.postJSON("RemoveTargets", payload, nil)
}

func (c *EventBridgeClient) DeleteRule(rule string) error {
	payload := map[string]any{
		"Name": rule,
	}

	return c.postJSON("DeleteRule", payload, nil)
}

func (c *EventBridgeClient) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://events.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", eventBridgeTargetPrefix+action)

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
		if awsErr := parseAwsError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("eventbridge request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *EventBridgeClient) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), c.creds, req, payloadHash, "events", c.region, time.Now())
}

func parseAwsError(body []byte) *awsError {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := extractString(payload["__type"])
	if code == "" {
		code = extractString(payload["code"])
	}
	if code == "" {
		code = extractString(payload["Code"])
	}

	message := extractString(payload["message"])
	if message == "" {
		message = extractString(payload["Message"])
	}

	if errPayload, ok := payload["Error"].(map[string]any); ok {
		if code == "" {
			code = extractString(errPayload["code"])
		}
		if code == "" {
			code = extractString(errPayload["Code"])
		}
		if code == "" {
			code = extractString(errPayload["type"])
		}
		if message == "" {
			message = extractString(errPayload["message"])
		}
		if message == "" {
			message = extractString(errPayload["Message"])
		}
	}

	code = normalizeAWSCode(code)
	if code == "" && message == "" {
		return nil
	}

	return &awsError{Code: code, Message: message}
}

func extractString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func normalizeAWSCode(code string) string {
	if code == "" {
		return ""
	}

	parts := strings.Split(code, "#")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return code
}

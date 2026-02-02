package eventbridge

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
	TargetPrefix = "AWSEvents."
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type Target struct {
	ID      string `json:"Id"`
	Arn     string `json:"Arn"`
	RoleArn string `json:"RoleArn,omitempty"`
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) CreateConnection(name, apiKeyHeader, apiKeyValue string, tags []common.Tag) (string, error) {
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

	if len(tags) > 0 {
		payload["Tags"] = common.TagsForAPI(tags)
	}

	var response struct {
		ConnectionArn string `json:"ConnectionArn"`
	}

	err := c.postJSON("CreateConnection", payload, &response)
	if err != nil {
		return "", err
	}

	return response.ConnectionArn, nil
}

func (c *Client) DescribeConnection(name string) (string, error) {
	payload := map[string]any{"Name": name}
	var response struct {
		ConnectionArn string `json:"ConnectionArn"`
	}

	if err := c.postJSON("DescribeConnection", payload, &response); err != nil {
		return "", err
	}

	return response.ConnectionArn, nil
}

func (c *Client) DeleteConnection(name string) error {
	payload := map[string]any{"Name": name}
	return c.postJSON("DeleteConnection", payload, nil)
}

func (c *Client) CreateApiDestination(name, connectionArn, url string, tags []common.Tag) (string, error) {
	payload := map[string]any{
		"Name":                         name,
		"ConnectionArn":                connectionArn,
		"InvocationEndpoint":           url,
		"HttpMethod":                   http.MethodPost,
		"InvocationRateLimitPerSecond": 10,
	}

	if len(tags) > 0 {
		payload["Tags"] = common.TagsForAPI(tags)
	}

	var response struct {
		ApiDestinationArn string `json:"ApiDestinationArn"`
	}

	if err := c.postJSON("CreateApiDestination", payload, &response); err != nil {
		return "", err
	}

	return response.ApiDestinationArn, nil
}

func (c *Client) DescribeApiDestination(name string) (string, error) {
	payload := map[string]any{"Name": name}
	var response struct {
		ApiDestinationArn string `json:"ApiDestinationArn"`
	}

	if err := c.postJSON("DescribeApiDestination", payload, &response); err != nil {
		return "", err
	}

	return response.ApiDestinationArn, nil
}

func (c *Client) DeleteApiDestination(name string) error {
	payload := map[string]any{"Name": name}
	return c.postJSON("DeleteApiDestination", payload, nil)
}

func (c *Client) PutRule(name, pattern string, tags []common.Tag) (string, error) {
	payload := map[string]any{
		"Name":         name,
		"EventPattern": pattern,
		"State":        "ENABLED",
	}

	if len(tags) > 0 {
		payload["Tags"] = common.TagsForAPI(tags)
	}

	var response struct {
		RuleArn string `json:"RuleArn"`
	}

	if err := c.postJSON("PutRule", payload, &response); err != nil {
		return "", err
	}

	return response.RuleArn, nil
}

func (c *Client) PutTargets(rule string, targets []Target) error {
	payload := map[string]any{
		"Rule":    rule,
		"Targets": targets,
	}

	return c.postJSON("PutTargets", payload, nil)
}

func (c *Client) RemoveTargets(rule string, targetIDs []string) error {
	payload := map[string]any{
		"Rule": rule,
		"Ids":  targetIDs,
	}

	return c.postJSON("RemoveTargets", payload, nil)
}

func (c *Client) DeleteRule(rule string) error {
	payload := map[string]any{
		"Name": rule,
	}

	return c.postJSON("DeleteRule", payload, nil)
}

func (c *Client) postJSON(action string, payload any, out any) error {
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
	req.Header.Set("X-Amz-Target", TargetPrefix+action)

	err = c.signRequest(req, body)
	if err != nil {
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
		return fmt.Errorf("EventBridge API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	err = json.Unmarshal(responseBody, out)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "events", c.region, time.Now())
}

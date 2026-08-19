package prometheus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	serviceName = "aps"
	maxResults  = "1000"
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type WorkspaceStatus struct {
	StatusCode string `json:"statusCode"`
}

type WorkspaceSummary struct {
	Alias       string            `json:"alias,omitempty"`
	Arn         string            `json:"arn"`
	CreatedAt   common.FloatTime  `json:"createdAt,omitempty"`
	KMSKeyArn   string            `json:"kmsKeyArn,omitempty"`
	Status      WorkspaceStatus   `json:"status"`
	Tags        map[string]string `json:"tags,omitempty"`
	WorkspaceID string            `json:"workspaceId"`
}

type WorkspaceDescription struct {
	WorkspaceSummary
	PrometheusEndpoint string `json:"prometheusEndpoint,omitempty"`
}

type CreateWorkspaceInput struct {
	Alias       string
	ClientToken string
	KMSKeyArn   string
	Tags        []common.Tag
}

type CreateWorkspaceResponse struct {
	Alias       string            `json:"alias,omitempty"`
	Arn         string            `json:"arn"`
	KMSKeyArn   string            `json:"kmsKeyArn,omitempty"`
	Status      WorkspaceStatus   `json:"status"`
	Tags        map[string]string `json:"tags,omitempty"`
	WorkspaceID string            `json:"workspaceId"`
}

type RuleGroupsNamespaceStatus struct {
	StatusCode   string `json:"statusCode"`
	StatusReason string `json:"statusReason,omitempty"`
}

type RuleGroupsNamespaceSummary struct {
	Arn        string                    `json:"arn"`
	Name       string                    `json:"name"`
	Status     RuleGroupsNamespaceStatus `json:"status"`
	CreatedAt  common.FloatTime          `json:"createdAt,omitempty"`
	ModifiedAt common.FloatTime          `json:"modifiedAt,omitempty"`
	Tags       map[string]string         `json:"tags,omitempty"`
}

// Data holds the decoded plain-text rules YAML, not the AWS base64 wire format.
type RuleGroupsNamespaceDescription struct {
	RuleGroupsNamespaceSummary
	Data string `json:"data"`
}

type CreateRuleGroupsNamespaceInput struct {
	WorkspaceID string
	Name        string
	Data        string
	ClientToken string
	Tags        []common.Tag
}

type CreateRuleGroupsNamespaceResponse struct {
	Name   string                    `json:"name"`
	Arn    string                    `json:"arn"`
	Status RuleGroupsNamespaceStatus `json:"status"`
	Tags   map[string]string         `json:"tags,omitempty"`
}

type QueryMetricsInput struct {
	WorkspaceID                         string
	Query                               string
	Time                                string
	Timeout                             string
	MaxSamplesProcessedWarningThreshold int
	MaxSamplesProcessedErrorThreshold   int
}

type QueryRangeMetricsInput struct {
	WorkspaceID                         string
	Query                               string
	Start                               string
	End                                 string
	Step                                string
	Timeout                             string
	MaxSamplesProcessedWarningThreshold int
	MaxSamplesProcessedErrorThreshold   int
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      strings.TrimSpace(region),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) CreateWorkspace(input CreateWorkspaceInput) (*CreateWorkspaceResponse, error) {
	payload := map[string]any{}
	if input.Alias != "" {
		payload["alias"] = input.Alias
	}
	if input.ClientToken != "" {
		payload["clientToken"] = input.ClientToken
	}
	if input.KMSKeyArn != "" {
		payload["kmsKeyArn"] = input.KMSKeyArn
	}
	if tags := tagsForAPI(input.Tags); len(tags) > 0 {
		payload["tags"] = tags
	}

	response := CreateWorkspaceResponse{}
	if err := c.requestJSON(http.MethodPost, "/workspaces", url.Values{}, payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DescribeWorkspace(workspaceID string) (*WorkspaceDescription, error) {
	var response struct {
		Workspace WorkspaceDescription `json:"workspace"`
	}

	if err := c.requestJSON(http.MethodGet, "/workspaces/"+url.PathEscape(workspaceID), url.Values{}, nil, &response); err != nil {
		return nil, err
	}

	return &response.Workspace, nil
}

func (c *Client) UpdateWorkspaceAlias(workspaceID string, alias string, clientToken string) error {
	payload := map[string]any{}
	if alias != "" {
		payload["alias"] = alias
	}
	if clientToken != "" {
		payload["clientToken"] = clientToken
	}

	return c.requestJSON(http.MethodPost, "/workspaces/"+url.PathEscape(workspaceID)+"/alias", url.Values{}, payload, nil)
}

func (c *Client) DeleteWorkspace(workspaceID string, clientToken string) error {
	query := url.Values{}
	if clientToken != "" {
		query.Set("clientToken", clientToken)
	}

	return c.requestJSON(http.MethodDelete, "/workspaces/"+url.PathEscape(workspaceID), query, nil, nil)
}

func (c *Client) ListWorkspaces(alias string) ([]WorkspaceSummary, error) {
	workspaces := []WorkspaceSummary{}
	nextToken := ""

	for {
		query := url.Values{}
		query.Set("maxResults", maxResults)
		if alias != "" {
			query.Set("alias", alias)
		}
		if nextToken != "" {
			query.Set("nextToken", nextToken)
		}

		var response struct {
			NextToken  string             `json:"nextToken"`
			Workspaces []WorkspaceSummary `json:"workspaces"`
		}
		if err := c.requestJSON(http.MethodGet, "/workspaces", query, nil, &response); err != nil {
			return nil, err
		}

		workspaces = append(workspaces, response.Workspaces...)
		if response.NextToken == "" {
			break
		}

		nextToken = response.NextToken
	}

	return workspaces, nil
}

func (c *Client) CreateRuleGroupsNamespace(input CreateRuleGroupsNamespaceInput) (*CreateRuleGroupsNamespaceResponse, error) {
	payload := map[string]any{
		"name": input.Name,
		"data": base64.StdEncoding.EncodeToString([]byte(input.Data)),
	}
	if input.ClientToken != "" {
		payload["clientToken"] = input.ClientToken
	}
	if tags := tagsForAPI(input.Tags); len(tags) > 0 {
		payload["tags"] = tags
	}

	response := CreateRuleGroupsNamespaceResponse{}
	path := "/workspaces/" + url.PathEscape(input.WorkspaceID) + "/rulegroupsnamespaces"
	if err := c.requestJSON(http.MethodPost, path, url.Values{}, payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DescribeRuleGroupsNamespace(workspaceID string, name string) (*RuleGroupsNamespaceDescription, error) {
	var response struct {
		RuleGroupsNamespace RuleGroupsNamespaceDescription `json:"ruleGroupsNamespace"`
	}

	path := "/workspaces/" + url.PathEscape(workspaceID) + "/rulegroupsnamespaces/" + url.PathEscape(name)
	if err := c.requestJSON(http.MethodGet, path, url.Values{}, nil, &response); err != nil {
		return nil, err
	}

	namespace := response.RuleGroupsNamespace
	data, err := base64.StdEncoding.DecodeString(namespace.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rule groups namespace data: %w", err)
	}
	namespace.Data = string(data)

	return &namespace, nil
}

func (c *Client) PutRuleGroupsNamespace(workspaceID string, name string, data string, clientToken string) (*CreateRuleGroupsNamespaceResponse, error) {
	payload := map[string]any{
		"data": base64.StdEncoding.EncodeToString([]byte(data)),
	}
	if clientToken != "" {
		payload["clientToken"] = clientToken
	}

	response := CreateRuleGroupsNamespaceResponse{}
	path := "/workspaces/" + url.PathEscape(workspaceID) + "/rulegroupsnamespaces/" + url.PathEscape(name)
	if err := c.requestJSON(http.MethodPut, path, url.Values{}, payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DeleteRuleGroupsNamespace(workspaceID string, name string, clientToken string) error {
	query := url.Values{}
	if clientToken != "" {
		query.Set("clientToken", clientToken)
	}

	path := "/workspaces/" + url.PathEscape(workspaceID) + "/rulegroupsnamespaces/" + url.PathEscape(name)
	return c.requestJSON(http.MethodDelete, path, query, nil, nil)
}

func (c *Client) ListRuleGroupsNamespaces(workspaceID string) ([]RuleGroupsNamespaceSummary, error) {
	namespaces := []RuleGroupsNamespaceSummary{}
	nextToken := ""
	path := "/workspaces/" + url.PathEscape(workspaceID) + "/rulegroupsnamespaces"

	for {
		query := url.Values{}
		query.Set("maxResults", maxResults)
		if nextToken != "" {
			query.Set("nextToken", nextToken)
		}

		var response struct {
			NextToken            string                       `json:"nextToken"`
			RuleGroupsNamespaces []RuleGroupsNamespaceSummary `json:"ruleGroupsNamespaces"`
		}
		if err := c.requestJSON(http.MethodGet, path, query, nil, &response); err != nil {
			return nil, err
		}

		namespaces = append(namespaces, response.RuleGroupsNamespaces...)
		if response.NextToken == "" {
			break
		}

		nextToken = response.NextToken
	}

	return namespaces, nil
}

func (c *Client) QueryMetrics(input QueryMetricsInput) (map[string]any, error) {
	query := url.Values{}
	query.Set("query", input.Query)
	if input.Timeout != "" {
		query.Set("timeout", input.Timeout)
	}
	if input.MaxSamplesProcessedWarningThreshold > 0 {
		query.Set("max_samples_processed_warning_threshold", fmt.Sprintf("%d", input.MaxSamplesProcessedWarningThreshold))
	}
	if input.MaxSamplesProcessedErrorThreshold > 0 {
		query.Set("max_samples_processed_error_threshold", fmt.Sprintf("%d", input.MaxSamplesProcessedErrorThreshold))
	}
	path := "/workspaces/" + url.PathEscape(input.WorkspaceID) + "/api/v1/query"
	if input.Time != "" {
		query.Set("time", input.Time)
	}

	response := map[string]any{}
	if err := c.requestJSONAtEndpoint(http.MethodGet, c.workspaceEndpoint(), path, query, nil, &response); err != nil {
		return nil, err
	}
	if err := validatePrometheusQueryResponse(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) QueryRangeMetrics(input QueryRangeMetricsInput) (map[string]any, error) {
	query := url.Values{}
	query.Set("query", input.Query)
	query.Set("start", input.Start)
	query.Set("end", input.End)
	query.Set("step", input.Step)
	if input.Timeout != "" {
		query.Set("timeout", input.Timeout)
	}
	if input.MaxSamplesProcessedWarningThreshold > 0 {
		query.Set("max_samples_processed_warning_threshold", fmt.Sprintf("%d", input.MaxSamplesProcessedWarningThreshold))
	}
	if input.MaxSamplesProcessedErrorThreshold > 0 {
		query.Set("max_samples_processed_error_threshold", fmt.Sprintf("%d", input.MaxSamplesProcessedErrorThreshold))
	}

	response := map[string]any{}
	path := "/workspaces/" + url.PathEscape(input.WorkspaceID) + "/api/v1/query_range"
	if err := c.requestJSONAtEndpoint(http.MethodGet, c.workspaceEndpoint(), path, query, nil, &response); err != nil {
		return nil, err
	}
	if err := validatePrometheusQueryResponse(response); err != nil {
		return nil, err
	}

	return response, nil
}

func validatePrometheusQueryResponse(response map[string]any) error {
	status, _ := response["status"].(string)
	if status == "success" {
		return nil
	}

	errorType, _ := response["errorType"].(string)
	errorMessage, _ := response["error"].(string)
	return formatPrometheusQueryError(errorType, errorMessage)
}

func formatPrometheusQueryError(errorType string, errorMessage string) error {
	if errorType == "" && errorMessage == "" {
		return fmt.Errorf("prometheus API returned non-success status")
	}

	if errorType == "" {
		return fmt.Errorf("prometheus API error: %s", errorMessage)
	}

	if errorMessage == "" {
		return fmt.Errorf("prometheus API error type: %s", errorType)
	}

	return fmt.Errorf("prometheus API error (%s): %s", errorType, errorMessage)
}

func (c *Client) requestJSON(method string, path string, query url.Values, payload any, out any) error {
	return c.requestJSONAtEndpoint(method, c.endpoint(), path, query, payload, out)
}

func (c *Client) requestJSONAtEndpoint(method string, endpoint string, path string, query url.Values, payload any, out any) error {
	body, err := requestBody(payload)
	if err != nil {
		return err
	}

	endpointURL := endpoint + path
	if len(query) > 0 {
		endpointURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, endpointURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build Prometheus request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Prometheus request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read Prometheus response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("Prometheus API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	return nil
}

func (c *Client) endpoint() string {
	return fmt.Sprintf("https://aps.%s.amazonaws.com", c.region)
}

func (c *Client) workspaceEndpoint() string {
	return fmt.Sprintf("https://aps-workspaces.%s.amazonaws.com", c.region)
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, serviceName, c.region, time.Now())
}

func requestBody(payload any) ([]byte, error) {
	if payload == nil {
		return []byte{}, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Prometheus request: %w", err)
	}

	return body, nil
}

func tagsForAPI(tags []common.Tag) map[string]string {
	normalized := common.NormalizeTags(tags)
	if len(normalized) == 0 {
		return nil
	}

	apiTags := make(map[string]string, len(normalized))
	for _, tag := range normalized {
		apiTags[tag.Key] = tag.Value
	}

	return apiTags
}

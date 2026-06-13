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
	CreatedAt  common.FloatTime          `json:"createdAt,omitempty"`
	ModifiedAt common.FloatTime          `json:"modifiedAt,omitempty"`
	Name       string                    `json:"name"`
	Status     RuleGroupsNamespaceStatus `json:"status"`
	Tags       map[string]string         `json:"tags,omitempty"`
}

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

type PutRuleGroupsNamespaceInput struct {
	WorkspaceID string
	Name        string
	Data        string
	ClientToken string
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

func (c *Client) CreateRuleGroupsNamespace(input CreateRuleGroupsNamespaceInput) (*RuleGroupsNamespaceSummary, error) {
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

	response := RuleGroupsNamespaceSummary{}
	if err := c.requestJSON(
		http.MethodPost,
		"/workspaces/"+url.PathEscape(input.WorkspaceID)+"/rulegroupsnamespaces",
		url.Values{},
		payload,
		&response,
	); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DescribeRuleGroupsNamespace(workspaceID string, name string) (*RuleGroupsNamespaceDescription, error) {
	var response struct {
		RuleGroupsNamespace RuleGroupsNamespaceDescription `json:"ruleGroupsNamespace"`
	}

	if err := c.requestJSON(
		http.MethodGet,
		ruleGroupsNamespacePath(workspaceID, name),
		url.Values{},
		nil,
		&response,
	); err != nil {
		return nil, err
	}

	return &response.RuleGroupsNamespace, nil
}

func (c *Client) PutRuleGroupsNamespace(input PutRuleGroupsNamespaceInput) (*RuleGroupsNamespaceSummary, error) {
	payload := map[string]any{
		"data": base64.StdEncoding.EncodeToString([]byte(input.Data)),
	}
	if input.ClientToken != "" {
		payload["clientToken"] = input.ClientToken
	}

	response := RuleGroupsNamespaceSummary{}
	if err := c.requestJSON(
		http.MethodPut,
		ruleGroupsNamespacePath(input.WorkspaceID, input.Name),
		url.Values{},
		payload,
		&response,
	); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DeleteRuleGroupsNamespace(workspaceID string, name string, clientToken string) error {
	query := url.Values{}
	if clientToken != "" {
		query.Set("clientToken", clientToken)
	}

	return c.requestJSON(http.MethodDelete, ruleGroupsNamespacePath(workspaceID, name), query, nil, nil)
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

func (c *Client) ListRuleGroupsNamespaces(workspaceID string, name string) ([]RuleGroupsNamespaceSummary, error) {
	namespaces := []RuleGroupsNamespaceSummary{}
	nextToken := ""

	for {
		query := url.Values{}
		query.Set("maxResults", maxResults)
		if name != "" {
			query.Set("name", name)
		}
		if nextToken != "" {
			query.Set("nextToken", nextToken)
		}

		var response struct {
			NextToken            string                       `json:"nextToken"`
			RuleGroupsNamespaces []RuleGroupsNamespaceSummary `json:"ruleGroupsNamespaces"`
		}
		if err := c.requestJSON(
			http.MethodGet,
			"/workspaces/"+url.PathEscape(workspaceID)+"/rulegroupsnamespaces",
			query,
			nil,
			&response,
		); err != nil {
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

func (c *Client) requestJSON(method string, path string, query url.Values, payload any, out any) error {
	body, err := requestBody(payload)
	if err != nil {
		return err
	}

	endpointURL := c.endpoint() + path
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

func ruleGroupsNamespacePath(workspaceID string, name string) string {
	return "/workspaces/" + url.PathEscape(workspaceID) + "/rulegroupsnamespaces/" + url.PathEscape(name)
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

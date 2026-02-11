package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultRenderBaseURL = "https://api.render.com/v1"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type workspaceWithCursor struct {
	Cursor string `json:"cursor"`
	// Render docs call this a workspace, but the API response uses the legacy "owner" key.
	Workspace Workspace `json:"owner"`
}

type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type serviceWithCursor struct {
	Cursor  string  `json:"cursor"`
	Service Service `json:"service"`
}

// ServiceDetail represents the full response from GET /services/{serviceId}.
// Note: Suspended is a string (e.g. "not_suspended", "suspended"), not a bool.
type ServiceDetail struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Suspended  string `json:"suspended"`
	AutoDeploy string `json:"autoDeploy"`
	Repo       string `json:"repo"`
	Branch     string `json:"branch"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// IsSuspended returns true if the service is in any suspended state.
func (s *ServiceDetail) IsSuspended() bool {
	return s.Suspended != "not_suspended"
}

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type envVarWithCursor struct {
	Cursor string `json:"cursor"`
	EnvVar EnvVar `json:"envVar"`
}

type rollbackRequest struct {
	DeployID string `json:"deployId"`
}

type Webhook struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"ownerId"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter"`
	Secret      string   `json:"secret"`
}

type webhookWithCursor struct {
	Cursor  string  `json:"cursor"`
	Webhook Webhook `json:"webhook"`
}

type CreateWebhookRequest struct {
	WorkspaceID string   `json:"ownerId"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter"`
}

type UpdateWebhookRequest struct {
	Name        string   `json:"name,omitempty"`
	URL         string   `json:"url,omitempty"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter,omitempty"`
}

type deployRequest struct {
	ClearCache string `json:"clearCache"`
}

type triggerDeployResponse struct {
	Deploy DeployResponse `json:"deploy"`
}

type DeployResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	FinishedAt string `json:"finishedAt"`
}

type EventResponse struct {
	ID        string               `json:"id"`
	Timestamp string               `json:"timestamp"`
	ServiceID string               `json:"serviceId"`
	Type      string               `json:"type"`
	Details   EventResponseDetails `json:"details"`
}

type EventResponseDetails interface{}

type EventDeployResponseDetails struct {
	DeployID string `json:"deployId"`
}

type EventBuildResponseDetails struct {
	BuildID string `json:"buildId"`
}

type EventUnknownResponseDetails struct{}

type EventResponseResourceDetails struct {
	ID string `json:"id"`
}

type eventResponsePayload struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	ServiceID string          `json:"serviceId"`
	Type      string          `json:"type"`
	Details   json.RawMessage `json:"details"`
}

type eventResponseDetailsEnvelope struct {
	DeployID string                        `json:"deployId"`
	BuildID  string                        `json:"buildId"`
	ID       string                        `json:"id"`
	Deploy   *EventResponseResourceDetails `json:"deploy"`
	Build    *EventResponseResourceDetails `json:"build"`
}

func (r *EventResponse) UnmarshalJSON(data []byte) error {
	payload := eventResponsePayload{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	r.ID = payload.ID
	r.Timestamp = payload.Timestamp
	r.ServiceID = payload.ServiceID
	r.Type = payload.Type
	r.Details = nil

	if len(payload.Details) == 0 {
		return nil
	}

	var detailsValue any
	if err := json.Unmarshal(payload.Details, &detailsValue); err != nil {
		return err
	}
	if detailsValue == nil {
		return nil
	}

	details := eventResponseDetailsEnvelope{}
	if err := json.Unmarshal(payload.Details, &details); err != nil {
		return err
	}

	if deployID := resolveDeployID(details, payload.Type); deployID != "" {
		r.Details = EventDeployResponseDetails{DeployID: deployID}
		return nil
	}

	if buildID := resolveBuildID(details, payload.Type); buildID != "" {
		r.Details = EventBuildResponseDetails{BuildID: buildID}
		return nil
	}

	r.Details = EventUnknownResponseDetails{}

	return nil
}

func resolveDeployID(details eventResponseDetailsEnvelope, eventType string) string {
	if details.DeployID != "" {
		return details.DeployID
	}

	if details.Deploy != nil && details.Deploy.ID != "" {
		return details.Deploy.ID
	}

	if details.ID != "" && looksLikeDeployEventType(strings.ToLower(eventType)) {
		return details.ID
	}

	return ""
}

func resolveBuildID(details eventResponseDetailsEnvelope, eventType string) string {
	if details.BuildID != "" {
		return details.BuildID
	}

	if details.Build != nil && details.Build.ID != "" {
		return details.Build.ID
	}

	if details.ID != "" && looksLikeBuildEventType(strings.ToLower(eventType)) {
		return details.ID
	}

	return ""
}

func looksLikeDeployEventType(eventType string) bool {
	return strings.Contains(eventType, "deploy")
}

func looksLikeBuildEventType(eventType string) bool {
	return strings.Contains(eventType, "build")
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	trimmedAPIKey := strings.TrimSpace(string(apiKey))
	if trimmedAPIKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	return &Client{
		APIKey:  trimmedAPIKey,
		BaseURL: defaultRenderBaseURL,
		http:    httpClient,
	}, nil
}

func (c *Client) Verify() error {
	query := url.Values{}
	query.Set("limit", "1")
	_, _, err := c.execRequestWithResponse(http.MethodGet, "/services", query, nil)
	return err
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	query := url.Values{}
	query.Set("limit", "100")

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/owners", query, nil)
	if err != nil {
		return nil, err
	}

	return parseWorkspaces(body)
}

func (c *Client) ListServices(workspaceID string) ([]Service, error) {
	query := url.Values{}
	query.Set("limit", "100")
	if strings.TrimSpace(workspaceID) != "" {
		query.Set("ownerId", strings.TrimSpace(workspaceID))
	}

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/services", query, nil)
	if err != nil {
		return nil, err
	}

	return parseServices(body)
}

func (c *Client) ListWebhooks(workspaceID string) ([]Webhook, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}

	query := url.Values{}
	query.Set("ownerId", workspaceID)
	query.Set("limit", "100")

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/webhooks", query, nil)
	if err != nil {
		return nil, err
	}

	return parseWebhooks(body)
}

func (c *Client) GetWebhook(webhookID string) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/webhooks/"+url.PathEscape(webhookID), nil, nil)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) CreateWebhook(request CreateWebhookRequest) (*Webhook, error) {
	if request.WorkspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}
	if request.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if request.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	_, body, err := c.execRequestWithResponse(http.MethodPost, "/webhooks", nil, request)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) UpdateWebhook(webhookID string, request UpdateWebhookRequest) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPatch,
		"/webhooks/"+url.PathEscape(webhookID),
		nil,
		request,
	)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) DeleteWebhook(webhookID string) error {
	if webhookID == "" {
		return fmt.Errorf("webhookID is required")
	}

	_, _, err := c.execRequestWithResponse(http.MethodDelete, "/webhooks/"+url.PathEscape(webhookID), nil, nil)
	return err
}

func (c *Client) TriggerDeploy(serviceID string, clearCache bool) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}

	clearCacheValue := "do_not_clear"
	if clearCache {
		clearCacheValue = "clear"
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/services/"+url.PathEscape(serviceID)+"/deploys",
		nil,
		deployRequest{ClearCache: clearCacheValue},
	)
	if err != nil {
		return DeployResponse{}, err
	}

	wrappedResponse := triggerDeployResponse{}
	if err := json.Unmarshal(body, &wrappedResponse); err == nil && wrappedResponse.Deploy.ID != "" {
		return wrappedResponse.Deploy, nil
	}

	deployResponse := DeployResponse{}
	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal deploy response: %w", err)
	}

	return deployResponse, nil
}

func (c *Client) GetDeploy(serviceID string, deployID string) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}
	if deployID == "" {
		return DeployResponse{}, fmt.Errorf("deployID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodGet,
		"/services/"+url.PathEscape(serviceID)+"/deploys/"+url.PathEscape(deployID),
		nil,
		nil,
	)
	if err != nil {
		return DeployResponse{}, err
	}

	deployResponse := DeployResponse{}
	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal deploy response: %w", err)
	}
	return deployResponse, nil
}

func (c *Client) GetEvent(eventID string) (EventResponse, error) {
	if eventID == "" {
		return EventResponse{}, fmt.Errorf("eventID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodGet,
		"/events/"+url.PathEscape(eventID),
		nil,
		nil,
	)
	if err != nil {
		return EventResponse{}, err
	}

	response := EventResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return EventResponse{}, fmt.Errorf("failed to unmarshal event response: %w", err)
	}

	return response, nil
}

func (c *Client) GetService(serviceID string) (*ServiceDetail, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("serviceID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodGet,
		"/services/"+url.PathEscape(serviceID),
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	service := ServiceDetail{}
	if err := json.Unmarshal(body, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service response: %w", err)
	}

	return &service, nil
}

func (c *Client) CancelDeploy(serviceID string, deployID string) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}
	if deployID == "" {
		return DeployResponse{}, fmt.Errorf("deployID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/services/"+url.PathEscape(serviceID)+"/deploys/"+url.PathEscape(deployID)+"/cancel",
		nil,
		nil,
	)
	if err != nil {
		return DeployResponse{}, err
	}

	deployResponse := DeployResponse{}
	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal deploy response: %w", err)
	}

	return deployResponse, nil
}

func (c *Client) Rollback(serviceID string, deployID string) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}
	if deployID == "" {
		return DeployResponse{}, fmt.Errorf("deployID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/services/"+url.PathEscape(serviceID)+"/rollbacks",
		nil,
		rollbackRequest{DeployID: deployID},
	)
	if err != nil {
		return DeployResponse{}, err
	}

	deployResponse := DeployResponse{}
	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal rollback response: %w", err)
	}

	return deployResponse, nil
}

func (c *Client) PurgeCache(serviceID string) error {
	if serviceID == "" {
		return fmt.Errorf("serviceID is required")
	}

	_, _, err := c.execRequestWithResponse(
		http.MethodDelete,
		"/services/"+url.PathEscape(serviceID)+"/cache",
		nil,
		nil,
	)
	return err
}

func (c *Client) UpdateEnvVars(serviceID string, envVars []EnvVar) ([]EnvVar, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("serviceID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPut,
		"/services/"+url.PathEscape(serviceID)+"/env-vars",
		nil,
		envVars,
	)
	if err != nil {
		return nil, err
	}

	return parseEnvVars(body)
}

func parseWorkspaces(body []byte) ([]Workspace, error) {
	withCursor := []workspaceWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil && len(withCursor) > 0 {
		return parseWorkspacesWithCursor(withCursor), nil
	}

	plainWorkspaces := []Workspace{}
	if err := json.Unmarshal(body, &plainWorkspaces); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspaces response: %w", err)
	}

	return plainWorkspaces, nil
}

func parseServices(body []byte) ([]Service, error) {
	withCursor := []serviceWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil && len(withCursor) > 0 {
		return parseServicesWithCursor(withCursor), nil
	}

	plainServices := []Service{}
	if err := json.Unmarshal(body, &plainServices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal services response: %w", err)
	}

	return plainServices, nil
}

func parseWebhooks(body []byte) ([]Webhook, error) {
	withCursor := []webhookWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil && len(withCursor) > 0 {
		return parseWebhooksWithCursor(withCursor), nil
	}

	plainWebhooks := []Webhook{}
	if err := json.Unmarshal(body, &plainWebhooks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhooks response: %w", err)
	}

	return plainWebhooks, nil
}

func parseWebhook(body []byte) (*Webhook, error) {
	webhook := Webhook{}
	if err := json.Unmarshal(body, &webhook); err == nil && webhook.ID != "" {
		return &webhook, nil
	}

	wrapper := struct {
		Webhook Webhook `json:"webhook"`
	}{}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook response: %w", err)
	}

	if wrapper.Webhook.ID == "" {
		return nil, fmt.Errorf("webhook id is missing in response")
	}

	return &wrapper.Webhook, nil
}

func parseWorkspacesWithCursor(withCursor []workspaceWithCursor) []Workspace {
	workspaces := make([]Workspace, 0, len(withCursor))
	for _, item := range withCursor {
		if item.Workspace.ID == "" {
			continue
		}

		workspaces = append(workspaces, item.Workspace)
	}

	return workspaces
}

func parseServicesWithCursor(withCursor []serviceWithCursor) []Service {
	services := make([]Service, 0, len(withCursor))
	for _, item := range withCursor {
		if item.Service.ID == "" {
			continue
		}

		services = append(services, item.Service)
	}

	return services
}

func parseWebhooksWithCursor(withCursor []webhookWithCursor) []Webhook {
	webhooks := make([]Webhook, 0, len(withCursor))
	for _, item := range withCursor {
		if item.Webhook.ID == "" {
			continue
		}

		webhooks = append(webhooks, item.Webhook)
	}

	return webhooks
}

func parseEnvVars(body []byte) ([]EnvVar, error) {
	withCursor := []envVarWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil && len(withCursor) > 0 {
		parsed := parseEnvVarsWithCursor(withCursor)
		if len(parsed) > 0 {
			return parsed, nil
		}
	}

	plainEnvVars := []EnvVar{}
	if err := json.Unmarshal(body, &plainEnvVars); err != nil {
		return nil, fmt.Errorf("failed to unmarshal env vars response: %w", err)
	}

	return plainEnvVars, nil
}

func parseEnvVarsWithCursor(withCursor []envVarWithCursor) []EnvVar {
	envVars := make([]EnvVar, 0, len(withCursor))
	for _, item := range withCursor {
		if item.EnvVar.Key == "" {
			continue
		}

		envVars = append(envVars, item.EnvVar)
	}

	return envVars
}

func (c *Client) execRequestWithResponse(
	method string,
	path string,
	query url.Values,
	payload any,
) (*http.Response, []byte, error) {
	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encodedBody, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(encodedBody)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}

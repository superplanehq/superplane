package octopus

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

type Client struct {
	ServerURL string
	APIKey    string
	http      core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type Space struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	IsDefault bool   `json:"IsDefault"`
}

type Project struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	Slug        string `json:"Slug"`
	Description string `json:"Description"`
}

type Environment struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	Slug        string `json:"Slug"`
	Description string `json:"Description"`
	SortOrder   int    `json:"SortOrder"`
}

type Release struct {
	ID           string `json:"Id"`
	Version      string `json:"Version"`
	ProjectID    string `json:"ProjectId"`
	ChannelID    string `json:"ChannelId"`
	Assembled    string `json:"Assembled"`
	ReleaseNotes string `json:"ReleaseNotes"`
}

type Deployment struct {
	ID                 string `json:"Id"`
	Name               string `json:"Name"`
	ProjectID          string `json:"ProjectId"`
	ReleaseID          string `json:"ReleaseId"`
	EnvironmentID      string `json:"EnvironmentId"`
	TaskID             string `json:"TaskId"`
	Created            string `json:"Created"`
	DeployedBy         string `json:"DeployedBy"`
	FailureEncountered bool   `json:"FailureEncountered"`
}

type Task struct {
	ID                   string `json:"Id"`
	Name                 string `json:"Name"`
	State                string `json:"State"`
	IsCompleted          bool   `json:"IsCompleted"`
	FinishedSuccessfully bool   `json:"FinishedSuccessfully"`
	StartTime            string `json:"StartTime"`
	CompletedTime        string `json:"CompletedTime"`
	ErrorMessage         string `json:"ErrorMessage"`
	Description          string `json:"Description"`
	Duration             string `json:"Duration"`
}

type Subscription struct {
	ID                            string                         `json:"Id"`
	Name                          string                         `json:"Name"`
	SpaceID                       string                         `json:"SpaceId"`
	Type                          string                         `json:"Type"`
	IsDisabled                    bool                           `json:"IsDisabled"`
	EventNotificationSubscription *EventNotificationSubscription `json:"EventNotificationSubscription"`
}

type EventNotificationSubscription struct {
	WebhookURI         string                   `json:"WebhookURI,omitempty"`
	WebhookHeaderKey   string                   `json:"WebhookHeaderKey,omitempty"`
	WebhookHeaderValue string                   `json:"WebhookHeaderValue,omitempty"`
	WebhookTimeout     string                   `json:"WebhookTimeout,omitempty"`
	Filter             *EventSubscriptionFilter `json:"Filter,omitempty"`
}

type EventSubscriptionFilter struct {
	EventCategories []string `json:"EventCategories,omitempty"`
	EventGroups     []string `json:"EventGroups,omitempty"`
	Projects        []string `json:"Projects,omitempty"`
	Environments    []string `json:"Environments,omitempty"`
}

type CreateSubscriptionRequest struct {
	Name                          string                         `json:"Name"`
	SpaceID                       string                         `json:"SpaceId"`
	EventNotificationSubscription *EventNotificationSubscription `json:"EventNotificationSubscription"`
}

// ResourceCollection is the generic paginated response wrapper from Octopus Deploy.
type ResourceCollection[T any] struct {
	Items        []T `json:"Items"`
	TotalResults int `json:"TotalResults"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	serverURL, err := ctx.GetConfig("serverUrl")
	if err != nil {
		return nil, err
	}

	trimmedURL := strings.TrimRight(strings.TrimSpace(string(serverURL)), "/")
	if trimmedURL == "" {
		return nil, fmt.Errorf("serverUrl is required")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	trimmedKey := strings.TrimSpace(string(apiKey))
	if trimmedKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	return &Client{
		ServerURL: trimmedURL,
		APIKey:    trimmedKey,
		http:      httpClient,
	}, nil
}

func (c *Client) ValidateCredentials() error {
	_, _, err := c.execRequest(http.MethodGet, "/api/users/me", nil, nil)
	return err
}

func (c *Client) ListSpaces() ([]Space, error) {
	_, body, err := c.execRequest(http.MethodGet, "/api/spaces/all", nil, nil)
	if err != nil {
		return nil, err
	}

	var spaces []Space
	if err := json.Unmarshal(body, &spaces); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spaces response: %w", err)
	}

	return spaces, nil
}

func (c *Client) ListProjects(spaceID string) ([]Project, error) {
	path := fmt.Sprintf("/api/%s/projects/all", url.PathEscape(spaceID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects response: %w", err)
	}

	return projects, nil
}

func (c *Client) ListEnvironments(spaceID string) ([]Environment, error) {
	path := fmt.Sprintf("/api/%s/environments/all", url.PathEscape(spaceID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var environments []Environment
	if err := json.Unmarshal(body, &environments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environments response: %w", err)
	}

	return environments, nil
}

func (c *Client) ListReleasesForProject(spaceID, projectID string) ([]Release, error) {
	path := fmt.Sprintf("/api/%s/projects/%s/releases", url.PathEscape(spaceID), url.PathEscape(projectID))
	query := url.Values{}
	query.Set("take", "30")

	_, body, err := c.execRequest(http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var collection ResourceCollection[Release]
	if err := json.Unmarshal(body, &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal releases response: %w", err)
	}

	return collection.Items, nil
}

func (c *Client) GetRelease(spaceID, releaseID string) (Release, error) {
	path := fmt.Sprintf("/api/%s/releases/%s", url.PathEscape(spaceID), url.PathEscape(releaseID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return Release{}, err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return Release{}, fmt.Errorf("failed to unmarshal release response: %w", err)
	}

	return release, nil
}

func (c *Client) CreateDeployment(spaceID, releaseID, environmentID string) (Deployment, error) {
	path := fmt.Sprintf("/api/%s/deployments", url.PathEscape(spaceID))
	payload := map[string]string{
		"ReleaseId":     releaseID,
		"EnvironmentId": environmentID,
	}

	_, body, err := c.execRequest(http.MethodPost, path, nil, payload)
	if err != nil {
		return Deployment{}, err
	}

	var deployment Deployment
	if err := json.Unmarshal(body, &deployment); err != nil {
		return Deployment{}, fmt.Errorf("failed to unmarshal deployment response: %w", err)
	}

	return deployment, nil
}

func (c *Client) GetTask(taskID string) (Task, error) {
	path := fmt.Sprintf("/api/tasks/%s", url.PathEscape(taskID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return Task{}, err
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return Task{}, fmt.Errorf("failed to unmarshal task response: %w", err)
	}

	return task, nil
}

func (c *Client) CancelTask(spaceID, taskID string) error {
	path := fmt.Sprintf("/api/%s/tasks/%s/cancel", url.PathEscape(spaceID), url.PathEscape(taskID))
	_, _, err := c.execRequest(http.MethodPost, path, nil, nil)
	return err
}

func (c *Client) GetProject(spaceID, projectID string) (Project, error) {
	path := fmt.Sprintf("/api/%s/projects/%s", url.PathEscape(spaceID), url.PathEscape(projectID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return Project{}, err
	}

	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return Project{}, fmt.Errorf("failed to unmarshal project response: %w", err)
	}

	return project, nil
}

func (c *Client) GetEnvironment(spaceID, environmentID string) (Environment, error) {
	path := fmt.Sprintf("/api/%s/environments/%s", url.PathEscape(spaceID), url.PathEscape(environmentID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return Environment{}, err
	}

	var env Environment
	if err := json.Unmarshal(body, &env); err != nil {
		return Environment{}, fmt.Errorf("failed to unmarshal environment response: %w", err)
	}

	return env, nil
}

func (c *Client) CreateSubscription(request CreateSubscriptionRequest) (Subscription, error) {
	path := fmt.Sprintf("/api/%s/subscriptions", url.PathEscape(request.SpaceID))
	_, body, err := c.execRequest(http.MethodPost, path, nil, request)
	if err != nil {
		return Subscription{}, err
	}

	var subscription Subscription
	if err := json.Unmarshal(body, &subscription); err != nil {
		return Subscription{}, fmt.Errorf("failed to unmarshal subscription response: %w", err)
	}

	return subscription, nil
}

func (c *Client) DeleteSubscription(spaceID, subscriptionID string) error {
	path := fmt.Sprintf("/api/%s/subscriptions/%s", url.PathEscape(spaceID), url.PathEscape(subscriptionID))
	_, _, err := c.execRequest(http.MethodDelete, path, nil, nil)
	return err
}

func (c *Client) ListSubscriptions(spaceID string) ([]Subscription, error) {
	path := fmt.Sprintf("/api/%s/subscriptions/all", url.PathEscape(spaceID))
	_, body, err := c.execRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var subscriptions []Subscription
	if err := json.Unmarshal(body, &subscriptions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscriptions response: %w", err)
	}

	return subscriptions, nil
}

func (c *Client) execRequest(
	method string,
	path string,
	query url.Values,
	payload any,
) (*http.Response, []byte, error) {
	endpoint := c.ServerURL + path
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

	req.Header.Set("X-Octopus-ApiKey", c.APIKey)
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

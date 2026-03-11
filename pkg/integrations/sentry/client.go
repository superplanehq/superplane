package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	BaseURL = "https://sentry.io"
)

type Client struct {
	httpCtx      core.HTTPContext
	organization string
	authToken    string
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	organization, err := integration.GetConfig("organization")
	if err != nil {
		return nil, err
	}

	authToken, err := integration.GetConfig("authToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		httpCtx:      httpCtx,
		organization: string(organization),
		authToken:    string(authToken),
	}, nil
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", BaseURL, path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Set("Content-Type", "application/json")

	return c.httpCtx.Do(req)
}

type Organization struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (c *Client) GetOrganization() (*Organization, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/", url.PathEscape(c.organization)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting organization: status=%d body=%s", resp.StatusCode, string(body))
	}

	var org Organization
	if err = json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &org, nil
}

type APIProject struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Organization struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"organization"`
}

func (c *Client) ListProjects() ([]APIProject, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/projects/", url.PathEscape(c.organization)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing projects: status=%d body=%s", resp.StatusCode, string(body))
	}

	var projects []APIProject
	if err = json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return projects, nil
}

func (c *Client) GetProject(projectSlug string) (*APIProject, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/", url.PathEscape(c.organization), url.PathEscape(projectSlug)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting project: status=%d body=%s", resp.StatusCode, string(body))
	}

	var project APIProject
	if err = json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &project, nil
}

type UpdateIssueRequest struct {
	Status       string `json:"status,omitempty"`
	AssignedTo   string `json:"assignedTo,omitempty"`
	HasSeen      *bool  `json:"hasSeen,omitempty"`
	IsBookmarked *bool  `json:"isBookmarked,omitempty"`
}

type Issue struct {
	ID           string `json:"id"`
	ShortID      string `json:"shortId"`
	Title        string `json:"title"`
	Culprit      string `json:"culprit"`
	Level        string `json:"level"`
	Status       string `json:"status"`
	IsPublic     bool   `json:"isPublic"`
	IsBookmarked bool   `json:"isBookmarked"`
	HasSeen      bool   `json:"hasSeen"`
	Project      struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"project"`
	AssignedTo *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"assignedTo,omitempty"`
	FirstSeen string `json:"firstSeen"`
	LastSeen  string `json:"lastSeen"`
	Count     string `json:"count"`
	UserCount int    `json:"userCount"`
	Permalink string `json:"permalink"`
}

func (c *Client) UpdateIssue(issueID string, request UpdateIssueRequest) (*Issue, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/0/organizations/%s/issues/%s/", url.PathEscape(c.organization), url.PathEscape(issueID)), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error updating issue: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var issue Issue
	if err = json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &issue, nil
}

type ServiceHook struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	Status      string   `json:"status"`
	Events      []string `json:"events"`
	DateCreated string   `json:"dateCreated"`
}

type CreateServiceHookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// POST /api/0/projects/{organization_slug}/{project_slug}/hooks/
func (c *Client) CreateServiceHook(projectSlug string, webhookURL string, events []string) (*ServiceHook, error) {
	request := CreateServiceHookRequest{
		URL:    webhookURL,
		Events: events,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/0/projects/%s/%s/hooks/", url.PathEscape(c.organization), url.PathEscape(projectSlug))
	resp, err := c.doRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating service hook: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var hook ServiceHook
	if err = json.NewDecoder(resp.Body).Decode(&hook); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &hook, nil
}

// GET /api/0/projects/{organization_slug}/{project_slug}/hooks/
func (c *Client) ListServiceHooks(projectSlug string) ([]ServiceHook, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/hooks/", url.PathEscape(c.organization), url.PathEscape(projectSlug)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing service hooks: status=%d body=%s", resp.StatusCode, string(body))
	}

	var hooks []ServiceHook
	if err = json.NewDecoder(resp.Body).Decode(&hooks); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return hooks, nil
}

// GET /api/0/projects/{organization_slug}/{project_slug}/hooks/{hook_id}/
func (c *Client) GetServiceHook(projectSlug, hookID string) (*ServiceHook, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/hooks/%s/", url.PathEscape(c.organization), url.PathEscape(projectSlug), url.PathEscape(hookID)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting service hook: status=%d body=%s", resp.StatusCode, string(body))
	}

	var hook ServiceHook
	if err = json.NewDecoder(resp.Body).Decode(&hook); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &hook, nil
}

// DELETE /api/0/projects/{organization_slug}/{project_slug}/hooks/{hook_id}/
func (c *Client) DeleteServiceHook(projectSlug, hookID string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/0/projects/%s/%s/hooks/%s/", url.PathEscape(c.organization), url.PathEscape(projectSlug), url.PathEscape(hookID)), nil)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting service hook: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

type ListIssuesOptions struct {
	Query   string
	Cursor  string
	PerPage int
}

// GET /api/0/projects/{organization_slug}/{project_slug}/issues/
func (c *Client) ListIssues(projectSlug string, opts *ListIssuesOptions) ([]Issue, error) {
	path := fmt.Sprintf("/api/0/projects/%s/%s/issues/", url.PathEscape(c.organization), url.PathEscape(projectSlug))

	if opts != nil {
		params := url.Values{}
		if opts.Query != "" {
			params.Set("query", opts.Query)
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.PerPage > 0 {
			params.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
		}
		if encoded := params.Encode(); encoded != "" {
			path = path + "?" + encoded
		}
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing issues: status=%d body=%s", resp.StatusCode, string(body))
	}

	var issues []Issue
	if err = json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return issues, nil
}

// GET /api/0/organizations/{organization_slug}/issues/{issue_id}/
func (c *Client) GetIssue(issueID string) (*Issue, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/issues/%s/", url.PathEscape(c.organization), url.PathEscape(issueID)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting issue: status=%d body=%s", resp.StatusCode, string(body))
	}

	var issue Issue
	if err = json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &issue, nil
}

type OrganizationMember struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	User  *struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user,omitempty"`
}

// GET /api/0/organizations/{organization_slug}/members/
func (c *Client) ListOrganizationMembers() ([]OrganizationMember, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/members/", url.PathEscape(c.organization)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing organization members: status=%d body=%s", resp.StatusCode, string(body))
	}

	var members []OrganizationMember
	if err = json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return members, nil
}

type SentryApp struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ClientSecret string `json:"clientSecret"`
	UUID         string `json:"uuid"`
}

type FormField struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Name    string `json:"name"`
	Default string `json:"default,omitempty"`
}

type AlertRuleActionSettings struct {
	Type           string      `json:"type"`
	URI            string      `json:"uri"`
	RequiredFields []FormField `json:"required_fields"`
	OptionalFields []FormField `json:"optional_fields,omitempty"`
}

type AlertRuleActionElement struct {
	Type     string                  `json:"type"`
	Title    string                  `json:"title"`
	Settings AlertRuleActionSettings `json:"settings"`
}

type SentryAppSchema struct {
	Elements []AlertRuleActionElement `json:"elements"`
}

type SentryAppCreateRequest struct {
	Name          string           `json:"name"`
	IsInternal    bool             `json:"isInternal"`
	IsAlertable   bool             `json:"isAlertable"`
	VerifyInstall bool             `json:"verifyInstall"`
	Organization  string           `json:"organization"`
	Scopes        []string         `json:"scopes"`
	WebhookURL    string           `json:"webhookUrl"`
	Events        []string         `json:"events"`
	Schema        *SentryAppSchema `json:"schema,omitempty"`
}

// POST /api/0/sentry-apps/
func (c *Client) CreateSentryApp(name, webhookURL string, events []string) (*SentryApp, error) {
	// NOTE: The URI must be a relative path (starting with /). Sentry will POST to webhookUrl + uri.
	// NOTE: required_fields cannot be empty, so we add a text field with a default value.
	schema := &SentryAppSchema{
		Elements: []AlertRuleActionElement{
			{
				Type:  "alert-rule-action",
				Title: "Send notification to SuperPlane",
				Settings: AlertRuleActionSettings{
					Type: "alert-rule-settings",
					URI:  "/",
					RequiredFields: []FormField{
						{
							Type:    "text",
							Label:   "Description",
							Name:    "description",
							Default: "issue.title",
						},
					},
				},
			},
		},
	}

	request := SentryAppCreateRequest{
		Name:          name,
		IsInternal:    true,
		IsAlertable:   true,
		VerifyInstall: false,
		Organization:  c.organization,
		Scopes:        []string{"org:read", "project:read", "event:read", "event:write", "alerts:read", "alerts:write"},
		WebhookURL:    webhookURL,
		Events:        events,
		Schema:        schema,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/0/sentry-apps/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating sentry app: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var app SentryApp
	if err := json.Unmarshal(respBody, &app); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &app, nil
}

// DELETE /api/0/sentry-apps/{slug}/
func (c *Client) DeleteSentryApp(slug string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/0/sentry-apps/%s/", url.PathEscape(slug)), nil)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting sentry app: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GET /api/0/sentry-apps/{slug}/
func (c *Client) GetSentryApp(slug string) (*SentryApp, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/sentry-apps/%s/", url.PathEscape(slug)), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("sentry app not found: %s", slug)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting sentry app: status=%d body=%s", resp.StatusCode, string(body))
	}

	var app SentryApp
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &app, nil
}

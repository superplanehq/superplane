package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

type HooksClient struct {
	*Client
}

type Hook struct {
	ID                       int    `json:"id"`
	URL                      string `json:"url"`
	ProjectID                int    `json:"project_id"`
	IssuesEvents             bool   `json:"issues_events"`
	MergeRequestsEvents      bool   `json:"merge_requests_events"`
	PushEvents               bool   `json:"push_events"`
	TagPushEvents            bool   `json:"tag_push_events"`
	NoteEvents               bool   `json:"note_events"`
	ConfidentialIssuesEvents bool   `json:"confidential_issues_events"`
	PipelineEvents           bool   `json:"pipeline_events"`
	WikiPageEvents           bool   `json:"wiki_page_events"`
	DeploymentEvents         bool   `json:"deployment_events"`
	ReleasesEvents           bool   `json:"releases_events"`
	MilestoneEvents          bool   `json:"milestone_events"`
	VulnerabilityEvents      bool   `json:"vulnerability_events"`
}

type HookEvents struct {
	IssuesEvents             bool
	MergeRequestsEvents      bool
	PushEvents               bool
	TagPushEvents            bool
	NoteEvents               bool
	ConfidentialIssuesEvents bool
	PipelineEvents           bool
	WikiPageEvents           bool
	DeploymentEvents         bool
	ReleasesEvents           bool
	MilestoneEvents          bool
	VulnerabilityEvents      bool
}

func NewHooksClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*HooksClient, error) {
	config, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("failed to get authType: %v", err)
	}
	authType := string(config)

	baseURLBytes, _ := ctx.GetConfig("baseUrl")
	baseURL := normalizeBaseURL(string(baseURLBytes))

	token, err := getAuthToken(ctx, authType)
	if err != nil {
		return nil, err
	}

	return &HooksClient{
		Client: &Client{
			baseURL:    baseURL,
			token:      token,
			authType:   authType,
			httpClient: httpClient,
		},
	}, nil
}

func (c *HooksClient) CreateHook(projectID string, webhookURL string, secret string, events HookEvents) (*Hook, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/hooks", c.baseURL, apiVersion, url.PathEscape(projectID))

	payload := map[string]any{
		"url":                        webhookURL,
		"token":                      secret,
		"issues_events":              events.IssuesEvents,
		"merge_requests_events":      events.MergeRequestsEvents,
		"push_events":                events.PushEvents,
		"tag_push_events":            events.TagPushEvents,
		"note_events":                events.NoteEvents,
		"confidential_issues_events": events.ConfidentialIssuesEvents,
		"pipeline_events":            events.PipelineEvents,
		"wiki_page_events":           events.WikiPageEvents,
		"deployment_events":          events.DeploymentEvents,
		"releases_events":            events.ReleasesEvents,
		"milestone_events":           events.MilestoneEvents,
		"vulnerability_events":       events.VulnerabilityEvents,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody := make([]byte, 1024)
		n, _ := resp.Body.Read(respBody)
		return nil, fmt.Errorf("failed to create hook for project %s: status %d, response: %s",
			projectID, resp.StatusCode, string(respBody[:n]))
	}

	var hook Hook
	if err := json.NewDecoder(resp.Body).Decode(&hook); err != nil {
		return nil, fmt.Errorf("failed to decode hook response: %v", err)
	}

	return &hook, nil
}

func (c *HooksClient) DeleteHook(projectID string, hookID int) error {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/hooks/%d", c.baseURL, apiVersion, url.PathEscape(projectID), hookID)

	req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete hook: status %d", resp.StatusCode)
	}

	return nil
}

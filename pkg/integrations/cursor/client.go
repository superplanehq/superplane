package cursor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.cursor.com"

type CloudAgentsClient struct {
	apiKey string
	http   core.HTTPContext
}

func NewCloudAgentsClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*CloudAgentsClient, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("cloudAgentsApiKey")
	if err != nil {
		return nil, err
	}

	return &CloudAgentsClient{
		apiKey: strings.TrimSpace(string(apiKey)),
		http:   httpClient,
	}, nil
}

func (c *CloudAgentsClient) Verify() error {
	_, err := c.execRequest(http.MethodGet, baseURL+"/v0/me", nil)
	return err
}

type ModelsResponse struct {
	Models []string `json:"models"`
}

func (c *CloudAgentsClient) ListModels() ([]string, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/v0/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Models, nil
}

type LaunchAgentPrompt struct {
	Text string `json:"text"`
}

type LaunchAgentSource struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref,omitempty"`
}

type LaunchAgentTarget struct {
	AutoCreatePr bool `json:"autoCreatePr,omitempty"`
	AutoBranch   bool `json:"autoBranch,omitempty"`
}

type LaunchAgentWebhook struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type LaunchAgentRequest struct {
	Prompt  LaunchAgentPrompt   `json:"prompt"`
	Model   string              `json:"model,omitempty"`
	Source  LaunchAgentSource   `json:"source"`
	Target  *LaunchAgentTarget  `json:"target,omitempty"`
	Webhook *LaunchAgentWebhook `json:"webhook,omitempty"`
}

type LaunchAgentResponse struct {
	ID        string             `json:"id"`
	Status    string             `json:"status,omitempty"`
	Source    *LaunchAgentSource `json:"source,omitempty"`
	Target    *AgentTarget       `json:"target,omitempty"`
	Summary   string             `json:"summary,omitempty"`
	CreatedAt string             `json:"createdAt,omitempty"`
}

type AgentTarget struct {
	URL        string `json:"url,omitempty"`
	BranchName string `json:"branchName,omitempty"`
	PRURL      string `json:"prUrl,omitempty"`
}

func (c *CloudAgentsClient) LaunchAgent(request LaunchAgentRequest) (*LaunchAgentResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, baseURL+"/v0/agents", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response LaunchAgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &response, nil
}

func (c *CloudAgentsClient) GetAgent(id string) (*LaunchAgentResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	responseBody, err := c.execRequest(http.MethodGet, baseURL+"/v0/agents/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	var response LaunchAgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &response, nil
}

func (c *CloudAgentsClient) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, "")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type AdminClient struct {
	apiKey string
	http   core.HTTPContext
}

func NewAdminClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*AdminClient, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("adminApiKey")
	if err != nil {
		return nil, err
	}

	return &AdminClient{
		apiKey: strings.TrimSpace(string(apiKey)),
		http:   httpClient,
	}, nil
}

func (c *AdminClient) Verify() error {
	_, err := c.execRequest(http.MethodGet, baseURL+"/teams/members", nil)
	return err
}

type DailyUsageRequest struct {
	StartDate int64 `json:"startDate"`
	EndDate   int64 `json:"endDate"`
}

// DailyUsageResponse is intentionally loose; Cursor's schema changes and we just return it as-is.
type DailyUsageResponse map[string]any

func (c *AdminClient) GetDailyUsageData(start, end time.Time) (DailyUsageResponse, error) {
	if end.Before(start) {
		return nil, fmt.Errorf("endDate must be >= startDate")
	}

	// Cursor docs allow up to 90 days; keep a conservative bound to avoid abuse.
	if end.Sub(start) > 90*24*time.Hour {
		return nil, fmt.Errorf("date range too large (max 90 days)")
	}

	reqBody := DailyUsageRequest{
		StartDate: start.UnixMilli(),
		EndDate:   end.UnixMilli(),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, baseURL+"/teams/daily-usage-data", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response DailyUsageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return response, nil
}

func (c *AdminClient) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, "")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		// Cursor Admin API returns 403 for non-enterprise / non-admin keys; keep error message readable.
		msg := strings.TrimSpace(string(responseBody))
		if msg == "" {
			msg = strconv.Itoa(res.StatusCode)
		}
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, msg)
	}

	return responseBody, nil
}

func parseRelativeDate(input string, now time.Time) (time.Time, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}

	if s == "today" || s == "now" {
		return now, nil
	}

	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err == nil && days >= 0 {
			return now.Add(-time.Duration(days) * 24 * time.Hour), nil
		}
	}

	// Try YYYY-MM-DD.
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Try RFC3339.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}

package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// MessagesUsageReportResponse is the response of GET /v1/organizations/usage_report/messages.
type MessagesUsageReportResponse struct {
	Data     []MessagesUsageBucket `json:"data"`
	HasMore  bool                  `json:"has_more"`
	NextPage string                `json:"next_page"`
}

type MessagesUsageBucket struct {
	StartingAt string                `json:"starting_at"`
	EndingAt   string                `json:"ending_at"`
	Results    []MessagesUsageResult `json:"results"`
}

type MessagesUsageResult struct {
	Model                string                     `json:"model"`
	UncachedInputTokens  int64                      `json:"uncached_input_tokens"`
	OutputTokens         int64                      `json:"output_tokens"`
	CacheReadInputTokens int64                      `json:"cache_read_input_tokens"`
	CacheCreation        MessagesUsageCacheCreation `json:"cache_creation"`
	ServerToolUse        MessagesUsageServerTool    `json:"server_tool_use"`
}

type MessagesUsageCacheCreation struct {
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
}

type MessagesUsageServerTool struct {
	WebSearchRequests int64 `json:"web_search_requests"`
}

// ClaudeCodeUsageReportResponse is the response of GET /v1/organizations/usage_report/claude_code.
type ClaudeCodeUsageReportResponse struct {
	Data     []ClaudeCodeUsageRecord `json:"data"`
	HasMore  bool                    `json:"has_more"`
	NextPage string                  `json:"next_page"`
}

type ClaudeCodeUsageRecord struct {
	Actor          ClaudeCodeActor                 `json:"actor"`
	Date           string                          `json:"date"`
	CoreMetrics    ClaudeCodeCoreMetrics           `json:"core_metrics"`
	ModelBreakdown []ClaudeCodeModelBreakdown      `json:"model_breakdown"`
	ToolActions    map[string]ClaudeCodeToolAction `json:"tool_actions"`
}

type ClaudeCodeActor struct {
	Type         string `json:"type"`
	EmailAddress string `json:"email_address"`
	APIKeyName   string `json:"api_key_name"`
}

func (a ClaudeCodeActor) Name() string {
	if a.EmailAddress != "" {
		return a.EmailAddress
	}
	return a.APIKeyName
}

type ClaudeCodeCoreMetrics struct {
	CommitsByClaudeCode      int64                 `json:"commits_by_claude_code"`
	LinesOfCode              ClaudeCodeLinesOfCode `json:"lines_of_code"`
	NumSessions              int64                 `json:"num_sessions"`
	PullRequestsByClaudeCode int64                 `json:"pull_requests_by_claude_code"`
}

type ClaudeCodeLinesOfCode struct {
	Added   int64 `json:"added"`
	Removed int64 `json:"removed"`
}

type ClaudeCodeModelBreakdown struct {
	Model         string                  `json:"model"`
	EstimatedCost ClaudeCodeEstimatedCost `json:"estimated_cost"`
	Tokens        ClaudeCodeModelTokens   `json:"tokens"`
}

type ClaudeCodeEstimatedCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ClaudeCodeModelTokens struct {
	CacheCreation int64 `json:"cache_creation"`
	CacheRead     int64 `json:"cache_read"`
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
}

type ClaudeCodeToolAction struct {
	Accepted int64 `json:"accepted"`
	Rejected int64 `json:"rejected"`
}

const usageReportPageLimit = 1000

// GetMessagesUsageReport fetches per-day, per-model token usage between startingAt (inclusive)
// and endingAt (exclusive), both RFC 3339 timestamps.
func (c *Client) GetMessagesUsageReport(startingAt, endingAt string, days int) ([]MessagesUsageBucket, error) {
	if c.AdminKey == "" {
		return nil, fmt.Errorf("admin API key is not configured")
	}

	limit := days
	if limit < 1 {
		limit = 1
	}
	if limit > 31 {
		limit = 31
	}

	var buckets []MessagesUsageBucket
	page := ""
	for {
		reqURL := c.messagesUsageReportURL(startingAt, endingAt, limit, page)
		body, err := c.execAdminRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		var response MessagesUsageReportResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal messages usage report: %w", err)
		}

		buckets = append(buckets, response.Data...)

		if !response.HasMore || response.NextPage == "" {
			break
		}
		page = response.NextPage
	}

	return buckets, nil
}

func (c *Client) messagesUsageReportURL(startingAt, endingAt string, limit int, page string) string {
	q := url.Values{}
	q.Set("starting_at", startingAt)
	q.Set("ending_at", endingAt)
	q.Set("bucket_width", "1d")
	q.Add("group_by", "model")
	q.Set("limit", fmt.Sprintf("%d", limit))
	if page != "" {
		q.Set("page", page)
	}

	return c.BaseURL + "/organizations/usage_report/messages?" + q.Encode()
}

// GetClaudeCodeUsageReport fetches per-day, per-actor Claude Code productivity metrics for every
// day between startDate and endDate (inclusive). The upstream endpoint only accepts a single day
// per call, so this issues one request (plus pagination) per day in range.
func (c *Client) GetClaudeCodeUsageReport(startDate, endDate time.Time) ([]ClaudeCodeUsageRecord, error) {
	if c.AdminKey == "" {
		return nil, fmt.Errorf("admin API key is not configured")
	}

	var records []ClaudeCodeUsageRecord
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		dayRecords, err := c.getClaudeCodeUsageReportForDay(day.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		records = append(records, dayRecords...)
	}

	return records, nil
}

func (c *Client) getClaudeCodeUsageReportForDay(date string) ([]ClaudeCodeUsageRecord, error) {
	var records []ClaudeCodeUsageRecord
	page := ""
	for {
		reqURL := c.claudeCodeUsageReportURL(date, page)
		body, err := c.execAdminRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		var response ClaudeCodeUsageReportResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal claude code usage report: %w", err)
		}

		records = append(records, response.Data...)

		if !response.HasMore || response.NextPage == "" {
			break
		}
		page = response.NextPage
	}

	return records, nil
}

func (c *Client) claudeCodeUsageReportURL(date, page string) string {
	q := url.Values{}
	q.Set("starting_at", date)
	q.Set("limit", fmt.Sprintf("%d", usageReportPageLimit))
	if page != "" {
		q.Set("page", page)
	}

	return c.BaseURL + "/organizations/usage_report/claude_code?" + q.Encode()
}

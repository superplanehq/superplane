package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxErrorSampleStackRunes  = 8000
	maxRelatedLogs            = 25
	maxRelatedLogMessageRunes = 500
	maxBreadcrumbs            = 20
	relatedLogsWindow         = 15 * time.Minute
	sampleEventSearchLimit    = 5
	logsEventSearchPath       = "/api/v2/logs/events/search"
	spansEventSearchPath      = "/api/v2/spans/events/search"
	rumEventSearchPath        = "/api/v2/rum/events/search"
)

// LoadErrorTrackingIssue reads the issue, its recent impact, a sample error
// event, and logs that share the sample trace. Impact, sample, and log lookups
// are optional: a missing permission still returns the issue and reports the
// failure to warn, which can be nil.
func (c *Client) LoadErrorTrackingIssue(issueID string, warn func(string, ...any)) (*ErrorTrackingIssue, error) {
	issue, err := c.GetErrorTrackingIssue(issueID)
	if err != nil {
		return nil, err
	}

	activity, err := c.errorTrackingActivity(issue.ID)
	if err != nil {
		warnOptionalLookup(warn, "failed to read datadog activity for issue %s: %v", issue.ID, err)
	}
	issue.TotalCount = activity.TotalCount
	issue.ImpactedUsers = activity.ImpactedUsers
	issue.ImpactedSessions = activity.ImpactedSessions
	issue.HasActivity = activity.Found

	sample, err := c.errorTrackingSample(issue)
	if err != nil && sample == nil {
		warnOptionalLookup(warn, "failed to read datadog error sample for issue %s (the application key may need apm_read, logs_read_data, or rum_apps_read): %v", issue.ID, err)
	}
	if sample == nil {
		return issue, nil
	}

	issue.Sample = sample
	issue.Stack = sample.Stack
	from, to := relatedLogsWindowFor(sample)
	sample.TraceURL = c.TraceURL(sample.TraceID)
	sample.LogsURL = c.LogsURL(relatedLogsQuery(sample), from, to)

	logs, err := c.relatedLogs(sample)
	if err != nil {
		warnOptionalLookup(warn, "failed to read datadog related logs for issue %s (the application key may need logs_read_data): %v", issue.ID, err)
		return issue, nil
	}
	issue.RelatedLogs = logs
	return issue, nil
}

func warnOptionalLookup(warn func(string, ...any), format string, args ...any) {
	if warn == nil {
		return
	}
	warn(format, args...)
}

type errorTrackingActivity struct {
	TotalCount       int64
	ImpactedUsers    int64
	ImpactedSessions int64
	Found            bool
}

func (c *Client) errorTrackingActivity(issueID string) (errorTrackingActivity, error) {
	now := time.Now()
	payload := errorTrackingSearchRequest{
		Data: errorTrackingSearchRequestData{
			Type: "search_request",
			Attributes: errorTrackingSearchRequestAttributes{
				Query:   "issue.id:" + strings.TrimSpace(issueID),
				From:    now.Add(-errorTrackingSearchWindow).UnixMilli(),
				To:      now.UnixMilli(),
				Persona: "ALL",
				States:  []string{"OPEN", "ACKNOWLEDGED", "RESOLVED", "IGNORED", "EXCLUDED"},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errorTrackingActivity{}, err
	}

	requestURL := fmt.Sprintf("%s/api/v2/error-tracking/issues/search?include=%s", c.BaseURL, url.QueryEscape("issue"))
	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return errorTrackingActivity{}, mapErrorTrackingError(err)
	}

	var response errorTrackingActivityResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return errorTrackingActivity{}, fmt.Errorf("error parsing issue activity: %v", err)
	}

	for _, result := range response.Data {
		matchedID := strings.TrimSpace(result.Relationships.Issue.Data.ID)
		if matchedID == "" {
			matchedID = strings.TrimSpace(result.ID)
		}
		if !strings.EqualFold(matchedID, issueID) {
			continue
		}
		return errorTrackingActivity{
			TotalCount:       result.Attributes.TotalCount,
			ImpactedUsers:    result.Attributes.ImpactedUsers,
			ImpactedSessions: result.Attributes.ImpactedSessions,
			Found:            true,
		}, nil
	}

	return errorTrackingActivity{}, nil
}

type errorTrackingActivityResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			TotalCount       int64 `json:"total_count"`
			ImpactedUsers    int64 `json:"impacted_users"`
			ImpactedSessions int64 `json:"impacted_sessions"`
		} `json:"attributes"`
		Relationships struct {
			Issue struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"issue"`
		} `json:"relationships"`
	} `json:"data"`
}

func (c *Client) errorTrackingSample(issue *ErrorTrackingIssue) (*ErrorSample, error) {
	if issue == nil || strings.TrimSpace(issue.ID) == "" {
		return nil, nil
	}

	query := issueSampleQuery(issue.ID)
	var lastErr error
	for _, path := range sampleEventPaths(issue.Platform) {
		sample, err := c.sampleAt(path, query)
		if err != nil {
			lastErr = err
			continue
		}
		if sample != nil {
			return sample, nil
		}
	}
	return nil, lastErr
}

// issueSampleQuery matches events of one issue. Logs keep issue.id as a
// reserved field, and other products keep it as the @issue.id attribute.
func issueSampleQuery(issueID string) string {
	id := strings.TrimSpace(issueID)
	return "issue.id:" + id + " OR @issue.id:" + id
}

func sampleEventPaths(platform string) []string {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case "BROWSER", "ANDROID", "IOS", "FLUTTER", "REACT_NATIVE", "ROKU":
		return []string{rumEventSearchPath, logsEventSearchPath, spansEventSearchPath}
	default:
		return []string{spansEventSearchPath, logsEventSearchPath, rumEventSearchPath}
	}
}

func sampleSourceForPath(path string) string {
	switch path {
	case rumEventSearchPath:
		return "rum"
	case logsEventSearchPath:
		return "log"
	default:
		return "span"
	}
}

func (c *Client) sampleAt(path, query string) (*ErrorSample, error) {
	decoded, err := c.searchEvents(path, query, "now-30d", "now", "-timestamp", sampleEventSearchLimit)
	if err != nil {
		return nil, err
	}
	return parseErrorSample(decoded, sampleSourceForPath(path)), nil
}

func (c *Client) relatedLogs(sample *ErrorSample) ([]LogLine, error) {
	query := relatedLogsQuery(sample)
	if query == "" {
		return nil, nil
	}

	from, to := relatedLogsWindowFor(sample)
	decoded, err := c.searchEvents(logsEventSearchPath, query, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano), "timestamp", maxRelatedLogs)
	if err != nil {
		return nil, err
	}
	return parseRelatedLogs(decoded), nil
}

func relatedLogsQuery(sample *ErrorSample) string {
	if sample == nil {
		return ""
	}
	if ids := telemetryIDs(sample.TraceID); len(ids) > 0 {
		return "trace_id:(" + strings.Join(ids, " OR ") + ")"
	}
	if ids := telemetryIDs(sample.SpanID); len(ids) > 0 {
		return "span_id:(" + strings.Join(ids, " OR ") + ")"
	}
	return ""
}

func relatedLogsWindowFor(sample *ErrorSample) (time.Time, time.Time) {
	if sample == nil || sample.Timestamp.IsZero() {
		now := time.Now().UTC()
		return now.Add(-errorTrackingSearchWindow), now
	}
	stamp := sample.Timestamp.UTC()
	return stamp.Add(-relatedLogsWindow), stamp.Add(relatedLogsWindow)
}

func (c *Client) searchEvents(path, query, from, to, sort string, limit int) (any, error) {
	body, err := json.Marshal(eventSearchPayload(path, query, from, to, sort, limit))
	if err != nil {
		return nil, err
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// eventSearchPayload builds the search body. The spans API wraps the request
// in data.attributes; the logs and RUM APIs read it at the top level and
// ignore unknown fields.
func eventSearchPayload(path, query, from, to, sort string, limit int) map[string]any {
	request := map[string]any{
		"filter": map[string]any{
			"query": query,
			"from":  from,
			"to":    to,
		},
		"sort": sort,
		"page": map[string]any{"limit": limit},
	}
	if path != spansEventSearchPath {
		return request
	}
	return map[string]any{
		"data": map[string]any{
			"type":       "search_request",
			"attributes": request,
		},
	}
}

func applyErrorTrackingIncluded(issue *ErrorTrackingIssue, included []errorTrackingIncludedResource) {
	if issue == nil {
		return
	}

	for _, item := range included {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "user":
			if issue.Assignee == "" {
				issue.Assignee = includedUserLabel(item.Attributes)
			}
		case "team":
			if name := includedTeamLabel(item.Attributes); name != "" {
				issue.Teams = append(issue.Teams, name)
			}
		case "case":
			applyIncludedCase(issue, item.Attributes)
		}
	}
}

func includedUserLabel(raw json.RawMessage) string {
	var user struct {
		Name   string `json:"name"`
		Handle string `json:"handle"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(raw, &user); err != nil {
		return ""
	}
	return firstNonEmpty(user.Name, user.Handle, user.Email)
}

func includedTeamLabel(raw json.RawMessage) string {
	var team struct {
		Name   string `json:"name"`
		Handle string `json:"handle"`
	}
	if err := json.Unmarshal(raw, &team); err != nil {
		return ""
	}
	return firstNonEmpty(team.Name, team.Handle)
}

func applyIncludedCase(issue *ErrorTrackingIssue, raw json.RawMessage) {
	var caseResource struct {
		Key         string      `json:"key"`
		Title       string      `json:"title"`
		JiraIssue   linkedIssue `json:"jira_issue"`
		LinearIssue linkedIssue `json:"linear_issue"`
	}
	if err := json.Unmarshal(raw, &caseResource); err != nil {
		return
	}
	issue.CaseKey = strings.TrimSpace(caseResource.Key)
	issue.CaseTitle = strings.TrimSpace(caseResource.Title)
	issue.CaseURL = firstNonEmpty(caseResource.JiraIssue.Result.IssueURL, caseResource.LinearIssue.Result.IssueURL)
}

type linkedIssue struct {
	Result struct {
		IssueURL string `json:"issue_url"`
	} `json:"result"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

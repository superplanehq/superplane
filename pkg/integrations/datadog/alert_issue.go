package datadog

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	maxIssueIDAttempts = 4
	maxAlertTitleRunes = 256
)

var (
	issueIDMarker   = regexp.MustCompile(`(?i)(?:@?issue\.id|issueId)[=:]([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	issuePathMarker = regexp.MustCompile(`(?i)/error-tracking/issue/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	uuidMarker      = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

func enrichErrorTrackingAlert(ctx core.IntegrationMessageContext, payload ErrorTrackingAlertPayload) ErrorTrackingAlertPayload {
	alert := alertDetailsFromPayload(payload)
	issue := loadAlertIssue(ctx, alert)
	if issue != nil {
		if title := limitRunes(issue.IssueTitle(), maxAlertTitleRunes); title != "" && title != "Datadog error" {
			payload.Title = title
		}
	}

	described := ErrorTrackingIssue{}
	if issue != nil {
		described = *issue
	}
	text := DescribeErrorTrackingIssue(described, alert)
	payload.Description = text
	payload.Body = text
	payload.Environment = strings.ToLower(strings.TrimSpace(environmentFromAlert(payload)))
	if payload.Environment == "" && issue != nil && issue.Sample != nil {
		payload.Environment = strings.ToLower(strings.TrimSpace(issue.Sample.Env))
	}
	return payload
}

func alertDetailsFromPayload(payload ErrorTrackingAlertPayload) AlertDetails {
	return AlertDetails{
		Title:        payload.Title,
		Body:         payload.Body,
		EventMessage: payload.EventMessage,
		Link:         payload.Link,
		Tags:         payload.Tags,
		AlertScope:   payload.AlertScope,
		AlertQuery:   payload.AlertQuery,
		AggregKey:    payload.AggregKey,
	}
}

func loadAlertIssue(ctx core.IntegrationMessageContext, alert AlertDetails) *ErrorTrackingIssue {
	if ctx.HTTP == nil || ctx.Integration == nil {
		return nil
	}

	candidates := issueIDCandidates(alert)
	if len(candidates) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		warnAlert(ctx, "failed to create datadog client for error tracking alert: %v", err)
		return nil
	}

	for _, id := range candidates {
		issue, err := client.LoadErrorTrackingIssue(id, func(format string, args ...any) {
			warnAlert(ctx, format, args...)
		})
		if err == nil && issue != nil && strings.TrimSpace(issue.ID) != "" {
			return issue
		}
		if errors.Is(err, ErrErrorTrackingForbidden) {
			warnAlert(ctx, "datadog refused error tracking read for issue %s: %v", id, err)
			return nil
		}
		if err != nil && !isMissingErrorTrackingIssue(err) {
			warnAlert(ctx, "failed to load datadog error tracking issue %s: %v", id, err)
		}
	}
	return nil
}

func isMissingErrorTrackingIssue(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusBadRequest
}

func warnAlert(ctx core.IntegrationMessageContext, format string, args ...any) {
	if ctx.Logger == nil {
		return
	}
	ctx.Logger.Warnf(format, args...)
}

func issueIDCandidates(alert AlertDetails) []string {
	texts := []string{
		alert.Link,
		alert.AlertScope,
		alert.Tags,
		alert.AggregKey,
		alert.AlertQuery,
		alert.EventMessage,
		alert.Body,
		alert.Title,
	}

	var marked, paths, versionOne, others []string
	seen := map[string]bool{}
	add := func(bucket *[]string, id string) {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		*bucket = append(*bucket, id)
	}

	for _, text := range texts {
		text = unescapeAlertText(text)
		for _, match := range issueIDMarker.FindAllStringSubmatch(text, -1) {
			add(&marked, match[1])
		}
		for _, match := range issuePathMarker.FindAllStringSubmatch(text, -1) {
			add(&paths, match[1])
		}
		for _, id := range uuidMarker.FindAllString(text, -1) {
			if isDatadogStyleIssueID(id) {
				add(&versionOne, id)
				continue
			}
			add(&others, id)
		}
	}

	combined := append(append(append([]string{}, marked...), paths...), versionOne...)
	combined = append(combined, others...)
	if len(combined) > maxIssueIDAttempts {
		return combined[:maxIssueIDAttempts]
	}
	return combined
}

func unescapeAlertText(text string) string {
	decoded, err := url.QueryUnescape(text)
	if err != nil {
		return text
	}
	return decoded
}

func isDatadogStyleIssueID(id string) bool {
	parts := strings.Split(id, "-")
	return len(parts) == 5 && strings.HasPrefix(strings.ToLower(parts[2]), "1")
}

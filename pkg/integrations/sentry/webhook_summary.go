package sentry

import (
	"encoding/json"
	"strings"
)

// WebhookSummary is the non-secret part of a Sentry webhook. It omits the
// payload, signature, tokens, and grant code.
type WebhookSummary struct {
	Resource         string
	Action           string
	InstallationUUID string
	OrganizationSlug string
	ProjectSlug      string
	IssueID          string
	IssueShortID     string
}

func SummarizeWebhook(resource string, body []byte) WebhookSummary {
	summary := WebhookSummary{Resource: strings.TrimSpace(resource)}

	var payload struct {
		Action       string `json:"action"`
		Installation struct {
			UUID string `json:"uuid"`
		} `json:"installation"`
		Data struct {
			Issue struct {
				ID      jsonString `json:"id"`
				ShortID string     `json:"shortId"`
				Project struct {
					Slug string `json:"slug"`
				} `json:"project"`
			} `json:"issue"`
			Installation struct {
				UUID         string `json:"uuid"`
				Organization struct {
					Slug string `json:"slug"`
				} `json:"organization"`
			} `json:"installation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return summary
	}

	summary.Action = strings.TrimSpace(payload.Action)
	summary.InstallationUUID = firstNonEmpty(
		strings.TrimSpace(payload.Installation.UUID),
		strings.TrimSpace(payload.Data.Installation.UUID),
	)
	summary.OrganizationSlug = strings.TrimSpace(payload.Data.Installation.Organization.Slug)
	summary.ProjectSlug = strings.TrimSpace(payload.Data.Issue.Project.Slug)
	summary.IssueID = strings.TrimSpace(string(payload.Data.Issue.ID))
	summary.IssueShortID = strings.TrimSpace(payload.Data.Issue.ShortID)
	return summary
}

type jsonString string

func (value *jsonString) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		*value = jsonString(text)
		return nil
	}
	*value = jsonString(strings.Trim(string(raw), `"`))
	return nil
}

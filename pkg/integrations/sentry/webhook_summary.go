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

	var envelope struct {
		Action       string          `json:"action"`
		Installation json.RawMessage `json:"installation"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return summary
	}

	summary.Action = strings.TrimSpace(envelope.Action)
	summary.InstallationUUID = installationUUID(envelope.Installation)

	var data struct {
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
	}
	if !jsonObject(envelope.Data) || json.Unmarshal(envelope.Data, &data) != nil {
		return summary
	}

	summary.InstallationUUID = firstNonEmpty(summary.InstallationUUID, strings.TrimSpace(data.Installation.UUID))
	summary.OrganizationSlug = strings.TrimSpace(data.Installation.Organization.Slug)
	summary.ProjectSlug = strings.TrimSpace(data.Issue.Project.Slug)
	summary.IssueID = strings.TrimSpace(string(data.Issue.ID))
	summary.IssueShortID = strings.TrimSpace(data.Issue.ShortID)
	return summary
}

func installationUUID(raw json.RawMessage) string {
	if !jsonObject(raw) {
		return ""
	}
	var installation struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(raw, &installation); err != nil {
		return ""
	}
	return strings.TrimSpace(installation.UUID)
}

func jsonObject(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return strings.HasPrefix(trimmed, "{")
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

package newrelic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

var validStatuses = []string{"ACTIVATED", "ACKNOWLEDGED", "CLOSED"}
var validPriorities = []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}

type OnIssue struct{}

type OnIssueConfiguration struct {
	Statuses   []string `json:"statuses" mapstructure:"statuses"`
	Priorities []string `json:"priorities" mapstructure:"priorities"`
}

type OnIssueMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

type NewRelicIssuePayload struct {
	IssueID       string   `json:"issueId"`
	IssueURL      string   `json:"issueUrl"`
	Title         string   `json:"title"`
	Priority      string   `json:"priority"`
	State         string   `json:"state"`
	PolicyName    string   `json:"policyName"`
	ConditionName string   `json:"conditionName"`
	AccountID     int      `json:"accountId"`
	CreatedAt     int64    `json:"createdAt"`
	UpdatedAt     int64    `json:"updatedAt"`
	Sources       []string `json:"sources"`
}

func (t *OnIssue) Name() string {
	return "newrelic.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Trigger when a New Relic alert issue occurs"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when a New Relic alert issue is received via webhook.

## What this trigger does

- Receives New Relic webhook payloads for alert issues
- Filters by issue state (ACTIVATED, ACKNOWLEDGED, CLOSED)
- Optionally filters by priority (CRITICAL, HIGH, MEDIUM, LOW)
- Emits matching issues as ` + "`newrelic.issue`" + ` events

## Configuration

- **Statuses**: Required list of issue states to listen for
- **Priorities**: Optional priority filter

## Webhook Setup

The webhook URL is generated when the integration is saved. See the integration setup instructions for how to configure it in New Relic.
`
}

func (t *OnIssue) Icon() string {
	return "chart-bar"
}

func (t *OnIssue) Color() string {
	return "gray"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"ACTIVATED"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Activated", Value: "ACTIVATED"},
						{Label: "Acknowledged", Value: "ACKNOWLEDGED"},
						{Label: "Closed", Value: "CLOSED"},
					},
				},
			},
			Description: "Only emit issues with these states",
		},
		{
			Name:     "priorities",
			Label:    "Priorities",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Critical", Value: "CRITICAL"},
						{Label: "High", Value: "HIGH"},
						{Label: "Medium", Value: "MEDIUM"},
						{Label: "Low", Value: "LOW"},
					},
				},
			},
			Description: "Optional priority filter",
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	metadata := OnIssueMetadata{}
	if ctx.Metadata != nil && ctx.Metadata.Get() != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	if _, err := parseAndValidateOnIssueConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if err := ctx.Integration.RequestWebhook(struct{}{}); err != nil {
		return err
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook URL: %w", err)
	}

	metadata.WebhookURL = webhookURL

	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(metadata)
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if statusCode, err := validateWebhookAuth(ctx); err != nil {
		return statusCode, err
	}

	config, err := parseAndValidateOnIssueConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var payload NewRelicIssuePayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err)
	}

	filteredStatuses := filterEmptyStrings(config.Statuses)
	if !containsIgnoreCase(filteredStatuses, payload.State) {
		return http.StatusOK, nil
	}

	filteredPriorities := filterEmptyStrings(config.Priorities)
	if len(filteredPriorities) > 0 && !containsIgnoreCase(filteredPriorities, payload.Priority) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit(NewRelicIssuePayloadType, issueToMap(payload)); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit issue event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func parseAndValidateOnIssueConfiguration(configuration any) (OnIssueConfiguration, error) {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnIssueConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = sanitizeOnIssueConfiguration(config)
	if err := validateOnIssueConfiguration(config); err != nil {
		return OnIssueConfiguration{}, err
	}

	return config, nil
}

func sanitizeOnIssueConfiguration(config OnIssueConfiguration) OnIssueConfiguration {
	for i := range config.Statuses {
		config.Statuses[i] = strings.ToUpper(strings.TrimSpace(config.Statuses[i]))
	}

	for i := range config.Priorities {
		config.Priorities[i] = strings.ToUpper(strings.TrimSpace(config.Priorities[i]))
	}

	return config
}

func validateOnIssueConfiguration(config OnIssueConfiguration) error {
	statuses := filterEmptyStrings(config.Statuses)
	if len(statuses) == 0 {
		return fmt.Errorf("at least one status must be selected")
	}

	for _, status := range statuses {
		if !slices.Contains(validStatuses, status) {
			return fmt.Errorf("invalid status %q", status)
		}
	}

	for _, priority := range filterEmptyStrings(config.Priorities) {
		if !slices.Contains(validPriorities, priority) {
			return fmt.Errorf("invalid priority %q", priority)
		}
	}

	return nil
}

func filterEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func containsIgnoreCase(allowed []string, value string) bool {
	for _, v := range allowed {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}

func validateWebhookAuth(ctx core.WebhookRequestContext) (int, error) {
	if ctx.Integration == nil {
		return http.StatusOK, nil
	}

	secret, err := optionalIntegrationConfig(ctx.Integration, "webhookSecret")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to read webhook auth configuration: %v", err)
	}

	if secret == "" {
		return http.StatusOK, nil
	}

	authorization := ctx.Headers.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return http.StatusForbidden, fmt.Errorf("missing bearer authorization")
	}

	token := authorization[len("Bearer "):]
	if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid bearer token")
	}

	return http.StatusOK, nil
}

func optionalIntegrationConfig(integration core.IntegrationContext, name string) (string, error) {
	if integration == nil {
		return "", nil
	}

	value, err := integration.GetConfig(name)
	if err != nil {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "not found") {
			return "", nil
		}

		// Optional fields can be stored as null in configuration.
		if strings.Contains(errText, "not a string") {
			return "", nil
		}

		return "", err
	}

	return string(value), nil
}

func issueToMap(payload NewRelicIssuePayload) map[string]any {
	return map[string]any{
		"issueId":       payload.IssueID,
		"issueUrl":      payload.IssueURL,
		"title":         payload.Title,
		"priority":      payload.Priority,
		"state":         payload.State,
		"policyName":    payload.PolicyName,
		"conditionName": payload.ConditionName,
		"accountId":     payload.AccountID,
		"createdAt":     payload.CreatedAt,
		"updatedAt":     payload.UpdatedAt,
		"sources":       payload.Sources,
	}
}

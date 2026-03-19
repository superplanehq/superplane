package elastic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlertFires struct{}

type OnAlertFiresConfiguration struct {
	Rule       string                    `json:"rule" mapstructure:"rule"`
	Spaces     []string                  `json:"spaces" mapstructure:"spaces"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
	Severities []string                  `json:"severities" mapstructure:"severities"`
	Statuses   []string                  `json:"statuses" mapstructure:"statuses"`
}

type OnAlertFiresMetadata struct {
	RuleName string `json:"ruleName" mapstructure:"ruleName"`
}

const kibanaAlertWebhookActionBody = `{
  "ruleId": "{{rule.id}}",
  "ruleName": "{{rule.name}}",
  "spaceId": "{{rule.spaceId}}",
  "tags": {{rule.tags}},
  "severity": "{{context.severity}}",
  "status": "{{rule.status}}"
}`

func (t *OnAlertFires) Name() string  { return "elastic.onAlertFires" }
func (t *OnAlertFires) Label() string { return "When Alert Fires" }
func (t *OnAlertFires) Description() string {
	return "Trigger a workflow when a Kibana alert rule fires"
}
func (t *OnAlertFires) Icon() string  { return "bell" }
func (t *OnAlertFires) Color() string { return "gray" }

func (t *OnAlertFires) Documentation() string {
	return `The When Alert Fires trigger starts a workflow execution when a Kibana alert rule fires via a webhook connector.

## Setup

1. Save this trigger — SuperPlane automatically creates a signed Kibana Webhook connector.
2. Select the Kibana alert rule in SuperPlane. SuperPlane attaches its webhook connector to that rule automatically.

### Recommended Kibana action body

SuperPlane configures the rule action body with these fields:

` + "```" + `json
` + kibanaAlertWebhookActionBody + `
` + "```" + `

Kibana substitutes ` + "`{{rule.id}}`" + ` and ` + "`{{rule.name}}`" + ` at delivery time. Fields omitted from the body will not be filterable in SuperPlane.

## Filtering

The **Rule** field is required. Additional filters are optional refinements. When multiple values are provided in a list, any value matching is sufficient (OR). All active filter types must match simultaneously (AND across types).

The selected **Rule** is the primary filter. Additional filters like space, tags, severity, and status are optional refinements.

## Webhook Verification

SuperPlane generates a random signing secret and configures the Kibana connector to include it on every request. Requests without the correct secret are rejected automatically.

## Event Data

Each received alert emits the parsed JSON body sent by Kibana directly as the event data. Use the workflow event timestamp to know when SuperPlane received it.`
}

func (t *OnAlertFires) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "rule",
			Label:       "Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Listen only for alerts from this Kibana alert rule.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeKibanaRule,
				},
			},
		},
		{
			Name:        "spaces",
			Label:       "Spaces",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only fire for alerts originating from these Kibana spaces. Leave empty to accept all spaces.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeKibanaSpace,
					Multi: true,
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for alerts that include at least one tag matching any of these predicates.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "severities",
			Label:       "Severities",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only fire for alerts with these severity levels. Leave empty to accept all severities.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeKibanaAlertSeverity,
					Multi: true,
				},
			},
		},
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only fire for alerts with these statuses. Leave empty to accept all statuses.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeKibanaAlertStatus,
					Multi: true,
				},
			},
		},
	}
}

func (t *OnAlertFires) Setup(ctx core.TriggerContext) error {
	var config OnAlertFiresConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Rule) == "" {
		return fmt.Errorf("rule is required")
	}

	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	rule, err := client.GetKibanaRule(config.Rule)
	if err != nil {
		return fmt.Errorf("failed to get Kibana rule %s: %w", config.Rule, err)
	}

	if err := ctx.Metadata.Set(OnAlertFiresMetadata{RuleName: rule.Name}); err != nil {
		return fmt.Errorf("failed to store rule metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(map[string]any{
		"kibanaUrl": string(kibanaURL),
		"ruleId":    config.Rule,
	})
}

func (t *OnAlertFires) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertFires) HandleAction(_ core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlertFires) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error retrieving webhook secret: %v", err)
	}

	headerVal := ctx.Headers.Get(SigningHeaderName)
	if headerVal == "" {
		return http.StatusForbidden, nil, fmt.Errorf("missing required header %q", SigningHeaderName)
	}
	if len(headerVal) != len(secret) || subtle.ConstantTimeCompare([]byte(headerVal), secret) != 1 {
		return http.StatusForbidden, nil, fmt.Errorf("invalid value for header %q", SigningHeaderName)
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	var config OnAlertFiresConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if !matchesFilters(payload, config) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("elastic.alert", payload); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnAlertFires) Cleanup(_ core.TriggerContext) error {
	return nil
}

// matchesFilters returns true if the alert payload satisfies all configured filters.
// An empty/nil filter is treated as a pass-through (no restriction).
func matchesFilters(payload map[string]any, config OnAlertFiresConfiguration) bool {
	if config.Rule != "" {
		ruleID := extractString(payload, "ruleId")
		if ruleID != "" && !strings.EqualFold(strings.TrimSpace(config.Rule), strings.TrimSpace(ruleID)) {
			return false
		}
	}

	if len(config.Spaces) > 0 {
		if !containsIgnoreCase(config.Spaces, extractString(payload, "spaceId")) {
			return false
		}
	}

	if len(config.Tags) > 0 {
		matched := false
		for _, tag := range extractStringSlice(payload, "tags") {
			if configuration.MatchesAnyPredicate(config.Tags, tag) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(config.Severities) > 0 {
		if !containsIgnoreCase(config.Severities, extractString(payload, "severity")) {
			return false
		}
	}

	if len(config.Statuses) > 0 {
		if !containsIgnoreCase(config.Statuses, extractString(payload, "status")) {
			return false
		}
	}

	return true
}

// extractString returns the first non-empty string value found for the given keys.
func extractString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// extractStringSlice returns a string slice from the payload for the given key.
func extractStringSlice(payload map[string]any, key string) []string {
	v, ok := payload[key]
	if !ok {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// containsIgnoreCase reports whether value is in list (case-insensitive).
func containsIgnoreCase(list []string, value string) bool {
	for _, item := range list {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

// matchesAnyString reports whether any candidate appears in list (case-insensitive).
func matchesAnyString(list []string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if containsIgnoreCase(list, candidate) {
			return true
		}
	}

	return false
}

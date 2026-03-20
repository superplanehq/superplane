package elastic

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

type OnAlertFires struct{}

type OnAlertFiresConfiguration struct {
	Rule       string                    `json:"rule" mapstructure:"rule"`
	Spaces     []string                  `json:"spaces" mapstructure:"spaces"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
	Severities []string                  `json:"severities" mapstructure:"severities"`
	Statuses   []string                  `json:"statuses" mapstructure:"statuses"`
}

type OnAlertFiresMetadata struct {
	RuleID   string   `json:"ruleId" mapstructure:"ruleId"`
	RuleName string   `json:"ruleName" mapstructure:"ruleName"`
	Spaces   []string `json:"spaces" mapstructure:"spaces"`
}

const kibanaAlertWebhookActionBody = `{
  "eventType": "alert_fired",
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

## Shared Connector

SuperPlane creates **one Kibana Webhook connector per integration**, shared across all triggers that use the same Kibana instance. Each incoming request is routed to the correct trigger using the ` + "`eventType`" + ` field in the request body — this trigger only processes requests where ` + "`eventType`" + ` is ` + "`\"alert_fired\"`" + `. Requests intended for other trigger types (e.g. ` + "`\"document_indexed\"`" + `) are silently ignored.

## Setup

1. Select the Kibana alert rule in SuperPlane and save the trigger.
2. SuperPlane automatically creates or reuses the shared Kibana Webhook connector and attaches it to the selected rule if it is missing.

### Kibana action body

SuperPlane configures the rule action body with these fields:

` + "```" + `json
` + kibanaAlertWebhookActionBody + `
` + "```" + `

The ` + "`eventType`" + ` field is required for routing. Kibana substitutes ` + "`{{rule.id}}`" + ` and ` + "`{{rule.name}}`" + ` at delivery time. Fields omitted from the body will not be filterable in SuperPlane.

## Filtering

Select at least one **Rule**. Additional filter fields are optional. When multiple values are provided in a list, any value matching is sufficient (OR). All active filter types must match simultaneously (AND across types).

**Rule ID** is the most reliable selector because rule names are user-editable. Use it when you need precise per-rule routing.

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
			Description: "Select the Kibana alert rule to listen to. SuperPlane ensures its connector is attached to the selected rule.",
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
	config.Rule = strings.TrimSpace(config.Rule)
	config.Spaces = normalizeStringList(config.Spaces)
	config.Severities = normalizeStringList(config.Severities)
	config.Statuses = normalizeStringList(config.Statuses)

	if config.Rule == "" {
		return fmt.Errorf("rule is required")
	}
	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}
	resolvedSpaces := make([]string, 0, len(config.Spaces))
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Elastic client: %w", err)
	}

	rule, err := client.GetKibanaRule(config.Rule)
	if err != nil {
		return fmt.Errorf("failed to get Kibana rule %s: %w", config.Rule, err)
	}

	resolvedSpaces = append(resolvedSpaces, config.Spaces...)
	if hasStaticSelection(config.Spaces) {
		spaces, err := client.ListKibanaSpaces()
		if err != nil {
			return fmt.Errorf("failed to list Kibana spaces: %w", err)
		}

		for idx, selectedSpace := range config.Spaces {
			if isTemplateExpression(selectedSpace) {
				continue
			}

			spaceName, ok := findSpaceName(spaces, selectedSpace)
			if !ok {
				return fmt.Errorf("selected space %q was not found in Kibana", selectedSpace)
			}
			resolvedSpaces[idx] = spaceName
		}
	}

	if err := validateStaticSelections(config.Severities, allowedAlertSeverities()); err != nil {
		return fmt.Errorf("invalid severity selection: %w", err)
	}
	if err := validateStaticSelections(config.Statuses, allowedAlertStatuses()); err != nil {
		return fmt.Errorf("invalid status selection: %w", err)
	}

	ruleName := strings.TrimSpace(rule.Name)
	if ruleName == "" {
		ruleName = config.Rule
	}

	if err := ctx.Metadata.Set(OnAlertFiresMetadata{
		RuleID:   config.Rule,
		RuleName: ruleName,
		Spaces:   resolvedSpaces,
	}); err != nil {
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

	if eventType := extractString(payload, "eventType"); eventType != "alert_fired" {
		return http.StatusOK, nil, nil
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
	return slices.ContainsFunc(list, func(item string) bool {
		return strings.EqualFold(item, value)
	})
}

// matchesAnyString reports whether any candidate appears in list (case-insensitive).
func matchesAnyString(list []string, candidates ...string) bool {
	return slices.ContainsFunc(candidates, func(candidate string) bool {
		if candidate == "" {
			return false
		}
		return containsIgnoreCase(list, candidate)
	})
}

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func hasStaticSelection(values []string) bool {
	return slices.ContainsFunc(values, func(value string) bool {
		return !isTemplateExpression(value)
	})
}

func isTemplateExpression(value string) bool {
	return strings.Contains(value, "{{") && strings.Contains(value, "}}")
}

func findSpaceName(spaces []KibanaSpace, selected string) (string, bool) {
	selected = strings.TrimSpace(selected)
	match := slices.IndexFunc(spaces, func(space KibanaSpace) bool {
		return strings.EqualFold(strings.TrimSpace(space.ID), selected) ||
			strings.EqualFold(strings.TrimSpace(space.Name), selected)
	})
	if match == -1 {
		return "", false
	}
	if strings.TrimSpace(spaces[match].Name) == "" {
		return selected, true
	}
	return spaces[match].Name, true
}

func validateStaticSelections(selected []string, allowed []string) error {
	for _, value := range selected {
		if isTemplateExpression(value) {
			continue
		}
		if !containsIgnoreCase(allowed, value) {
			return fmt.Errorf("%q is not supported", value)
		}
	}

	return nil
}

func allowedAlertSeverities() []string {
	return []string{"low", "medium", "high", "critical"}
}

func allowedAlertStatuses() []string {
	return []string{"active", "flapping", "recovered", "untracked"}
}

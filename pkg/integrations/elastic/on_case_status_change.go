package elastic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnCaseStatusChange struct{}

type OnCaseStatusChangeConfiguration struct {
	Statuses   []string                  `json:"statuses" mapstructure:"statuses"`
	Severities []string                  `json:"severities" mapstructure:"severities"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
}

type OnCaseStatusChangeMetadata struct {
	LastPollTime string `json:"lastPollTime,omitempty" mapstructure:"lastPollTime"`
}

func (t *OnCaseStatusChange) Name() string  { return "elastic.onCaseStatusChange" }
func (t *OnCaseStatusChange) Label() string { return "When Case Status Changes" }
func (t *OnCaseStatusChange) Description() string {
	return "React when an Elastic Security case status changes"
}
func (t *OnCaseStatusChange) Icon() string  { return "alert-circle" }
func (t *OnCaseStatusChange) Color() string { return "gray" }

func (t *OnCaseStatusChange) Documentation() string {
	return `The When Case Status Changes trigger fires a workflow execution when a Kibana Security case is updated.

## How it works

1. When the trigger is saved, SuperPlane automatically creates a signed Kibana Webhook connector.
2. In Kibana, open **Stack Management → Rules** and create an **Elasticsearch query** rule.
3. Configure the rule to watch the cases index pattern ` + "`.cases-*`" + ` and use the case update time field as the time field.
4. Attach the **SuperPlane Alert** connector as an action on that rule.
5. Configure the action body to send this JSON:

` + "```" + `json
{
  "eventType": "case_status_changed"
}
` + "```" + `

6. Enable the rule. Each time the rule detects case updates, Kibana calls the SuperPlane webhook.
7. SuperPlane receives the webhook, queries Kibana for cases updated since the last checkpoint, and fires one event per matching case.

## Recommended Kibana setup

- Use an **Elasticsearch query** rule as the default choice.
- Use the cases index pattern ` + "`.cases-*`" + `.
- Use the case update time field as the rule time field.
- Use a rule condition based on case updates, not case creation only.
- Run the rule frequently enough for your workflow latency needs.
- The webhook body only needs ` + "`eventType`" + ` because SuperPlane retrieves the matching case details from Kibana after the webhook arrives.

## Configuration

- **Statuses** *(optional)*: Only fire when a case has one of these statuses. Leave empty to fire for any case update.
- **Severities** *(optional)*: Only fire for cases with one of these severities. Leave empty to accept all severities.
- **Tags** *(optional)*: Only fire for cases that include at least one tag matching any of these predicates. Leave empty to accept all cases.

## Event Data

The trigger emits the full case details including id, title, status, severity, version, tags, description, and timestamps.`
}

func (t *OnCaseStatusChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only fire for cases with one of these statuses. Leave empty to fire for all status values.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeCaseStatus,
					Multi: true,
				},
			},
		},
		{
			Name:        "severities",
			Label:       "Severities",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only fire for cases with one of these severities. Leave empty to fire for all severity values.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeCaseSeverity,
					Multi: true,
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for cases that include at least one tag matching any of these predicates. Leave empty to fire for all cases.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnCaseStatusChange) Setup(ctx core.TriggerContext) error {
	meta := loadCaseStatusChangeMetadata(ctx.Metadata)
	if meta.LastPollTime == "" {
		meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}

	return ctx.Integration.RequestWebhook(map[string]any{"kibanaUrl": string(kibanaURL)})
}

func (t *OnCaseStatusChange) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnCaseStatusChange) HandleAction(_ core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnCaseStatusChange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
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

	if eventType := extractString(payload, "eventType"); eventType != "" && eventType != "case_status_changed" {
		return http.StatusOK, nil, nil
	}

	meta := loadCaseStatusChangeMetadata(ctx.Metadata)
	if meta.LastPollTime == "" {
		return http.StatusOK, nil, nil
	}

	var config OnCaseStatusChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create client: %v", err)
	}

	cases, err := client.ListCasesUpdatedSince(meta.LastPollTime, config.Statuses, config.Severities, nil)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to list cases: %v", err)
	}

	newLastPollTime := meta.LastPollTime
	for _, c := range cases {
		if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, strings.ToLower(c.Status)) {
			continue
		}

		if len(config.Severities) > 0 && !slices.Contains(config.Severities, strings.ToLower(c.Severity)) {
			continue
		}

		if len(config.Tags) > 0 {
			matched := false
			for _, tag := range c.Tags {
				if configuration.MatchesAnyPredicate(config.Tags, tag) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		eventPayload := map[string]any{
			"id":          c.ID,
			"title":       c.Title,
			"status":      c.Status,
			"severity":    c.Severity,
			"version":     c.Version,
			"tags":        c.Tags,
			"description": c.Description,
			"createdAt":   c.CreatedAt,
			"updatedAt":   c.UpdatedAt,
		}
		if err := ctx.Events.Emit("elastic.case.status.changed", eventPayload); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
		}

		if c.UpdatedAt > newLastPollTime {
			newLastPollTime = c.UpdatedAt
		}
	}

	if newLastPollTime != meta.LastPollTime {
		meta.LastPollTime = newLastPollTime
		if err := ctx.Metadata.Set(meta); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnCaseStatusChange) Cleanup(_ core.TriggerContext) error {
	return nil
}

func loadCaseStatusChangeMetadata(metadata core.MetadataContext) OnCaseStatusChangeMetadata {
	var meta OnCaseStatusChangeMetadata
	if metadata == nil {
		return meta
	}
	_ = mapstructure.Decode(metadata.Get(), &meta)
	return meta
}

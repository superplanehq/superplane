package elastic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
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
	RouteKey     string `json:"routeKey,omitempty" mapstructure:"routeKey"`
}


func (t *OnCaseStatusChange) Name() string  { return "elastic.onCaseStatusChange" }
func (t *OnCaseStatusChange) Label() string { return "When Case Status Changes" }
func (t *OnCaseStatusChange) Description() string {
	return "React when an Elastic Security case status changes"
}
func (t *OnCaseStatusChange) Icon() string  { return "alert-circle" }
func (t *OnCaseStatusChange) Color() string { return "gray" }

func (t *OnCaseStatusChange) Documentation() string {
	return `The When Case Status Changes trigger starts a workflow execution when a Kibana Security case is updated.

## Shared Connector

SuperPlane creates **one Kibana Webhook connector per integration**, shared across all triggers that use the same Kibana instance. Each incoming request is routed to the correct trigger instance using two fields in the request body:

- ` + "`eventType`" + `: must be ` + "`\"case_status_changed\"`" + ` — requests with any other value are silently ignored.
- ` + "`routeKey`" + `: a unique ID assigned per trigger node — allows multiple When Case Status Changes nodes on the same canvas to each react independently.

## How it works

1. When the trigger is saved, SuperPlane creates or reuses the shared Kibana Webhook connector.
2. Configure a Kibana rule or automation to POST to the connector with the required body (see below) when a case changes.
3. SuperPlane receives the webhook, queries the Kibana Cases API for cases updated since its stored checkpoint, applies the status filter, and emits one event per matching case.

### Required connector action body

` + "```" + `json
{
  "eventType": "case_status_changed",
  "routeKey":  "<routeKey shown in trigger settings>"
}
` + "```" + `

The ` + "`routeKey`" + ` value is generated when the trigger is saved and is visible in the trigger metadata. It ensures that multiple instances of this trigger on the same canvas each react only to their own events.

## Configuration

- **Statuses** *(optional)*: Only fire when a case transitions to one of these statuses. Leave empty to fire for any case update.

## Webhook Verification

SuperPlane generates a random signing secret and configures the Kibana connector to include it on every request. Requests without the correct secret are rejected automatically.

## Event Data

The trigger emits the full case details including id, title, status, severity, version, and timestamps.`
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
	if ctx.Metadata != nil {
		meta := loadCaseStatusChangeMetadata(ctx.Metadata)
		changed := false
		if meta.LastPollTime == "" {
			meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
			changed = true
		}
		if meta.RouteKey == "" {
			meta.RouteKey = uuid.NewString()
			changed = true
		}
		if changed {
			if err := ctx.Metadata.Set(meta); err != nil {
				return fmt.Errorf("failed to save metadata: %w", err)
			}
		}
	}

	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}

	if err := ctx.Integration.RequestWebhook(map[string]any{"kibanaUrl": string(kibanaURL)}); err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	return nil
}

func (t *OnCaseStatusChange) Actions() []core.Action { return nil }

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

	if extractString(payload, "eventType") != "case_status_changed" {
		return http.StatusOK, nil, nil
	}

	meta := loadCaseStatusChangeMetadata(ctx.Metadata)
	if meta.RouteKey == "" || extractString(payload, "routeKey") != meta.RouteKey {
		return http.StatusOK, nil, nil
	}

	var config OnCaseStatusChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if meta.LastPollTime == "" {
		meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		if ctx.Metadata != nil {
			if err := ctx.Metadata.Set(meta); err != nil {
				return http.StatusInternalServerError, nil, fmt.Errorf("failed to initialize metadata: %w", err)
			}
		}
		return http.StatusOK, nil, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create client: %w", err)
	}

	cases, err := client.ListCasesUpdatedSince(meta.LastPollTime, config.Statuses, config.Severities, nil)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to list cases: %v", err)
		}
		return http.StatusOK, nil, nil
	}

	newLastPollTime := meta.LastPollTime
	for _, c := range cases {
		if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, strings.ToLower(c.Status)) {
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

	if newLastPollTime != meta.LastPollTime && ctx.Metadata != nil {
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

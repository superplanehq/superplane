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

const (
	checkCaseConnectorAction        = "checkCaseConnectorAvailability"
	checkCaseConnectorRetryInterval = 10 * time.Second
)

type OnCaseStatusChange struct{}

type OnCaseStatusChangeConfiguration struct {
	Cases      []string                  `json:"cases" mapstructure:"cases"`
	Statuses   []string                  `json:"statuses" mapstructure:"statuses"`
	Severities []string                  `json:"severities" mapstructure:"severities"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
}

type OnCaseStatusChangeMetadata struct {
	LastPollTime string            `json:"lastPollTime,omitempty" mapstructure:"lastPollTime"`
	CaseNames    map[string]string `json:"caseNames,omitempty" mapstructure:"caseNames"`
	CaseStatuses map[string]string `json:"caseStatuses,omitempty" mapstructure:"caseStatuses"`
	RouteKey     string            `json:"routeKey,omitempty" mapstructure:"routeKey"`
	RuleID       string            `json:"ruleId,omitempty" mapstructure:"ruleId"`
}

func (t *OnCaseStatusChange) Name() string  { return "elastic.onCaseStatusChange" }
func (t *OnCaseStatusChange) Label() string { return "When Case Status Changes" }
func (t *OnCaseStatusChange) Description() string {
	return "React when an Elastic Security case status changes"
}
func (t *OnCaseStatusChange) Icon() string  { return "alert-circle" }
func (t *OnCaseStatusChange) Color() string { return "gray" }

func (t *OnCaseStatusChange) Documentation() string {
	return `The When Case Status Changes trigger fires a workflow execution when a Kibana Security case changes status.

## Shared Connector

SuperPlane creates **one Kibana Webhook connector per integration**, shared across Elastic triggers that use the same Kibana instance. Each incoming request is routed to the correct trigger instance using two fields in the request body:

- ` + "`eventType`" + `: must be ` + "`\"case_status_changed\"`" + `.
- ` + "`routeKey`" + `: a unique ID assigned per trigger node so multiple case-status triggers can coexist safely.

## How it works

1. When the trigger is saved, SuperPlane creates or reuses the shared Kibana Webhook connector.
2. SuperPlane automatically provisions a Kibana **Elasticsearch query** rule against ` + "`.kibana_alerting_cases`" + ` using ` + "`cases.updated_at`" + ` as the time field.
3. Every minute, that Kibana rule checks for case updates in the current window and fires the shared connector when matches are found.
4. SuperPlane receives the webhook, verifies the secret, validates the routing fields, then queries Kibana for cases updated since the stored checkpoint.
5. SuperPlane compares each returned case's current status to the last status stored in trigger metadata and only emits when the value changed.
6. SuperPlane emits one ` + "`elastic.case.status.changed`" + ` event per matching case whose status actually changed.

## Configuration

- **Cases**: Select one or more specific cases to monitor.
- **Statuses** *(optional)*: Only fire when a case has one of these statuses. Leave empty to fire for any case update.
- **Severities** *(optional)*: Only fire for cases with one of these severities. Leave empty to accept all severities.
- **Tags** *(optional)*: Only fire for cases that include at least one tag matching any of these predicates. Leave empty to accept all cases.

## Event Data

The trigger emits the full case details including id, title, status, severity, version, tags, description, and timestamps.`
}

func (t *OnCaseStatusChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "cases",
			Label:       "Cases",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select one or more specific cases to monitor.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeCase,
					Multi: true,
				},
			},
		},
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
	changed := false
	if meta.LastPollTime == "" {
		meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		changed = true
	}
	if meta.RouteKey == "" {
		meta.RouteKey = uuid.NewString()
		changed = true
	}

	var config OnCaseStatusChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if len(config.Cases) == 0 {
		return fmt.Errorf("at least one case is required")
	}

	names, statuses, err := t.resolveCaseMetadata(ctx, config.Cases)
	if err != nil {
		return err
	}
	meta.CaseNames = names
	meta.CaseStatuses = statuses
	changed = true

	if changed {
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}

	if err := ctx.Integration.RequestWebhook(map[string]any{"kibanaUrl": string(kibanaURL)}); err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	if meta.RuleID != "" {
		return nil
	}

	return ctx.Requests.ScheduleActionCall(checkCaseConnectorAction, map[string]any{}, checkCaseConnectorRetryInterval)
}

func (t *OnCaseStatusChange) resolveCaseMetadata(ctx core.TriggerContext, caseIDs []string) (map[string]string, map[string]string, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Elastic client: %w", err)
	}

	names := make(map[string]string, len(caseIDs))
	statuses := make(map[string]string, len(caseIDs))
	for _, id := range caseIDs {
		if strings.Contains(id, "{{") {
			names[id] = id
			continue
		}
		caseResp, err := client.GetCase(id)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get case %s: %w", id, err)
		}
		names[id] = caseResp.Title
		statuses[id] = strings.ToLower(caseResp.Status)
	}
	return names, statuses, nil
}

func (t *OnCaseStatusChange) Actions() []core.Action {
	return []core.Action{
		{
			Name:           checkCaseConnectorAction,
			Description:    "Find the Kibana connector and create the case change rule",
			UserAccessible: false,
		},
	}
}

func (t *OnCaseStatusChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name == checkCaseConnectorAction {
		return nil, t.checkConnectorAndCreateRule(ctx)
	}
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnCaseStatusChange) checkConnectorAndCreateRule(ctx core.TriggerActionContext) error {
	meta := loadCaseStatusChangeMetadata(ctx.Metadata)
	if meta.RuleID != "" {
		return nil
	}
	if meta.RouteKey == "" {
		meta.RouteKey = uuid.NewString()
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to create client: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkCaseConnectorAction, map[string]any{}, checkCaseConnectorRetryInterval)
	}

	connectors, err := client.ListKibanaConnectors()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to list connectors: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkCaseConnectorAction, map[string]any{}, checkCaseConnectorRetryInterval)
	}

	var connectorID string
	for _, connector := range connectors {
		if connector.Name == KibanaConnectorName {
			connectorID = connector.ID
			break
		}
	}

	if connectorID == "" {
		if ctx.Logger != nil {
			ctx.Logger.Infof("elastic onCaseStatusChange: connector %q not found yet, retrying", KibanaConnectorName)
		}
		return ctx.Requests.ScheduleActionCall(checkCaseConnectorAction, map[string]any{}, checkCaseConnectorRetryInterval)
	}

	rule, err := client.CreateKibanaCaseQueryRule(connectorID, meta.RouteKey)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to create rule: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkCaseConnectorAction, map[string]any{}, checkCaseConnectorRetryInterval)
	}

	meta.RuleID = rule.ID
	return ctx.Metadata.Set(meta)
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
	if meta.LastPollTime == "" {
		return http.StatusOK, nil, nil
	}
	if meta.RouteKey == "" || extractString(payload, "routeKey") != meta.RouteKey {
		return http.StatusOK, nil, nil
	}

	var config OnCaseStatusChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if len(config.Cases) == 0 {
		// Keep older misconfigured trigger nodes quiet rather than broadening scope.
		return http.StatusOK, nil, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create client: %v", err)
	}

	cases, err := client.ListCasesUpdatedSince(meta.LastPollTime, nil, nil, nil)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to list cases after webhook: %v", err)
		}
		return http.StatusOK, nil, nil
	}

	newLastPollTime := meta.LastPollTime
	if meta.CaseStatuses == nil {
		meta.CaseStatuses = map[string]string{}
	}
	statusesChanged := false
	for _, c := range cases {
		currentStatus := strings.ToLower(c.Status)
		previousStatus, hasPreviousStatus := meta.CaseStatuses[c.ID]
		if previousStatus != currentStatus {
			meta.CaseStatuses[c.ID] = currentStatus
			statusesChanged = true
		}

		if c.UpdatedAt > newLastPollTime {
			newLastPollTime = c.UpdatedAt
		}

		if !hasPreviousStatus || previousStatus == currentStatus {
			continue
		}

		if len(config.Cases) > 0 && !slices.Contains(config.Cases, c.ID) {
			continue
		}

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
	}

	if newLastPollTime != meta.LastPollTime || statusesChanged {
		meta.LastPollTime = newLastPollTime
		if err := ctx.Metadata.Set(meta); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnCaseStatusChange) Cleanup(ctx core.TriggerContext) error {
	meta := loadCaseStatusChangeMetadata(ctx.Metadata)
	if meta.RuleID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	return client.DeleteKibanaRule(meta.RuleID)
}

func loadCaseStatusChangeMetadata(metadata core.MetadataContext) OnCaseStatusChangeMetadata {
	var meta OnCaseStatusChangeMetadata
	if metadata == nil {
		return meta
	}
	_ = mapstructure.Decode(metadata.Get(), &meta)
	return meta
}

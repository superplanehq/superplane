package sentry

import (
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	alertThresholdTypeAbove   = "above"
	alertThresholdTypeBelow   = "below"
	alertTriggerLabelCritical = "critical"
	alertTriggerLabelWarning  = "warning"
	alertActionTypeEmail      = "email"
	alertTargetTypeUser       = "user"
	alertTargetTypeTeam       = "team"
	alertDatasetEvents        = "events"
	alertQueryTypeEvents      = 0
)

var (
	defaultAlertEventTypes    = []string{"default", "error"}
	alertThresholdTypeOptions = []configuration.FieldOption{
		{Label: "Above", Value: alertThresholdTypeAbove},
		{Label: "Below", Value: alertThresholdTypeBelow},
	}
	alertNotificationTargetOptions = []configuration.FieldOption{
		{Label: "User", Value: alertTargetTypeUser},
		{Label: "Team", Value: alertTargetTypeTeam},
	}
	alertEventTypeOptions = []configuration.FieldOption{
		{Label: "Default", Value: "default"},
		{Label: "Error", Value: "error"},
	}
)

type AlertNotificationConfiguration struct {
	TargetType       string `json:"targetType" mapstructure:"targetType"`
	TargetIdentifier string `json:"targetIdentifier" mapstructure:"targetIdentifier"`
}

type AlertThresholdConfiguration struct {
	Threshold        *float64                       `json:"threshold" mapstructure:"threshold"`
	ResolveThreshold *float64                       `json:"resolveThreshold" mapstructure:"resolveThreshold"`
	Notification     AlertNotificationConfiguration `json:"notification" mapstructure:"notification"`
}

type AlertRuleNodeMetadata struct {
	Project   *ProjectSummary `json:"project,omitempty" mapstructure:"project"`
	AlertName string          `json:"alertName,omitempty" mapstructure:"alertName"`
}

func alertRuleBaseFields(projectRequired bool) []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    projectRequired,
			Description: "Sentry project for this metric alert rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeProject},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    projectRequired,
			Description: "Metric alert rule name",
		},
		{
			Name:        "aggregate",
			Label:       "Aggregate",
			Type:        configuration.FieldTypeString,
			Required:    projectRequired,
			Description: "Sentry metric aggregate expression, such as count()",
			Default:     "count()",
		},
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional search query to scope the alert rule",
		},
		{
			Name:        "timeWindow",
			Label:       "Time Window (Minutes)",
			Type:        configuration.FieldTypeNumber,
			Required:    projectRequired,
			Description: "Rolling evaluation window in minutes",
			Default:     60,
		},
		{
			Name:        "thresholdType",
			Label:       "Threshold Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Whether the alert fires when the metric is above or below the threshold",
			Default:     alertThresholdTypeAbove,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: alertThresholdTypeOptions},
			},
		},
		{
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional environment filter, such as production or staging",
		},
		{
			Name:        "eventTypes",
			Label:       "Event Types",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Event types to include in the metric alert",
			Default:     []string{"default", "error"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{Options: alertEventTypeOptions},
			},
		},
	}
}

func alertThresholdField(label, name string, thresholdRequired bool) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeObject,
		Required:    thresholdRequired,
		Description: label + " configuration",
		TypeOptions: &configuration.TypeOptions{
			Object: &configuration.ObjectTypeOptions{
				Schema: []configuration.Field{
					{
						Name:        "threshold",
						Label:       "Threshold",
						Type:        configuration.FieldTypeNumber,
						Required:    thresholdRequired,
						Description: "Threshold that fires this trigger",
					},
					{
						Name:        "resolveThreshold",
						Label:       "Resolve Threshold",
						Type:        configuration.FieldTypeNumber,
						Required:    false,
						Description: "Optional threshold that resolves this trigger",
						Togglable:   true,
					},
					{
						Name:        "notification",
						Label:       "Notification Target",
						Type:        configuration.FieldTypeObject,
						Required:    thresholdRequired,
						Description: "Who Sentry should notify when this trigger fires",
						TypeOptions: &configuration.TypeOptions{
							Object: &configuration.ObjectTypeOptions{
								Schema: []configuration.Field{
									{
										Name:        "targetType",
										Label:       "Target Type",
										Type:        configuration.FieldTypeSelect,
										Required:    thresholdRequired,
										Description: "Whether the notification goes to a user or team",
										TypeOptions: &configuration.TypeOptions{
											Select: &configuration.SelectTypeOptions{
												Options: alertNotificationTargetOptions,
											},
										},
									},
									{
										Name:        "targetIdentifier",
										Label:       "Target",
										Type:        configuration.FieldTypeIntegrationResource,
										Required:    thresholdRequired,
										Description: "Sentry user or team to notify. Choose Target Type first to load options.",
										TypeOptions: &configuration.TypeOptions{
											Resource: &configuration.ResourceTypeOptions{
												Type: ResourceTypeAlertTarget,
												Parameters: []configuration.ParameterRef{
													{
														Name:      "project",
														ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
													},
													{
														Name:      "alertId",
														ValueFrom: &configuration.ParameterValueFrom{Field: "alertId"},
													},
													{
														Name:      "targetType",
														ValueFrom: &configuration.ParameterValueFrom{Field: "targetType"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func normalizeAlertEventTypes(values []string) []string {
	if len(values) == 0 {
		return append([]string{}, defaultAlertEventTypes...)
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}

		result = append(result, value)
	}

	if len(result) == 0 {
		return append([]string{}, defaultAlertEventTypes...)
	}

	return result
}

// trimAlertEventTypeSelections trims and deduplicates user-provided event types for update flows.
// It returns nil when the configuration omits or clears the list, so callers preserve the existing rule.
func trimAlertEventTypeSelections(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}

	return result
}

func parseAlertThresholdType(value string) int {
	if strings.EqualFold(strings.TrimSpace(value), alertThresholdTypeBelow) {
		return 1
	}

	return 0
}

func buildAlertTriggerInput(
	label string,
	configuration AlertThresholdConfiguration,
) (MetricAlertTriggerInput, error) {
	if configuration.Threshold == nil {
		return MetricAlertTriggerInput{}, fmt.Errorf("%s threshold is required", label)
	}

	notification, err := buildAlertActionInput(configuration.Notification)
	if err != nil {
		return MetricAlertTriggerInput{}, fmt.Errorf("%s notification is invalid: %w", label, err)
	}

	return MetricAlertTriggerInput{
		Label:            label,
		AlertThreshold:   *configuration.Threshold,
		ResolveThreshold: configuration.ResolveThreshold,
		Actions:          []MetricAlertActionInput{notification},
	}, nil
}

func buildAlertActionInput(configuration AlertNotificationConfiguration) (MetricAlertActionInput, error) {
	targetType := strings.TrimSpace(configuration.TargetType)
	targetIdentifier := strings.TrimSpace(configuration.TargetIdentifier)

	if targetType == "" || targetIdentifier == "" {
		return MetricAlertActionInput{}, fmt.Errorf("target type and target identifier are required")
	}

	if targetType != alertTargetTypeUser && targetType != alertTargetTypeTeam {
		return MetricAlertActionInput{}, fmt.Errorf("unsupported target type %q", targetType)
	}

	return MetricAlertActionInput{
		Type:             alertActionTypeEmail,
		TargetType:       targetType,
		TargetIdentifier: targetIdentifier,
	}, nil
}

func buildAlertRequestFromRule(
	alertRule MetricAlertRule,
	project string,
	name string,
	aggregate string,
	query string,
	timeWindow *float64,
	thresholdType string,
	environment string,
	eventTypes []string,
	critical AlertThresholdConfiguration,
	warning AlertThresholdConfiguration,
) (CreateOrUpdateMetricAlertRuleRequest, error) {
	request := CreateOrUpdateMetricAlertRuleRequest{
		Name:          strings.TrimSpace(name),
		Aggregate:     strings.TrimSpace(aggregate),
		TimeWindow:    intOrDefault(timeWindow, int(alertRule.TimeWindow)),
		Projects:      append([]string{}, alertRule.Projects...),
		Query:         query,
		ThresholdType: alertRule.ThresholdType,
		Environment:   normalizeStringOrNil(environment, alertRule.Environment),
		Dataset:       alertDatasetEvents,
		QueryType:     intPointer(alertQueryTypeEvents),
		EventTypes:    append([]string{}, alertRule.EventTypes...),
	}

	if strings.TrimSpace(project) != "" {
		request.Projects = []string{strings.TrimSpace(project)}
	}
	if strings.TrimSpace(request.Name) == "" {
		request.Name = strings.TrimSpace(alertRule.Name)
	}
	if strings.TrimSpace(request.Aggregate) == "" {
		request.Aggregate = strings.TrimSpace(alertRule.Aggregate)
	}
	if strings.TrimSpace(query) == "" {
		request.Query = strings.TrimSpace(alertRule.Query)
	}
	if strings.TrimSpace(thresholdType) != "" {
		request.ThresholdType = parseAlertThresholdType(thresholdType)
	}
	if len(eventTypes) > 0 {
		request.EventTypes = normalizeAlertEventTypes(eventTypes)
	}
	if len(request.EventTypes) == 0 {
		request.EventTypes = append([]string{}, defaultAlertEventTypes...)
	}

	triggers, err := mergeAlertTriggers(alertRule.Triggers, critical, warning)
	if err != nil {
		return CreateOrUpdateMetricAlertRuleRequest{}, err
	}

	request.Triggers = triggers
	return request, validateAlertRequest(request)
}

func buildCreateAlertRequest(
	project string,
	name string,
	aggregate string,
	query string,
	timeWindow float64,
	thresholdType string,
	environment string,
	eventTypes []string,
	critical AlertThresholdConfiguration,
	warning AlertThresholdConfiguration,
) (CreateOrUpdateMetricAlertRuleRequest, error) {
	request := CreateOrUpdateMetricAlertRuleRequest{
		Name:          strings.TrimSpace(name),
		Aggregate:     strings.TrimSpace(aggregate),
		TimeWindow:    int(timeWindow),
		Projects:      []string{strings.TrimSpace(project)},
		Query:         strings.TrimSpace(query),
		ThresholdType: parseAlertThresholdType(thresholdType),
		Environment:   strings.TrimSpace(environment),
		Dataset:       alertDatasetEvents,
		QueryType:     intPointer(alertQueryTypeEvents),
		EventTypes:    normalizeAlertEventTypes(eventTypes),
	}

	criticalTrigger, err := buildAlertTriggerInput(alertTriggerLabelCritical, critical)
	if err != nil {
		return CreateOrUpdateMetricAlertRuleRequest{}, err
	}

	request.Triggers = append(request.Triggers, criticalTrigger)

	if warning.Threshold != nil {
		warningTrigger, err := buildAlertTriggerInput(alertTriggerLabelWarning, warning)
		if err != nil {
			return CreateOrUpdateMetricAlertRuleRequest{}, err
		}

		request.Triggers = append(request.Triggers, warningTrigger)
	}

	return request, validateAlertRequest(request)
}

func validateAlertRequest(request CreateOrUpdateMetricAlertRuleRequest) error {
	if strings.TrimSpace(request.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(request.Aggregate) == "" {
		return fmt.Errorf("aggregate is required")
	}
	if request.TimeWindow <= 0 {
		return fmt.Errorf("timeWindow must be greater than 0")
	}
	if len(request.Projects) == 0 || strings.TrimSpace(request.Projects[0]) == "" {
		return fmt.Errorf("project is required")
	}
	if len(request.Triggers) == 0 {
		return fmt.Errorf("at least one trigger is required")
	}

	return nil
}

func mergeAlertTriggers(
	existing []MetricAlertTrigger,
	critical AlertThresholdConfiguration,
	warning AlertThresholdConfiguration,
) ([]MetricAlertTriggerInput, error) {
	result := make([]MetricAlertTriggerInput, 0, len(existing))
	handledLabels := map[string]bool{}

	for _, trigger := range existing {
		label := strings.TrimSpace(trigger.Label)
		if label == "" {
			continue
		}

		switch label {
		case alertTriggerLabelCritical:
			merged, err := mergeAlertTrigger(trigger, critical)
			if err != nil {
				return nil, fmt.Errorf("critical trigger is invalid: %w", err)
			}
			result = append(result, merged)
			handledLabels[label] = true
		case alertTriggerLabelWarning:
			merged, err := mergeAlertTrigger(trigger, warning)
			if err != nil {
				return nil, fmt.Errorf("warning trigger is invalid: %w", err)
			}
			result = append(result, merged)
			handledLabels[label] = true
		default:
			result = append(result, metricAlertTriggerToInput(trigger))
			handledLabels[label] = true
		}
	}

	if !handledLabels[alertTriggerLabelCritical] && critical.Threshold != nil {
		created, err := buildAlertTriggerInput(alertTriggerLabelCritical, critical)
		if err != nil {
			return nil, err
		}
		result = append(result, created)
	}

	if !handledLabels[alertTriggerLabelWarning] && warning.Threshold != nil {
		created, err := buildAlertTriggerInput(alertTriggerLabelWarning, warning)
		if err != nil {
			return nil, err
		}
		result = append(result, created)
	}

	return result, nil
}

func mergeAlertTrigger(
	existing MetricAlertTrigger,
	override AlertThresholdConfiguration,
) (MetricAlertTriggerInput, error) {
	result := metricAlertTriggerToInput(existing)

	if override.Threshold != nil {
		result.AlertThreshold = *override.Threshold
	}
	if override.ResolveThreshold != nil {
		result.ResolveThreshold = override.ResolveThreshold
	}
	if strings.TrimSpace(override.Notification.TargetType) != "" ||
		strings.TrimSpace(override.Notification.TargetIdentifier) != "" {
		action, err := buildAlertActionInput(override.Notification)
		if err != nil {
			return MetricAlertTriggerInput{}, err
		}
		result.Actions = []MetricAlertActionInput{action}
	}

	if result.AlertThreshold == 0 && override.Threshold == nil {
		if threshold, ok := floatFromAny(existing.AlertThreshold); ok {
			result.AlertThreshold = threshold
		}
	}
	if len(result.Actions) == 0 {
		return MetricAlertTriggerInput{}, fmt.Errorf("at least one notification target is required")
	}

	return result, nil
}

func metricAlertTriggerToInput(trigger MetricAlertTrigger) MetricAlertTriggerInput {
	result := MetricAlertTriggerInput{
		Label:   trigger.Label,
		Actions: make([]MetricAlertActionInput, 0, len(trigger.Actions)),
	}

	if threshold, ok := floatFromAny(trigger.AlertThreshold); ok {
		result.AlertThreshold = threshold
	}
	if resolveThreshold, ok := floatFromAny(trigger.ResolveThreshold); ok {
		result.ResolveThreshold = &resolveThreshold
	}

	for _, action := range trigger.Actions {
		result.Actions = append(result.Actions, MetricAlertActionInput{
			Type:             action.Type,
			TargetType:       action.TargetType,
			TargetIdentifier: action.TargetIdentifier,
			InputChannelID:   action.InputChannelID,
			IntegrationID:    action.IntegrationID,
			Priority:         action.Priority,
		})
	}

	return result
}

func intOrDefault(value *float64, fallback int) int {
	if value == nil {
		return fallback
	}

	return int(*value)
}

func intPointer(value int) *int {
	return &value
}

func normalizeStringOrNil(value string, fallback any) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}

	if fallbackString, ok := fallback.(string); ok {
		return strings.TrimSpace(fallbackString)
	}

	return ""
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		return 0, false
	default:
		return 0, false
	}
}

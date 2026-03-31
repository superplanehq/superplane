package grafana

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	defaultAlertRuleCondition     = "A"
	defaultAlertRuleLookback      = 300
	defaultAlertRuleIntervalMS    = 1000
	defaultAlertRuleMaxDataPoints = 43200
	resourceTypeFolder            = "folder"
)

type AlertRuleKeyValuePair struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

type CreateAlertRuleSpec struct {
	Title           string                   `json:"title" mapstructure:"title"`
	FolderUID       string                   `json:"folderUID" mapstructure:"folderUID"`
	RuleGroup       string                   `json:"ruleGroup" mapstructure:"ruleGroup"`
	DataSourceUID   string                   `json:"dataSourceUid" mapstructure:"dataSourceUid"`
	Query           string                   `json:"query" mapstructure:"query"`
	LookbackSeconds int                      `json:"lookbackSeconds" mapstructure:"lookbackSeconds"`
	For             string                   `json:"for" mapstructure:"for"`
	NoDataState     string                   `json:"noDataState" mapstructure:"noDataState"`
	ExecErrState    string                   `json:"execErrState" mapstructure:"execErrState"`
	Labels          *[]AlertRuleKeyValuePair `json:"labels,omitempty" mapstructure:"labels"`
	Annotations     *[]AlertRuleKeyValuePair `json:"annotations,omitempty" mapstructure:"annotations"`
	IsPaused        bool                     `json:"isPaused" mapstructure:"isPaused"`
}

type GetAlertRuleSpec struct {
	AlertRuleUID string `json:"alertRuleUid" mapstructure:"alertRuleUid"`
}

type UpdateAlertRuleSpec struct {
	AlertRuleUID        string `json:"alertRuleUid" mapstructure:"alertRuleUid"`
	CreateAlertRuleSpec `mapstructure:",squash"`
}

func decodeCreateAlertRuleSpec(input any) (CreateAlertRuleSpec, error) {
	spec := CreateAlertRuleSpec{
		LookbackSeconds: defaultAlertRuleLookback,
		For:             "5m",
		NoDataState:     "NoData",
		ExecErrState:    "Alerting",
	}
	if err := mapstructure.Decode(input, &spec); err != nil {
		return CreateAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	return sanitizeCreateAlertRuleSpec(spec), nil
}

func decodeGetAlertRuleSpec(input any) (GetAlertRuleSpec, error) {
	spec := GetAlertRuleSpec{}
	if err := mapstructure.Decode(input, &spec); err != nil {
		return GetAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.AlertRuleUID = strings.TrimSpace(spec.AlertRuleUID)
	return spec, nil
}

func decodeUpdateAlertRuleSpec(input any) (UpdateAlertRuleSpec, error) {
	spec := UpdateAlertRuleSpec{
		CreateAlertRuleSpec: CreateAlertRuleSpec{
			LookbackSeconds: defaultAlertRuleLookback,
			For:             "5m",
			NoDataState:     "NoData",
			ExecErrState:    "Alerting",
		},
	}
	if err := mapstructure.Decode(input, &spec); err != nil {
		return UpdateAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.AlertRuleUID = strings.TrimSpace(spec.AlertRuleUID)
	spec.CreateAlertRuleSpec = sanitizeCreateAlertRuleSpec(spec.CreateAlertRuleSpec)
	return spec, nil
}

func sanitizeCreateAlertRuleSpec(spec CreateAlertRuleSpec) CreateAlertRuleSpec {
	spec.Title = strings.TrimSpace(spec.Title)
	spec.FolderUID = strings.TrimSpace(spec.FolderUID)
	spec.RuleGroup = strings.TrimSpace(spec.RuleGroup)
	spec.DataSourceUID = strings.TrimSpace(spec.DataSourceUID)
	spec.Query = strings.TrimSpace(spec.Query)
	spec.For = strings.TrimSpace(spec.For)
	spec.NoDataState = strings.TrimSpace(spec.NoDataState)
	spec.ExecErrState = strings.TrimSpace(spec.ExecErrState)

	if spec.LookbackSeconds <= 0 {
		spec.LookbackSeconds = defaultAlertRuleLookback
	}

	if spec.For == "" {
		spec.For = "5m"
	}

	if spec.NoDataState == "" {
		spec.NoDataState = "NoData"
	}

	if spec.ExecErrState == "" {
		spec.ExecErrState = "Alerting"
	}

	if spec.Labels != nil {
		normalized := normalizeAlertRuleKeyValuePairs(*spec.Labels)
		spec.Labels = &normalized
	}

	if spec.Annotations != nil {
		normalized := normalizeAlertRuleKeyValuePairs(*spec.Annotations)
		spec.Annotations = &normalized
	}

	return spec
}

func validateCreateAlertRuleSpec(spec CreateAlertRuleSpec) error {
	if spec.Title == "" {
		return errors.New("title is required")
	}
	if spec.FolderUID == "" {
		return errors.New("folderUID is required")
	}
	if spec.RuleGroup == "" {
		return errors.New("ruleGroup is required")
	}
	if spec.DataSourceUID == "" {
		return errors.New("dataSourceUid is required")
	}
	if spec.Query == "" {
		return errors.New("query is required")
	}
	if spec.LookbackSeconds <= 0 {
		return errors.New("lookbackSeconds must be greater than 0")
	}

	return nil
}

func validateGetAlertRuleSpec(spec GetAlertRuleSpec) error {
	if spec.AlertRuleUID == "" {
		return errors.New("alertRuleUid is required")
	}

	return nil
}

func validateUpdateAlertRuleSpec(spec UpdateAlertRuleSpec) error {
	if spec.AlertRuleUID == "" {
		return errors.New("alertRuleUid is required")
	}

	return validateCreateAlertRuleSpec(spec.CreateAlertRuleSpec)
}

func alertRuleStateOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "No Data", Value: "NoData"},
		{Label: "Alerting", Value: "Alerting"},
		{Label: "OK", Value: "OK"},
		{Label: "Keep Last State", Value: "KeepLast"},
	}
}

func alertRuleKeyValueListSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "key",
			Label:    "Key",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "value",
			Label:    "Value",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func alertRuleFieldConfiguration(includeAlertRuleSelector bool) []configuration.Field {
	fields := make([]configuration.Field, 0, 12)
	if includeAlertRuleSelector {
		fields = append(fields, configuration.Field{
			Name:        "alertRuleUid",
			Label:       "Alert Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana alert rule to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeAlertRule,
				},
			},
		})
	}

	fields = append(fields,
		configuration.Field{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "High error rate",
			Description: "Human-readable alert rule title shown in Grafana",
		},
		configuration.Field{
			Name:        "folderUID",
			Label:       "Folder",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana folder that will contain the alert rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeFolder,
				},
			},
		},
		configuration.Field{
			Name:        "ruleGroup",
			Label:       "Rule Group",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "service-health",
			Description: "The Grafana rule group to create or update",
		},
		configuration.Field{
			Name:        "dataSourceUid",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana data source the alert query should use",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDataSource,
				},
			},
		},
		configuration.Field{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Default:     `sum(rate(http_requests_total{status=~"5.."}[5m]))`,
			Description: "The alert query expression Grafana should evaluate",
		},
		configuration.Field{
			Name:        "lookbackSeconds",
			Label:       "Lookback Window (Seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     defaultAlertRuleLookback,
			Description: "How far back Grafana should query when evaluating the rule",
		},
		configuration.Field{
			Name:        "for",
			Label:       "For",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "5m",
			Description: "How long the condition must remain true before the alert fires",
		},
		configuration.Field{
			Name:     "noDataState",
			Label:    "No Data State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "NoData",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertRuleStateOptions(),
				},
			},
			Description: "How Grafana should behave when the query returns no data",
		},
		configuration.Field{
			Name:     "execErrState",
			Label:    "Execution Error State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "Alerting",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertRuleStateOptions(),
				},
			},
			Description: "How Grafana should behave when the query evaluation errors",
		},
		configuration.Field{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Labels to attach to the Grafana alert rule",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: alertRuleKeyValueListSchema(),
					},
				},
			},
		},
		configuration.Field{
			Name:        "annotations",
			Label:       "Annotations",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Annotations to attach to the Grafana alert rule",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Annotation",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: alertRuleKeyValueListSchema(),
					},
				},
			},
		},
		configuration.Field{
			Name:        "isPaused",
			Label:       "Paused",
			Type:        configuration.FieldTypeBool,
			Required:    true,
			Default:     false,
			Description: "Whether the alert rule should be created or updated in a paused state",
		},
	)

	return fields
}

func buildAlertRulePayload(spec CreateAlertRuleSpec) map[string]any {
	return map[string]any{
		"title":        spec.Title,
		"folderUID":    spec.FolderUID,
		"ruleGroup":    spec.RuleGroup,
		"condition":    defaultAlertRuleCondition,
		"data":         buildAlertRuleQueryData(spec),
		"noDataState":  spec.NoDataState,
		"execErrState": spec.ExecErrState,
		"for":          spec.For,
		"annotations":  keyValuePairsToMap(spec.Annotations),
		"labels":       keyValuePairsToMap(spec.Labels),
		"isPaused":     spec.IsPaused,
	}
}

func mergeAlertRulePayload(existing map[string]any, spec CreateAlertRuleSpec, uid string) map[string]any {
	updated := make(map[string]any, len(existing)+10)
	for key, value := range existing {
		updated[key] = value
	}

	for key, value := range buildAlertRulePayload(spec) {
		updated[key] = value
	}

	updated["uid"] = uid
	return updated
}

func buildAlertRuleQueryData(spec CreateAlertRuleSpec) []map[string]any {
	return []map[string]any{
		{
			"refId":     defaultAlertRuleCondition,
			"queryType": "",
			"relativeTimeRange": map[string]any{
				"from": spec.LookbackSeconds,
				"to":   0,
			},
			"datasourceUid": spec.DataSourceUID,
			"model": map[string]any{
				"editorMode":    "code",
				"expr":          spec.Query,
				"query":         spec.Query,
				"intervalMs":    defaultAlertRuleIntervalMS,
				"maxDataPoints": defaultAlertRuleMaxDataPoints,
				"refId":         defaultAlertRuleCondition,
			},
		},
	}
}

func normalizeAlertRuleKeyValuePairs(pairs []AlertRuleKeyValuePair) []AlertRuleKeyValuePair {
	normalized := make([]AlertRuleKeyValuePair, 0, len(pairs))
	for _, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		value := strings.TrimSpace(pair.Value)
		if key == "" || value == "" {
			continue
		}

		normalized = append(normalized, AlertRuleKeyValuePair{
			Key:   key,
			Value: value,
		})
	}

	return normalized
}

func keyValuePairsToMap(pairs *[]AlertRuleKeyValuePair) map[string]string {
	values := map[string]string{}
	if pairs == nil {
		return values
	}

	for _, pair := range *pairs {
		if pair.Key == "" || pair.Value == "" {
			continue
		}

		values[pair.Key] = pair.Value
	}

	return values
}

package grafana

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	alertRuleConditionRefID  = "A"
	alertRuleQueryIntervalMS = 1000
	alertRuleMaxDataPoints   = 43200
	resourceTypeFolder       = "folder"
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
	AlertRuleUID    string                   `json:"alertRuleUid" mapstructure:"alertRuleUid"`
	Title           *string                  `json:"title,omitempty" mapstructure:"title"`
	FolderUID       *string                  `json:"folderUID,omitempty" mapstructure:"folderUID"`
	RuleGroup       *string                  `json:"ruleGroup,omitempty" mapstructure:"ruleGroup"`
	DataSourceUID   *string                  `json:"dataSourceUid,omitempty" mapstructure:"dataSourceUid"`
	Query           *string                  `json:"query,omitempty" mapstructure:"query"`
	LookbackSeconds *int                     `json:"lookbackSeconds,omitempty" mapstructure:"lookbackSeconds"`
	For             *string                  `json:"for,omitempty" mapstructure:"for"`
	NoDataState     *string                  `json:"noDataState,omitempty" mapstructure:"noDataState"`
	ExecErrState    *string                  `json:"execErrState,omitempty" mapstructure:"execErrState"`
	Labels          *[]AlertRuleKeyValuePair `json:"labels,omitempty" mapstructure:"labels"`
	Annotations     *[]AlertRuleKeyValuePair `json:"annotations,omitempty" mapstructure:"annotations"`
	IsPaused        *bool                    `json:"isPaused,omitempty" mapstructure:"isPaused"`
}

func decodeCreateAlertRuleSpec(input any) (CreateAlertRuleSpec, error) {
	spec := CreateAlertRuleSpec{}
	if err := decodeAlertRuleSpec(input, &spec); err != nil {
		return CreateAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	return sanitizeCreateAlertRuleSpec(spec), nil
}

func decodeGetAlertRuleSpec(input any) (GetAlertRuleSpec, error) {
	spec := GetAlertRuleSpec{}
	if err := decodeAlertRuleSpec(input, &spec); err != nil {
		return GetAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.AlertRuleUID = strings.TrimSpace(spec.AlertRuleUID)
	return spec, nil
}

func decodeUpdateAlertRuleSpec(input any) (UpdateAlertRuleSpec, error) {
	spec := UpdateAlertRuleSpec{}
	if err := decodeAlertRuleSpec(input, &spec); err != nil {
		return UpdateAlertRuleSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	return sanitizeUpdateAlertRuleSpec(spec), nil
}

func decodeAlertRuleSpec(input any, result any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           result,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
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

func sanitizeUpdateAlertRuleSpec(spec UpdateAlertRuleSpec) UpdateAlertRuleSpec {
	spec.AlertRuleUID = strings.TrimSpace(spec.AlertRuleUID)
	spec.Title = sanitizeOptionalAlertRuleString(spec.Title)
	spec.FolderUID = sanitizeOptionalAlertRuleString(spec.FolderUID)
	spec.RuleGroup = sanitizeOptionalAlertRuleString(spec.RuleGroup)
	spec.DataSourceUID = sanitizeOptionalAlertRuleString(spec.DataSourceUID)
	spec.Query = sanitizeOptionalAlertRuleString(spec.Query)
	spec.For = sanitizeOptionalAlertRuleString(spec.For)
	spec.NoDataState = sanitizeOptionalAlertRuleString(spec.NoDataState)
	spec.ExecErrState = sanitizeOptionalAlertRuleString(spec.ExecErrState)

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

func sanitizeOptionalAlertRuleString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func validateAlertRuleUpdateSupport(rule map[string]any) error {
	provenance, _ := rule["provenance"].(string)
	provenance = strings.TrimSpace(provenance)
	if provenance == "" || strings.EqualFold(provenance, "api") {
		return nil
	}

	if strings.EqualFold(provenance, "file") {
		return errors.New(
			"file-provisioned Grafana alert rules cannot be updated via the provisioning API; update the provisioning file or recreate the rule as an API-managed rule",
		)
	}

	return fmt.Errorf("Grafana alert rules with provenance %q cannot be updated via the provisioning API", provenance)
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
	if !spec.HasUpdates() {
		return errors.New("at least one field to update must be provided")
	}
	if spec.LookbackSeconds != nil && *spec.LookbackSeconds <= 0 {
		return errors.New("lookbackSeconds must be greater than 0")
	}

	return nil
}

func (spec UpdateAlertRuleSpec) HasUpdates() bool {
	return spec.Title != nil ||
		spec.FolderUID != nil ||
		spec.RuleGroup != nil ||
		spec.DataSourceUID != nil ||
		spec.Query != nil ||
		spec.LookbackSeconds != nil ||
		spec.For != nil ||
		spec.NoDataState != nil ||
		spec.ExecErrState != nil ||
		spec.Labels != nil ||
		spec.Annotations != nil ||
		spec.IsPaused != nil
}

func alertRuleNoDataStateOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "No Data", Value: "NoData"},
		{Label: "Alerting", Value: "Alerting"},
		{Label: "OK", Value: "OK"},
		{Label: "Keep Last State", Value: "KeepLast"},
	}
}

func alertRuleExecErrStateOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "Error", Value: "Error"},
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

func alertRuleFieldConfiguration(includeAlertRuleSelector bool, partialUpdate bool) []configuration.Field {
	fieldRequired := !partialUpdate

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
			Required:    fieldRequired,
			Description: "Human-readable alert rule title shown in Grafana",
		},
		configuration.Field{
			Name:        "folderUID",
			Label:       "Folder",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    fieldRequired,
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
			Required:    fieldRequired,
			Description: "The Grafana rule group to create or update",
		},
		configuration.Field{
			Name:        "dataSourceUid",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    fieldRequired,
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
			Required:    fieldRequired,
			Description: "The alert query expression Grafana should evaluate",
		},
		configuration.Field{
			Name:        "lookbackSeconds",
			Label:       "Lookback Window (Seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    fieldRequired,
			Description: "How far back Grafana should query when evaluating the rule",
		},
		configuration.Field{
			Name:        "for",
			Label:       "For",
			Type:        configuration.FieldTypeString,
			Required:    fieldRequired,
			Description: "How long the condition must remain true before the alert fires",
		},
		configuration.Field{
			Name:     "noDataState",
			Label:    "No Data State",
			Type:     configuration.FieldTypeSelect,
			Required: fieldRequired,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertRuleNoDataStateOptions(),
				},
			},
			Description: "How Grafana should behave when the query returns no data",
		},
		configuration.Field{
			Name:     "execErrState",
			Label:    "Execution Error State",
			Type:     configuration.FieldTypeSelect,
			Required: fieldRequired,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertRuleExecErrStateOptions(),
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
			Required:    false,
			Togglable:   false,
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
		"condition":    alertRuleConditionRefID,
		"data":         buildAlertRuleQueryData(spec),
		"noDataState":  spec.NoDataState,
		"execErrState": spec.ExecErrState,
		"for":          spec.For,
		"annotations":  keyValuePairsToMap(spec.Annotations),
		"labels":       keyValuePairsToMap(spec.Labels),
		"isPaused":     spec.IsPaused,
	}
}

func mergeAlertRulePayload(existing map[string]any, spec UpdateAlertRuleSpec) (map[string]any, error) {
	updated := sanitizeExistingAlertRulePayload(existing)

	if spec.Title != nil {
		updated["title"] = *spec.Title
	}
	if spec.FolderUID != nil {
		updated["folderUID"] = *spec.FolderUID
	}
	if spec.RuleGroup != nil {
		updated["ruleGroup"] = *spec.RuleGroup
	}
	if spec.For != nil {
		updated["for"] = *spec.For
	}
	if spec.NoDataState != nil {
		updated["noDataState"] = *spec.NoDataState
	}
	if spec.ExecErrState != nil {
		updated["execErrState"] = *spec.ExecErrState
	}
	if spec.Labels != nil {
		updated["labels"] = keyValuePairsToMap(spec.Labels)
	}
	if spec.Annotations != nil {
		updated["annotations"] = keyValuePairsToMap(spec.Annotations)
	}
	if spec.IsPaused != nil {
		updated["isPaused"] = *spec.IsPaused
	}
	if spec.DataSourceUID != nil || spec.Query != nil || spec.LookbackSeconds != nil {
		queryData, err := mergeAlertRuleQueryData(existing["data"], spec)
		if err != nil {
			return nil, err
		}
		updated["data"] = queryData
	}

	updated["uid"] = spec.AlertRuleUID
	return updated, nil
}

func sanitizeExistingAlertRulePayload(existing map[string]any) map[string]any {
	sanitized := cloneAlertRuleMap(existing)
	delete(sanitized, "updated")
	delete(sanitized, "provenance")

	copyAlertRuleOrganizationFields(sanitized, existing)

	return sanitized
}

func copyAlertRuleOrganizationFields(destination map[string]any, source map[string]any) {
	if value, exists := source["orgID"]; exists {
		destination["orgID"] = value
		destination["orgId"] = value
		return
	}

	if value, exists := source["orgId"]; exists {
		destination["orgID"] = value
		destination["orgId"] = value
	}
}

func buildAlertRuleQueryData(spec CreateAlertRuleSpec) []map[string]any {
	return []map[string]any{
		{
			"refId":     alertRuleConditionRefID,
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
				"intervalMs":    alertRuleQueryIntervalMS,
				"maxDataPoints": alertRuleMaxDataPoints,
				"refId":         alertRuleConditionRefID,
			},
		},
	}
}

func mergeAlertRuleQueryData(existingData any, spec UpdateAlertRuleSpec) ([]any, error) {
	data, ok := existingData.([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("existing alert rule is missing query data")
	}

	firstQuery, ok := data[0].(map[string]any)
	if !ok {
		return nil, errors.New("existing alert rule query data is invalid")
	}

	updatedData := make([]any, len(data))
	copy(updatedData, data)

	updatedQuery := cloneAlertRuleMap(firstQuery)
	model := cloneAlertRuleMapFromValue(updatedQuery["model"])

	if spec.DataSourceUID != nil {
		updatedQuery["datasourceUid"] = *spec.DataSourceUID

		datasource := cloneAlertRuleMapFromValue(model["datasource"])
		if len(datasource) > 0 {
			datasource["uid"] = *spec.DataSourceUID
			model["datasource"] = datasource
		}
	}

	if spec.LookbackSeconds != nil {
		relativeTimeRange := cloneAlertRuleMapFromValue(updatedQuery["relativeTimeRange"])
		relativeTimeRange["from"] = *spec.LookbackSeconds
		if _, exists := relativeTimeRange["to"]; !exists {
			relativeTimeRange["to"] = 0
		}
		updatedQuery["relativeTimeRange"] = relativeTimeRange
	}

	if spec.Query != nil {
		if _, exists := updatedQuery["refId"]; !exists {
			updatedQuery["refId"] = alertRuleConditionRefID
		}

		model["editorMode"] = "code"
		model["expr"] = *spec.Query
		model["query"] = *spec.Query
		if _, exists := model["expression"]; exists {
			model["expression"] = *spec.Query
		}
		if _, exists := model["intervalMs"]; !exists {
			model["intervalMs"] = alertRuleQueryIntervalMS
		}
		if _, exists := model["maxDataPoints"]; !exists {
			model["maxDataPoints"] = alertRuleMaxDataPoints
		}
		if _, exists := model["refId"]; !exists {
			model["refId"] = updatedQuery["refId"]
		}

		updatedQuery["model"] = model
	}

	if spec.DataSourceUID != nil && spec.Query == nil {
		updatedQuery["model"] = model
	}

	updatedData[0] = updatedQuery
	return updatedData, nil
}

func cloneAlertRuleMap(value map[string]any) map[string]any {
	cloned := make(map[string]any, len(value))
	for key, entry := range value {
		cloned[key] = entry
	}

	return cloned
}

func cloneAlertRuleMapFromValue(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	record, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return cloneAlertRuleMap(record)
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

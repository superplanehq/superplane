package dash0

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListIssues struct{}

type ListIssuesSpec struct {
	CheckRules []string `json:"checkRules,omitempty"`
}

type ListIssuesNodeMetadata struct {
	CheckRules []CheckRule `json:"checkRules" mapstructure:"checkRules"`
}

func (l *ListIssues) Name() string {
	return "dash0.listIssues"
}

func (l *ListIssues) Label() string {
	return "List Issues"
}

func (l *ListIssues) Description() string {
	return "Query Dash0 to get a list of all current issues using the metric dash0.issue.status"
}

func (l *ListIssues) Icon() string {
	return "alert-triangle"
}

func (l *ListIssues) Color() string {
	return "orange"
}

func (l *ListIssues) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListIssues) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:      "checkRules",
			Label:     "Check Rules",
			Type:      configuration.FieldTypeAppInstallationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "check-rule",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
			Description: "Select one or more check rules to filter issues",
		},
	}
}

func (l *ListIssues) Setup(ctx core.SetupContext) error {
	// Fetch check rules once during setup and store them in node metadata
	// This avoids making API calls on every Execute() invocation
	var nodeMetadata ListIssuesNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	//
	// If check rules are already set, skip setup
	//
	if len(nodeMetadata.CheckRules) > 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client during setup to fetch check rules: %w", err)
	}

	checkRules, err := client.ListCheckRules()
	if err != nil {
		return fmt.Errorf("error fetching check rules during setup: %w", err)
	}

	// Store check rules in node metadata for reuse in Execute()
	return ctx.Metadata.Set(ListIssuesNodeMetadata{
		CheckRules: checkRules,
	})
}

func (l *ListIssues) Execute(ctx core.ExecutionContext) error {
	spec := ListIssuesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Execute the query to get all current issues
	query := `{otel_metric_name="dash0.issue.status"} >= 1`
	data, err := client.ExecutePrometheusInstantQuery(query, "default")
	if err != nil {
		return fmt.Errorf("failed to execute Prometheus query: %v", err)
	}

	// Get check rules from node metadata (stored during Setup())
	if ctx.NodeMetadata != nil {
		var nodeMetadata ListIssuesNodeMetadata
		err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
		if err != nil {
			ctx.Logger.Warnf("Error decoding node metadata for check rules: %v", err)
			// Continue without filtering if metadata cannot be decoded
		} else if len(spec.CheckRules) > 0 && len(nodeMetadata.CheckRules) > 0 {
			// Filter issues by check rules if check rules are specified in configuration
			data = l.filterIssuesByCheckRules(data, spec.CheckRules, nodeMetadata.CheckRules)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.issues.list",
		[]any{data},
	)
}

// buildCheckRuleLookupSets creates lookup sets for check rule names and IDs
// to enable efficient matching during filtering.
func buildCheckRuleLookupSets(checkRuleNames []string, allCheckRules []CheckRule) (checkRuleNameSet, checkRuleIDSet map[string]bool) {
	checkRuleNameSet = make(map[string]bool, len(checkRuleNames))
	checkRuleIDSet = make(map[string]bool, len(checkRuleNames))

	// Build a mapping from name to ID
	nameToID := make(map[string]string, len(allCheckRules))
	for _, rule := range allCheckRules {
		nameToID[rule.Name] = rule.ID
		if rule.Name == "" {
			nameToID[rule.ID] = rule.ID // Handle case where name is empty
		}
	}

	// Add selected names and their corresponding IDs to the sets
	for _, name := range checkRuleNames {
		checkRuleNameSet[name] = true
		// Also add the ID if we have a mapping
		if id, ok := nameToID[name]; ok {
			checkRuleIDSet[id] = true
		}
		// Also check if the name itself is an ID
		checkRuleIDSet[name] = true
	}

	return checkRuleNameSet, checkRuleIDSet
}

// extractPrometheusResults extracts and normalizes the Prometheus response data structure.
// It handles both struct and map representations of the response.
func extractPrometheusResults(data map[string]any) (resultValue []any, dataValue map[string]any, ok bool) {
	if dataStruct, ok := data["data"].(PrometheusResponseData); ok {
		// Convert struct to map for easier manipulation
		resultValue = make([]any, len(dataStruct.Result))
		for i, r := range dataStruct.Result {
			resultValue[i] = map[string]any{
				"metric": r.Metric,
				"value":  r.Value,
				"values": r.Values,
			}
		}
		dataValue = map[string]any{
			"resultType": dataStruct.ResultType,
			"result":     resultValue,
		}
		return resultValue, dataValue, true
	}

	if dataMap, ok := data["data"].(map[string]any); ok {
		dataValue = dataMap
		var ok bool
		resultValue, ok = dataValue["result"].([]any)
		if !ok {
			return nil, nil, false
		}
		return resultValue, dataValue, true
	}

	return nil, nil, false
}

// extractMetricFromResult extracts the metric map from a Prometheus result item.
func extractMetricFromResult(resultItem map[string]any) (map[string]string, bool) {
	if metricMap, ok := resultItem["metric"].(map[string]any); ok {
		// Convert map[string]any to map[string]string
		metricValue := make(map[string]string)
		for k, v := range metricMap {
			if strVal, ok := v.(string); ok {
				metricValue[k] = strVal
			}
		}
		return metricValue, true
	}

	if metricStrMap, ok := resultItem["metric"].(map[string]string); ok {
		return metricStrMap, true
	}

	return nil, false
}

// extractCheckRuleFromMetric finds the check rule value in metric labels.
// It checks common label names first, then falls back to checking all labels
// for values that match the check rule sets.
func extractCheckRuleFromMetric(metricValue map[string]string, checkRuleNameSet, checkRuleIDSet map[string]bool) string {
	// Check various possible label names for check rule information
	// Common label names: check_rule, check_rule_name, check_rule_id, rule_name, check_name, check
	labelNamesToCheck := []string{"check_rule", "check_rule_name", "check_rule_id", "rule_name", "check_name", "check", "check_id", "rule_id"}

	for _, labelName := range labelNamesToCheck {
		if labelValue, exists := metricValue[labelName]; exists && labelValue != "" {
			return labelValue
		}
	}

	// If we didn't find a check rule in the expected labels, check all labels
	// This helps us discover what label name is actually used
	for labelName, labelValue := range metricValue {
		// Skip known non-check-rule labels
		if labelName == "otel_metric_name" || labelName == "issue_id" || labelName == "__name__" {
			continue
		}
		// If this label value matches any of our check rules, use it
		if checkRuleNameSet[labelValue] || checkRuleIDSet[labelValue] {
			return labelValue
		}
	}

	return ""
}

// filterIssuesByCheckRules filters the Prometheus response to only include issues
// from the specified check rules. It looks for check rule information in metric labels.
// It matches against both check rule names and IDs since metric labels may contain either.
func (l *ListIssues) filterIssuesByCheckRules(data map[string]any, checkRuleNames []string, allCheckRules []CheckRule) map[string]any {
	checkRuleNameSet, checkRuleIDSet := buildCheckRuleLookupSets(checkRuleNames, allCheckRules)

	// Extract the Prometheus results data structure
	resultValue, dataValue, ok := extractPrometheusResults(data)
	if !ok {
		return data
	}

	// Filter results based on check rule names in metric labels
	filteredResults := make([]any, 0)
	for _, resultItem := range resultValue {
		resultMap, ok := resultItem.(map[string]any)
		if !ok {
			continue
		}

		metricValue, ok := extractMetricFromResult(resultMap)
		if !ok {
			continue
		}

		checkRuleValue := extractCheckRuleFromMetric(metricValue, checkRuleNameSet, checkRuleIDSet)

		// If we found a check rule value, check if it matches either the name or ID
		// If no check rule label is found, exclude the issue when filtering is active
		if checkRuleValue != "" {
			if checkRuleNameSet[checkRuleValue] || checkRuleIDSet[checkRuleValue] {
				filteredResults = append(filteredResults, resultItem)
			}
		}
	}

	// Update the data with filtered results
	dataValue["result"] = filteredResults
	data["data"] = dataValue

	return data
}

func (l *ListIssues) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListIssues) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListIssues) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListIssues) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListIssues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

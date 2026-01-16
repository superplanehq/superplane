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
			Name:     "checkRules",
			Label:    "Check Rules",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: false,
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
	// No validation needed since dataset is hardcoded to "default"
	return nil
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

	// Filter issues by check rules if check rules are specified
	if len(spec.CheckRules) > 0 {
		// Get check rules to map names to IDs
		checkRules, err := client.ListCheckRules()
		if err != nil {
			ctx.Logger.Warnf("Error fetching check rules for filtering: %v", err)
			// Continue without filtering if we can't fetch check rules
		} else {
			data = l.filterIssuesByCheckRules(data, spec.CheckRules, checkRules)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.issues.list",
		[]any{data},
	)
}

// filterIssuesByCheckRules filters the Prometheus response to only include issues
// from the specified check rules. It looks for check rule information in metric labels.
// It matches against both check rule names and IDs since metric labels may contain either.
func (l *ListIssues) filterIssuesByCheckRules(data map[string]any, checkRuleNames []string, allCheckRules []CheckRule) map[string]any {
	// Create sets for both names and IDs for efficient lookup
	checkRuleNameSet := make(map[string]bool, len(checkRuleNames))
	checkRuleIDSet := make(map[string]bool, len(checkRuleNames))
	
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

	// Extract the data structure - it can be either a struct or map
	var resultValue []any
	var dataValue map[string]any

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
	} else if dataMap, ok := data["data"].(map[string]any); ok {
		dataValue = dataMap
		resultValue, ok = dataValue["result"].([]any)
		if !ok {
			return data
		}
	} else {
		return data
	}

	// Filter results based on check rule names in metric labels
	filteredResults := make([]any, 0)
	for _, resultItem := range resultValue {
		resultMap, ok := resultItem.(map[string]any)
		if !ok {
			continue
		}

		var metricValue map[string]string
		if metricMap, ok := resultMap["metric"].(map[string]any); ok {
			// Convert map[string]any to map[string]string
			metricValue = make(map[string]string)
			for k, v := range metricMap {
				if strVal, ok := v.(string); ok {
					metricValue[k] = strVal
				}
			}
		} else if metricStrMap, ok := resultMap["metric"].(map[string]string); ok {
			metricValue = metricStrMap
		} else {
			continue
		}

		// Check various possible label names for check rule information
		// Common label names: check_rule, check_rule_name, check_rule_id, rule_name, check_name, check
		var checkRuleValue string
		labelNamesToCheck := []string{"check_rule", "check_rule_name", "check_rule_id", "rule_name", "check_name", "check", "check_id", "rule_id"}
		
		for _, labelName := range labelNamesToCheck {
			if labelValue, exists := metricValue[labelName]; exists && labelValue != "" {
				checkRuleValue = labelValue
				break
			}
		}

		// If we didn't find a check rule in the expected labels, check all labels
		// This helps us discover what label name is actually used
		if checkRuleValue == "" {
			// Check all metric labels for any value that matches our check rules
			for labelName, labelValue := range metricValue {
				// Skip known non-check-rule labels
				if labelName == "otel_metric_name" || labelName == "issue_id" || labelName == "__name__" {
					continue
				}
				// If this label value matches any of our check rules, use it
				if checkRuleNameSet[labelValue] || checkRuleIDSet[labelValue] {
					checkRuleValue = labelValue
					break
				}
			}
		}

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

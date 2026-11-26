package factories

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
)

// Public settings have a fixed schema. Decode over fresh defaults so presence,
// including false, zero, and empty lists, is independent of the installed graph.
func decodeFactoryConfigurationSettings(section string, values map[string]any, fields map[string]string, defaults, result any) error {
	encoded, err := json.Marshal(defaults)
	if err != nil {
		return err
	}
	var merged map[string]any
	if err := json.Unmarshal(encoded, &merged); err != nil {
		return err
	}
	for field, value := range values {
		key, supported := fields[field]
		if !supported {
			return configurationSettingError(section+"."+field, "This setting is not supported by this built-in automation.")
		}
		if value == nil {
			return configurationSettingError(section+"."+field, "Use an explicit value. Null is not supported.")
		}
		if _, err := json.Marshal(value); err != nil {
			return configurationSettingError(section+"."+field, "Use a finite value with the supported setting type.")
		}
		merged[key] = value
	}
	encoded, err = json.Marshal(merged)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, result); err != nil {
		var mismatch *json.UnmarshalTypeError
		if errors.As(err, &mismatch) {
			field := mismatch.Field
			for name, key := range fields {
				if key == field {
					field = name
					break
				}
			}
			return configurationSettingError(section+"."+field, "Use a value with the supported setting type.")
		}
		return err
	}
	return nil
}

func configurationSettingError(field, message string) error {
	return &factoryConfigurationSettingError{Field: field, Message: message}
}

func factoryConfigurationIntakeFields(source string) map[string]string {
	fields := map[string]string{"confidence_pct": "ConfidencePct", "default_owner_id": "DefaultOwnerID", "paused": "Paused"}
	switch source {
	case models.FactoryIntakeSourceGitHubIssues, models.FactoryIntakeSourceJiraIssues:
		maps.Copy(fields, map[string]string{"labels": "Labels", "label_filter_mode": "LabelFilterMode", "assignment": "Assignment", "new_issues": "NewIssues", "reopened_issues": "ReopenedIssues"})
		if source == models.FactoryIntakeSourceGitHubIssues {
			maps.Copy(fields, map[string]string{"authors_with_access": "AuthorsWithAccess", "superplane_label_added": "SuperplaneLabelAdded"})
		}
		if source == models.FactoryIntakeSourceJiraIssues {
			maps.Copy(fields, map[string]string{"jira_move_on_complete": "JiraMoveOnComplete", "jira_completion_column": "JiraCompletionColumn"})
		}
	case models.FactoryIntakeSourceSentryExceptions:
		maps.Copy(fields, map[string]string{"sentry_new_issues": "SentryNewIssues", "sentry_regressed_issues": "SentryRegressedIssues", "sentry_assigned_issues": "SentryAssignedIssues", "sentry_levels": "SentryLevels", "sentry_project_ids": "SentryProjectIDs"})
	case models.FactoryIntakeSourceProductiveTasks:
		maps.Copy(fields, map[string]string{"exclude_key_tasks": "ExcludeKeyTasks", "task_list_ids": "TaskListIDs"})
	case models.FactoryIntakeSourceDependabotAlerts:
		fields["dependabot_severities"] = "DependabotSeverities"
	case models.FactoryIntakeSourceDatadog:
		maps.Copy(fields, map[string]string{"datadog_triggered_alerts": "DatadogTriggeredAlerts", "datadog_retriggered_alerts": "DatadogRetriggeredAlerts", "datadog_environments": "DatadogEnvironments"})
	case models.FactoryIntakeSourceLinearIssues:
		maps.Copy(fields, map[string]string{"linear_project_ids": "LinearProjectIDs", "linear_include_issues_without_project": "LinearIncludeNoProject", "linear_labels": "LinearLabels"})
	}
	return fields
}

func parseFactoryConfigurationIntakeSettings(source string, section map[string]any) (intakeSettings, error) {
	defaults := intakeSettingsOrDefault(source, intakeSettings{})
	settings := defaults
	if err := decodeFactoryConfigurationSettings("intake", section, factoryConfigurationIntakeFields(source), defaults, &settings); err != nil {
		return settings, err
	}
	if settings.ConfidencePct < 0 || settings.ConfidencePct > 100 {
		return settings, configurationSettingError("intake.confidence_pct", "Use an integer from 0 through 100.")
	}
	if !slices.Contains([]string{intakeLabelFilterInclude, intakeLabelFilterExclude}, settings.LabelFilterMode) {
		return settings, configurationSettingError("intake.label_filter_mode", "Use include or exclude.")
	}
	if !slices.Contains([]string{intakeAssignmentAny, intakeAssignmentAssigned, intakeAssignmentUnassigned}, settings.Assignment) {
		return settings, configurationSettingError("intake.assignment", "Use any, assigned, or unassigned.")
	}
	for _, level := range settings.SentryLevels {
		if !slices.Contains(intakeSentryKnownLevels, level) {
			return settings, configurationSettingError("intake.sentry_levels", "Use fatal, error, warning, info, or debug.")
		}
	}
	for _, severity := range settings.DependabotSeverities {
		if !slices.Contains(intakeDependabotKnownSeverities, severity) {
			return settings, configurationSettingError("intake.dependabot_severities", "Use critical, high, medium, or low.")
		}
	}
	if source == models.FactoryIntakeSourceJiraIssues && !settings.NewIssues && !settings.ReopenedIssues {
		return settings, configurationSettingError("intake.new_issues", "Enable at least one Jira issue event.")
	}
	if source == models.FactoryIntakeSourceDatadog && !settings.DatadogTriggeredAlerts && !settings.DatadogRetriggeredAlerts {
		return settings, configurationSettingError("intake.datadog_triggered_alerts", "Enable at least one Datadog alert transition.")
	}
	if settings.Paused && !intakeSourceSupportsPause(source) {
		return settings, configurationSettingError("intake.paused", "Pause is not supported for this intake.")
	}
	if source == models.FactoryIntakeSourceSentryExceptions {
		if _, present := section["sentry_project_ids"]; present && len(settings.SentryProjectIDs) == 0 {
			return settings, configurationSettingError("intake.sentry_project_ids", "Select at least one Sentry project.")
		}
	}
	if source == models.FactoryIntakeSourceGitHubIssues {
		_, newIssues := section["new_issues"]
		_, labeled := section["superplane_label_added"]
		if !newIssues && !labeled {
			settings = deriveGitHubIssueListening(settings)
		}
		settings.listeningConfigured = true
	}
	return settings.normalized(), nil
}

func exportFactoryConfigurationIntakeSettings(source string, settings intakeSettings) map[string]any {
	encoded, _ := json.Marshal(settings)
	var values map[string]any
	_ = json.Unmarshal(encoded, &values)
	result := map[string]any{}
	for field, key := range factoryConfigurationIntakeFields(source) {
		value := values[key]
		if value == nil {
			value = []any{}
		}
		result[field] = value
	}
	return result
}

func factoryConfigurationPRFeedbackFields(source string) map[string]string {
	if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		return map[string]string{"check_names": "CheckNames", "maximum_attempts": "MaximumAttempts", "runner_integration_ids": "RunnerIntegrationIDs"}
	}
	return map[string]string{"mention": "Mention", "ignore_bots": "IgnoreBots", "address_all_bots": "AddressAllBots", "allowed_bots": "AllowedBots"}
}

func parseFactoryConfigurationPRFeedbackSettings(source string, section map[string]any) (prFeedbackSettings, error) {
	settings := defaultPRFeedbackSettings()
	if err := decodeFactoryConfigurationSettings("pr_feedback", section, factoryConfigurationPRFeedbackFields(source), settings, &settings); err != nil {
		return settings, err
	}
	if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		if settings.MaximumAttempts < prFeedbackMaximumAttemptsMin || settings.MaximumAttempts > prFeedbackMaximumAttemptsMax {
			return settings, configurationSettingError("pr_feedback.maximum_attempts", "Use an integer from 1 through 10.")
		}
		if _, present := section["check_names"]; present && len(settings.CheckNames) == 0 {
			return settings, configurationSettingError("pr_feedback.check_names", "Select at least one check name.")
		}
		for _, name := range settings.CheckNames {
			if strings.TrimSpace(name) == "" {
				return settings, configurationSettingError("pr_feedback.check_names", "Use nonempty check names.")
			}
		}
	}
	return settings.normalized(), nil
}

func exportFactoryConfigurationPRFeedbackSettings(source string, settings prFeedbackSettings) map[string]any {
	encoded, _ := json.Marshal(settings)
	var values map[string]any
	_ = json.Unmarshal(encoded, &values)
	result := map[string]any{}
	for field, key := range factoryConfigurationPRFeedbackFields(source) {
		value := values[key]
		if value == nil {
			value = []any{}
		}
		result[field] = value
	}
	return result
}

func factoryConfigurationTemplateFields(templateID string) map[string]string {
	switch templateID {
	case scheduledIntakeTemplateID:
		return map[string]string{"type": "type", "hour": "hour", "minute": "minute", "timezone": "timezone", "cron_expression": "cronExpression", "prompt": "prompt", "default_owner_id": "defaultOwnerId"}
	case lineImplementationTemplateID:
		return map[string]string{"draft_pull_requests": "draftPullRequests"}
	case "risk-score":
		return map[string]string{"enabled_checks": "enabledChecks", "risk_rules": "riskRules", "publish_pr_check": "publishPRCheck", "publish_results": "publishResults", "results_body": "resultsBody", "results_comment": "resultsComment"}
	case reviewRequestTemplateID:
		return map[string]string{"merge_confidence": "mergeConfidence", "fallback_reviewer": "fallbackReviewer", "publish_pr_check": "publishPRCheck", "publish_review_findings": "publishReviewFindings", "review_findings_body": "reviewFindingsBody", "review_findings_comment": "reviewFindingsComment"}
	}
	return nil
}

func applyFactoryConfigurationTemplateSettings(templateID string, section map[string]any, input *factoryTemplateInput) error {
	fields := factoryConfigurationTemplateFields(templateID)
	if input.installParams == nil {
		input.installParams = map[string]string{}
	}
	for field, value := range section {
		key, supported := fields[field]
		if !supported {
			return configurationSettingError("template."+field, "This setting is not supported by this built-in automation.")
		}
		if field == "merge_confidence" {
			continue
		}
		switch field {
		case "hour", "minute":
			encoded, _ := json.Marshal(value)
			var number int
			if value == nil || json.Unmarshal(encoded, &number) != nil {
				return configurationSettingError("template."+field, "Use an integer value.")
			}
			maximum := 23
			if field == "minute" {
				maximum = 59
			}
			if number < 0 || number > maximum {
				return configurationSettingError("template."+field, fmt.Sprintf("Use an integer from 0 through %d.", maximum))
			}
			input.installParams[key] = strconv.Itoa(number)
		case "enabled_checks":
			names, err := configurationStringList(value)
			if err != nil {
				return configurationSettingError("template."+field, err.Error())
			}
			for _, name := range names {
				if !slices.Contains(strings.Split(defaultMergeConfidenceChecks, ", "), name) {
					return configurationSettingError("template."+field, "Select a supported Merge confidence check.")
				}
			}
			input.installParams[key] = strings.Join(names, ", ")
			if len(names) == 0 {
				input.installParams[key] = "none"
			}
		case "draft_pull_requests":
			enabled, ok := value.(bool)
			if !ok {
				return configurationSettingError("template."+field, "Use a boolean value.")
			}
			input.draftPullRequests = &enabled
		case "publish_pr_check", "publish_results", "results_body", "results_comment", "publish_review_findings", "review_findings_body", "review_findings_comment":
			enabled, ok := value.(bool)
			if !ok {
				return configurationSettingError("template."+field, "Use a boolean value.")
			}
			setFactoryConfigurationTemplateAgentSetting(input, templateID, key, enabled)
		default:
			text, ok := value.(string)
			if !ok {
				return configurationSettingError("template."+field, "Use an explicit string value.")
			}
			if field == "fallback_reviewer" {
				setFactoryConfigurationTemplateAgentSetting(input, templateID, key, text)
				continue
			}
			input.installParams[key] = text
			if field == "risk_rules" {
				setFactoryConfigurationTemplateAgentSetting(input, templateID, "mergeConfidenceRiskRules", text)
			}
		}
	}
	if templateID == scheduledIntakeTemplateID {
		return validateFactoryConfigurationSchedule(section, input.installParams)
	}
	return nil
}

func setFactoryConfigurationTemplateAgentSetting(input *factoryTemplateInput, templateID, key string, value any) {
	id := "assess-risk"
	if templateID == reviewRequestTemplateID {
		id = reviewRequestAgentNodeID
	}
	if input.agentSettings == nil {
		input.agentSettings = map[string]*factoryAgentSettings{}
	}
	if input.agentSettings[id] == nil {
		input.agentSettings[id] = &factoryAgentSettings{configuration: map[string]any{}}
	}
	input.agentSettings[id].configuration[key] = value
}

func configurationStringList(value any) ([]string, error) {
	if strings, ok := value.([]string); ok {
		return strings, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("use a list of strings")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("use a list of strings")
		}
		result = append(result, text)
	}
	return result, nil
}

func exportFactoryConfigurationPublicSettings(version *models.CanvasVersion, templateID string, settings map[string]any) error {
	if strings.HasPrefix(templateID, "intake:") {
		source := strings.TrimPrefix(templateID, "intake:")
		spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
		graph := resolveIntakeGraph(source, spec)
		settings["intake"] = exportFactoryConfigurationIntakeSettings(source, intakeSettingsFromGraph(source, graph, spec))
		return nil
	}
	if templateID == prFeedbackDiscussionTemplateID || templateID == prFeedbackChecksTemplateID {
		source := models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion
		if templateID == prFeedbackChecksTemplateID {
			source = models.FactoryPRFeedbackHandlerSourcePullRequestChecks
		}
		spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
		values := prFeedbackSettingsFromGraph(resolvePRFeedbackGraph(spec), spec)
		settings["pr_feedback"] = exportFactoryConfigurationPRFeedbackSettings(source, values)
		return nil
	}
	section := map[string]any{}
	switch templateID {
	case scheduledIntakeTemplateID:
		for _, node := range version.Nodes {
			if node.ID == "schedule" {
				for field, key := range factoryConfigurationTemplateFields(templateID) {
					if value, ok := node.Configuration[key]; ok {
						section[field] = value
					}
				}
			}
			if node.ID == "scheduled-agent" {
				section["prompt"] = scheduledIntakePromptFromSteps(node.Configuration["steps"])
			}
			if node.ID == "create-task" {
				section["default_owner_id"] = configString(node.Configuration, "defaultOwnerId")
			}
		}
	case lineImplementationTemplateID:
		for _, node := range version.Nodes {
			if value, ok := factoryDraftPullRequests(node.ComponentName(), node.Configuration); ok {
				section["draft_pull_requests"] = value
				break
			}
		}
	case "risk-score", reviewRequestTemplateID:
		id := "assess-risk"
		if templateID == reviewRequestTemplateID {
			id = reviewRequestAgentNodeID
		}
		node := findIntakeNode(version.Nodes, id)
		if node == nil {
			return fmt.Errorf("the built-in agent is missing")
		}
		for field, key := range factoryConfigurationTemplateFields(templateID) {
			if value, ok := node.Configuration[key]; ok {
				section[field] = value
			}
		}
		if templateID == "risk-score" {
			section["enabled_checks"], section["risk_rules"] = factoryConfigurationMergeChecks(node.Configuration["steps"])
			if value, ok := node.Configuration["mergeConfidenceRiskRules"]; ok {
				section["risk_rules"] = value
			}
			// Defaults were implicit on older graphs.
			if _, ok := section["publish_results"]; !ok {
				section["publish_results"] = true
				if publication := findIntakeNode(version.Nodes, "publish-merge-confidence"); publication != nil {
					section["publish_results"] = publication.Configuration["enabled"] != false
				}
			}
			if _, ok := section["results_body"]; !ok {
				section["results_body"] = true
			}
			if _, ok := section["results_comment"]; !ok {
				section["results_comment"] = false
			}
		}
	}
	if len(section) > 0 {
		settings["template"] = section
	}
	return nil
}

func factoryConfigurationMergeChecks(raw any) ([]string, string) {
	encoded, _ := json.Marshal(raw)
	var steps []runner.AgentStep
	_ = json.Unmarshal(encoded, &steps)
	checks, _ := mergeconfidence.MergeConfidenceChecksFromSteps(steps)
	if checks == nil {
		checks = []string{}
	}
	rules := ""
	for _, step := range steps {
		if step.Prompt != nil {
			if text := mergeRiskRulesLine.FindString(*step.Prompt); text != "" {
				rules = strings.TrimSpace(text)
				break
			}
		}
	}
	return checks, rules
}

func validateFactoryConfigurationBuiltInSettingSections(templateID string, settings map[string]any) error {
	for field, raw := range settings {
		if field == "additional_instructions" && templateID == lineImplementationTemplateID {
			continue
		}
		supported := field == "agents" || field == "template" && len(factoryConfigurationTemplateFields(templateID)) > 0 || field == "intake" && strings.HasPrefix(templateID, "intake:") || field == "pr_feedback" && (templateID == prFeedbackChecksTemplateID || templateID == prFeedbackDiscussionTemplateID)
		if !supported {
			return configurationSettingError(field, "This setting is not supported by this built-in automation.")
		}
		if _, ok := raw.(map[string]any); !ok {
			return configurationSettingError(field, "Use a mapping of supported settings.")
		}
	}
	return nil
}

func validateFactoryConfigurationSchedule(section map[string]any, params map[string]string) error {
	scheduleType := params["type"]
	if _, present := section["type"]; !present {
		scheduleType = "days"
	}
	if scheduleType != "days" && scheduleType != "cron" {
		return configurationSettingError("template.type", "Use days or cron.")
	}
	if _, present := section["timezone"]; present {
		offset, err := strconv.ParseFloat(strings.TrimSpace(params["timezone"]), 64)
		if err != nil || math.IsNaN(offset) || math.IsInf(offset, 0) || offset < -12 || offset > 14 {
			return configurationSettingError("template.timezone", "Use a numeric UTC offset from -12 through 14.")
		}
	}
	if scheduleType == "days" {
		if _, present := section["cron_expression"]; present {
			return configurationSettingError("template.cron_expression", "Set type to cron before you specify a cron expression.")
		}
		return nil
	}
	for _, field := range []string{"hour", "minute"} {
		if _, present := section[field]; present {
			return configurationSettingError("template."+field, "Use the cron expression to set this schedule.")
		}
	}
	if _, err := scheduledIntakeSchedule(params); err != nil {
		return configurationSettingError("template.cron_expression", err.Error())
	}
	return nil
}

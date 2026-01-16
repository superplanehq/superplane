package dash0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueStatus struct{}

type OnIssueStatusConfiguration struct {
	MinutesInterval *int     `json:"minutesInterval"`
	CheckRules      []string `json:"checkRules,omitempty"`
}

type OnIssueStatusMetadata struct {
	NextTrigger        *string  `json:"nextTrigger"`
	ReferenceTime      *string  `json:"referenceTime"`                // Time when schedule was first set up
	LastCheck          *string  `json:"lastCheck,omitempty"`          // Time when the last check was performed
	LastDetectedChecks []string `json:"lastDetectedChecks,omitempty"` // List of check identifiers from the last event emission
}

func (t *OnIssueStatus) Name() string {
	return "dash0.onIssueStatus"
}

func (t *OnIssueStatus) Label() string {
	return "On Issue Status"
}

func (t *OnIssueStatus) Description() string {
	return "Periodically check Dash0 for issues and trigger when issues are detected"
}

func (t *OnIssueStatus) Icon() string {
	return "alert-triangle"
}

func (t *OnIssueStatus) Color() string {
	return "red"
}

func (t *OnIssueStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "minutesInterval",
			Label:       "Check every (minutes)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     intPtr(5),
			Description: "Number of minutes between checks (1-59)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(59),
				},
			},
		},
		{
			Name:     "checkRules",
			Label:    "Check Rules",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "check-rule",
					Multi: true,
				},
			},
			Description: "Select specific check rules to monitor. When disabled, all check rules will be monitored. Check rules will be fetched from your Dash0 account.",
		},
	}
}

func (t *OnIssueStatus) Setup(ctx core.TriggerContext) error {
	config := OnIssueStatusConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.MinutesInterval == nil {
		return fmt.Errorf("minutesInterval is required")
	}

	if *config.MinutesInterval < 1 || *config.MinutesInterval > 59 {
		return fmt.Errorf("minutesInterval must be between 1 and 59, got: %d", *config.MinutesInterval)
	}

	var metadata OnIssueStatusMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	now := time.Now()

	if metadata.ReferenceTime == nil {
		referenceTime := now.Format(time.RFC3339)
		metadata.ReferenceTime = &referenceTime
	}

	nextTrigger, err := t.nextTrigger(*config.MinutesInterval, now, metadata.ReferenceTime)
	if err != nil {
		return err
	}

	//
	// If the configuration didn't change, don't schedule a new action.
	//
	if metadata.NextTrigger != nil {
		currentTrigger, err := time.Parse(time.RFC3339, *metadata.NextTrigger)
		if err != nil {
			return fmt.Errorf("error parsing next trigger: %v", err)
		}

		if currentTrigger.Sub(*nextTrigger).Abs() < time.Second {
			return nil
		}
	}

	//
	// Always schedule the next and save the next trigger in the metadata.
	//
	err = ctx.Requests.ScheduleActionCall("checkIssues", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return fmt.Errorf("error scheduling action call: %w", err)
	}

	formatted := nextTrigger.Format(time.RFC3339)
	return ctx.Metadata.Set(OnIssueStatusMetadata{
		NextTrigger:   &formatted,
		ReferenceTime: metadata.ReferenceTime,
	})
}

func (t *OnIssueStatus) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "checkIssues",
			UserAccessible: false,
		},
	}
}

func (t *OnIssueStatus) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkIssues":
		return nil, t.checkIssues(ctx)
	}

	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnIssueStatus) checkIssues(ctx core.TriggerActionContext) error {
	config := OnIssueStatusConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.MinutesInterval == nil {
		return fmt.Errorf("minutesInterval is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating dash0 client: %w", err)
	}

	query := `{otel_metric_name="dash0.issue.status"} >= 1`
	dataset := "default"

	response, err := client.ExecutePrometheusInstantQuery(query, dataset)
	if err != nil {
		ctx.Logger.Warnf("Error executing Prometheus query: %v", err)
		// Continue to reschedule even if query fails
	} else {
		t.processQueryResults(ctx, response, query, dataset, config)
	}

	// Update last check time AFTER the check is complete and reschedule
	// This ensures lastCheck reflects when the check actually finished
	lastCheckTime := time.Now()
	return t.rescheduleCheck(ctx, config, &lastCheckTime)
}

func (t *OnIssueStatus) processQueryResults(ctx core.TriggerActionContext, response map[string]any, query, dataset string, config OnIssueStatusConfiguration) {
	dataValue := response["data"]
	dataMap := t.convertDataToMap(dataValue, ctx)
	if dataMap == nil {
		return
	}

	result, ok := dataMap["result"].([]interface{})
	if !ok {
		ctx.Logger.Warnf("Unexpected response format: result is not an array, got %T", dataMap["result"])
		return
	}

	// Get existing metadata to check for last detected checks
	var existingMetadata OnIssueStatusMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existingMetadata)
	if err != nil {
		existingMetadata = OnIssueStatusMetadata{}
	}

	// If no issues found, clear last detected checks so returning issues will be detected as new
	if len(result) == 0 {
		if len(existingMetadata.LastDetectedChecks) > 0 {
			existingMetadata.LastDetectedChecks = []string{}
			err = ctx.Metadata.Set(existingMetadata)
			if err != nil {
				ctx.Logger.Warnf("Failed to clear last detected checks: %v", err)
			}
			ctx.Logger.Infof("No issues found, cleared last detected checks")
		}
		return
	}

	// Filter results by selected check rules if any are specified
	// If checkRules is empty, monitor all check rules (no filtering)
	filteredResults := result
	if len(config.CheckRules) > 0 {
		filteredResults = t.filterResultsByCheckRules(result, config.CheckRules, ctx)
		if len(filteredResults) == 0 {
			// No matching check rules found, don't emit event
			ctx.Logger.Infof("No results match selected check rules, skipping event emission")
			return
		}
	}

	// Extract check identifiers from current results
	currentCheckIds := t.extractCheckIdentifiers(filteredResults)

	// Check if the set of checks has changed
	if t.hasChecksChanged(currentCheckIds, existingMetadata.LastDetectedChecks) {
		// Issues detected or changed - emit event
		payload := map[string]any{
			"query":   query,
			"dataset": dataset,
			"results": filteredResults,
			"count":   len(filteredResults),
		}

		err := ctx.Events.Emit("dash0.issue.detected", payload)
		if err != nil {
			ctx.Logger.Errorf("Error emitting event: %v", err)
			return
		}

		// Update metadata with current checks
		existingMetadata.LastDetectedChecks = currentCheckIds
		err = ctx.Metadata.Set(existingMetadata)
		if err != nil {
			ctx.Logger.Warnf("Failed to update last detected checks: %v", err)
		}

		ctx.Logger.Infof("Issues detected: %d issue(s) found", len(filteredResults))
	} else {
		// Same checks detected, skip event emission to avoid spam
		ctx.Logger.Infof("Same checks detected as last time, skipping event emission (checks: %v)", currentCheckIds)
	}
}

// extractCheckIdentifiers extracts unique check identifiers from Prometheus query results
// Uses check name from metric labels (dash0_check_name, check_rule_name, etc.)
func (t *OnIssueStatus) extractCheckIdentifiers(results []interface{}) []string {
	checkIds := make(map[string]bool)
	labelNames := []string{"dash0_check_name", "check_rule_name", "check_rule_id", "rule_name", "rule_id", "alertname", "alert_name"}

	for _, resultItem := range results {
		resultMap, ok := resultItem.(map[string]any)
		if !ok {
			continue
		}

		metric, ok := resultMap["metric"].(map[string]any)
		if !ok {
			continue
		}

		// Try each label name to find the check identifier
		for _, labelName := range labelNames {
			if labelValue, exists := metric[labelName]; exists {
				if labelStr, ok := labelValue.(string); ok && labelStr != "" {
					checkIds[labelStr] = true
					break // Use first found identifier
				}
			}
		}
	}

	// Convert map keys to slice
	ids := make([]string, 0, len(checkIds))
	for id := range checkIds {
		ids = append(ids, id)
	}

	return ids
}

// hasChecksChanged compares two sets of check identifiers and returns true if they differ
func (t *OnIssueStatus) hasChecksChanged(current, previous []string) bool {
	// If no previous checks, consider it changed (first detection)
	if len(previous) == 0 {
		return true
	}

	// If different lengths, they've changed
	if len(current) != len(previous) {
		return true
	}

	// Create a map of previous checks for O(1) lookup
	previousMap := make(map[string]bool, len(previous))
	for _, id := range previous {
		previousMap[id] = true
	}

	// Check if all current checks were in previous set
	for _, id := range current {
		if !previousMap[id] {
			return true // New check found
		}
	}

	// Same set of checks
	return false
}

func (t *OnIssueStatus) filterResultsByCheckRules(results []interface{}, checkRules []string, ctx core.TriggerActionContext) []interface{} {
	if len(checkRules) == 0 {
		return results
	}

	// Create a map for faster lookup
	checkRuleSet := make(map[string]bool, len(checkRules))
	for _, rule := range checkRules {
		checkRuleSet[rule] = true
	}

	var filtered []interface{}
	for _, resultItem := range results {
		resultMap, ok := resultItem.(map[string]any)
		if !ok {
			ctx.Logger.Warnf("Unexpected result item format, skipping: %T", resultItem)
			continue
		}

		metric, ok := resultMap["metric"].(map[string]any)
		if !ok {
			ctx.Logger.Warnf("Unexpected metric format, skipping: %T", resultMap["metric"])
			continue
		}

		// Check multiple common label names that might identify the check rule
		// Common labels: check_rule_id, check_rule_name, rule_id, rule_name, alertname, alert_name
		labelNames := []string{"check_rule_id", "check_rule_name", "rule_id", "rule_name", "alertname", "alert_name"}
		matched := false

		for _, labelName := range labelNames {
			if labelValue, exists := metric[labelName]; exists {
				if labelStr, ok := labelValue.(string); ok {
					if checkRuleSet[labelStr] {
						matched = true
						break
					}
				}
			}
		}

		if matched {
			filtered = append(filtered, resultItem)
		}
	}

	return filtered
}

func (t *OnIssueStatus) convertDataToMap(dataValue any, ctx core.TriggerActionContext) map[string]any {
	if dataMapValue, ok := dataValue.(map[string]any); ok {
		return dataMapValue
	}

	// If it's a struct, marshal and unmarshal it to convert to map
	jsonBytes, err := json.Marshal(dataValue)
	if err != nil {
		ctx.Logger.Warnf("Failed to marshal response data: %v", err)
		return nil
	}

	var dataMap map[string]any
	err = json.Unmarshal(jsonBytes, &dataMap)
	if err != nil {
		ctx.Logger.Warnf("Failed to unmarshal response data: %v", err)
		return nil
	}

	return dataMap
}

func (t *OnIssueStatus) rescheduleCheck(ctx core.TriggerActionContext, config OnIssueStatusConfiguration, lastCheckTime *time.Time) error {
	// Reschedule next check
	var existingMetadata OnIssueStatusMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existingMetadata)
	if err != nil {
		// Use current time as reference if metadata is invalid
		nowStr := time.Now().Format(time.RFC3339)
		existingMetadata = OnIssueStatusMetadata{
			ReferenceTime: &nowStr,
		}
	}

	nowUTC := time.Now()
	if lastCheckTime != nil {
		nowUTC = *lastCheckTime
	}

	nextTrigger, err := t.nextTrigger(*config.MinutesInterval, nowUTC, existingMetadata.ReferenceTime)
	if err != nil {
		return fmt.Errorf("error calculating next trigger: %w", err)
	}

	err = ctx.Requests.ScheduleActionCall("checkIssues", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return fmt.Errorf("error rescheduling action call: %w", err)
	}

	formatted := nextTrigger.Format(time.RFC3339)
	metadata := OnIssueStatusMetadata{
		NextTrigger:        &formatted,
		ReferenceTime:      existingMetadata.ReferenceTime,
		LastCheck:          existingMetadata.LastCheck,          // Preserve existing last check by default
		LastDetectedChecks: existingMetadata.LastDetectedChecks, // Preserve last detected checks
	}

	// Update last check time if provided (when checkIssues runs)
	if lastCheckTime != nil {
		lastCheckFormatted := lastCheckTime.Format(time.RFC3339)
		metadata.LastCheck = &lastCheckFormatted
	}

	return ctx.Metadata.Set(metadata)
}

func (t *OnIssueStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnIssueStatus) nextTrigger(interval int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 59 {
		return nil, fmt.Errorf("interval must be between 1 and 59 minutes, got: %d", interval)
	}

	nowInTZ := now

	var reference time.Time
	if referenceTime != nil {
		var err error
		reference, err = time.Parse(time.RFC3339, *referenceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference time: %w", err)
		}
		reference = reference.In(nowInTZ.Location())
	} else {
		reference = nowInTZ
	}

	minutesElapsed := int(nowInTZ.Sub(reference).Minutes())

	if minutesElapsed < 0 {
		minutesElapsed = 0
	}
	completedIntervals := minutesElapsed / interval

	nextTriggerMinutes := (completedIntervals + 1) * interval
	nextTrigger := reference.Add(time.Duration(nextTriggerMinutes) * time.Minute)

	if nextTrigger.Before(nowInTZ) || nextTrigger.Equal(nowInTZ) {
		nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Minute)
	}

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
}

func intPtr(v int) *int {
	return &v
}

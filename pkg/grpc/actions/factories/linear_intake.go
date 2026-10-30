package factories

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
)

func defaultLinearIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.AuthorsWithAccess = false
	settings.LinearTeamID = ""
	settings.LinearProjectIDs = []string{}
	settings.LinearIncludeNoProject = true
	settings.LinearLabels = []string{}
	return settings
}

// linearIntakeHasScope reports whether the settings select any issue. A team
// does. So do projects without a team, which is how older intakes listen.
func linearIntakeHasScope(settings intakeSettings) bool {
	return settings.LinearTeamID != "" || len(settings.LinearProjectIDs) > 0
}

// Trigger configuration keys of a team-scoped Linear intake. The trigger reads
// team. The other two belong to the intake: the trigger has no project filter
// that accepts issues without a project, so the intake filter node applies the
// project rule and the trigger itself listens to the whole team.
const (
	linearIntakeProjectsKey         = "intakeProjects"
	linearIntakeIncludeNoProjectKey = "includeIssuesWithoutProject"
)

// applyLinearTriggerConfiguration writes the Linear intake scope onto trigger
// configuration. A team scope registers one team webhook and leaves the
// trigger's own project filter empty. An intake without a team keeps the
// project list on the trigger.
func applyLinearTriggerConfiguration(configuration map[string]any, settings intakeSettings) {
	configuration["labels"] = linearLabelPredicates(settings.LinearLabels)
	configuration["actions"] = []any{"create", "update"}

	if settings.LinearTeamID == "" {
		configuration["projects"] = configurationAnyStrings(settings.LinearProjectIDs)
		delete(configuration, "team")
		delete(configuration, linearIntakeProjectsKey)
		delete(configuration, linearIntakeIncludeNoProjectKey)
		return
	}

	configuration["team"] = settings.LinearTeamID
	configuration["projects"] = []any{}
	configuration[linearIntakeProjectsKey] = configurationAnyStrings(settings.LinearProjectIDs)
	configuration[linearIntakeIncludeNoProjectKey] = settings.LinearIncludeNoProject
}

// linearScopeSettingsFromConfiguration reads the team, projects, and labels
// back from the trigger configuration that applyLinearTriggerConfiguration
// wrote.
func linearScopeSettingsFromConfiguration(configuration map[string]any, settings intakeSettings) intakeSettings {
	settings = linearTeamScopeFromConfiguration(configuration, settings)
	settings.LinearLabels = linearLabelsFromConfiguration(configuration["labels"])
	return settings
}

func linearTeamScopeFromConfiguration(configuration map[string]any, settings intakeSettings) intakeSettings {
	team, _ := configuration["team"].(string)
	settings.LinearTeamID = strings.TrimSpace(team)
	if settings.LinearTeamID == "" {
		settings.LinearProjectIDs = configurationStrings(configuration["projects"])
		return settings
	}

	settings.LinearProjectIDs = configurationStrings(configuration[linearIntakeProjectsKey])
	if value, ok := configuration[linearIntakeIncludeNoProjectKey]; ok {
		settings.LinearIncludeNoProject = metadataBool(value, settings.LinearIncludeNoProject)
	}
	return settings
}

// linearTriggerSettings returns the settings to write onto a generated
// trigger. Settings that carry no scope, such as a restore of the template
// defaults, keep the scope the trigger configuration already holds.
func linearTriggerSettings(settings intakeSettings, configuration map[string]any) intakeSettings {
	if settings.LinearTeamID != "" || len(settings.LinearProjectIDs) > 0 {
		return settings
	}
	held := linearTeamScopeFromConfiguration(configuration, defaultLinearIntakeSettings())
	settings.LinearTeamID = held.LinearTeamID
	settings.LinearProjectIDs = held.LinearProjectIDs
	settings.LinearIncludeNoProject = held.LinearIncludeNoProject
	return settings
}

// linearIssueScope is the issue scope that a trigger configuration describes.
func linearIssueScope(configuration map[string]any) linear.IssueScope {
	settings := linearScopeSettingsFromConfiguration(configuration, defaultLinearIntakeSettings())
	return linear.IssueScope{
		TeamID:           settings.LinearTeamID,
		ProjectIDs:       settings.LinearProjectIDs,
		IncludeNoProject: settings.LinearIncludeNoProject,
	}
}

func linearProjectIDsFromResource(resourceID string) []string {
	return normalizeLinearValues(strings.Split(resourceID, ","))
}

func normalizeLinearValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

// withLinearIntakeScope copies the intake scope onto the binding. Create
// resolves the binding from the integration and project list only. The first
// import reads the team, projects, and labels from that binding, so they have
// to be present before it runs.
func withLinearIntakeScope(binding *intakeBinding, settings intakeSettings) *intakeBinding {
	if binding == nil {
		return nil
	}
	if binding.Configuration == nil {
		binding.Configuration = map[string]any{}
	}
	applyLinearTriggerConfiguration(binding.Configuration, settings)
	return binding
}

func linearLabelPredicates(labels []string) []any {
	predicates := make([]any, 0, len(labels))
	for _, label := range labels {
		predicates = append(predicates, map[string]any{
			"type":  configuration.PredicateTypeEquals,
			"value": label,
		})
	}
	return predicates
}

func linearLabelsFromConfiguration(value any) []string {
	switch typed := value.(type) {
	case []any:
		labels := make([]string, 0, len(typed))
		for _, item := range typed {
			switch predicate := item.(type) {
			case map[string]any:
				label, _ := predicate["value"].(string)
				if label = strings.TrimSpace(label); label != "" {
					labels = append(labels, label)
				}
			case configuration.Predicate:
				if label := strings.TrimSpace(predicate.Value); label != "" {
					labels = append(labels, label)
				}
			}
		}
		return labels
	default:
		return nil
	}
}

func intakeLinearFilterExpression(settings intakeSettings) string {
	parts := []string{}
	if settings.LinearTeamID != "" {
		teamParts, ok := linearTeamFilterParts(settings)
		if !ok {
			return "true"
		}
		parts = append(parts, teamParts...)
	} else if len(settings.LinearProjectIDs) > 0 {
		encoded, err := json.Marshal(settings.LinearProjectIDs)
		if err != nil {
			return "true"
		}
		parts = append(parts, fmt.Sprintf(`(root().data.data.projectId ?? "") in %s`, encoded))
	}
	if len(settings.LinearLabels) > 0 {
		encoded, err := json.Marshal(settings.LinearLabels)
		if err != nil {
			return "true"
		}
		parts = append(parts, fmt.Sprintf(`any(root().data.data.labels, .name in %s)`, encoded))
	}
	if len(parts) == 0 {
		return "true"
	}
	return strings.Join(parts, " && ")
}

// linearTeamFilterParts builds the conditions of a team-scoped intake. The
// trigger passes every event of the team, so the filter does three things:
//   - skip completed and canceled issues;
//   - let an update through only when it moves the issue into the team or a
//     project, because every other edit would create a task again;
//   - apply the optional project rule.
func linearTeamFilterParts(settings intakeSettings) ([]string, bool) {
	closedStates, err := json.Marshal([]string{"completed", "canceled"})
	if err != nil {
		return nil, false
	}
	parts := []string{
		fmt.Sprintf(`(root().data.data.state?.type ?? "") not in %s`, closedStates),
		`(root().data.action == "create" || "teamId" in (root().data.updatedFrom ?? {}) || "projectId" in (root().data.updatedFrom ?? {}))`,
	}
	if len(settings.LinearProjectIDs) == 0 {
		return parts, true
	}

	projects, err := json.Marshal(settings.LinearProjectIDs)
	if err != nil {
		return nil, false
	}
	projectRule := fmt.Sprintf(`(root().data.data.projectId ?? "") in %s`, projects)
	if settings.LinearIncludeNoProject {
		projectRule = fmt.Sprintf(`(%s || (root().data.data.projectId ?? "") == "")`, projectRule)
	}
	return append(parts, projectRule), true
}

func linearIssueMatchesLabels(issue linear.Issue, labels []string) bool {
	if len(labels) == 0 {
		return true
	}
	for _, label := range issue.Labels {
		for _, wanted := range labels {
			if label.Name == wanted {
				return true
			}
		}
	}
	return false
}

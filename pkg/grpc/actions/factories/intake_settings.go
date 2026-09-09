package factories

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/proto"
)

const (
	intakeLabelFilterInclude = "include"
	intakeLabelFilterExclude = "exclude"

	intakeAssignmentAny        = "any"
	intakeAssignmentAssigned   = "assigned"
	intakeAssignmentUnassigned = "unassigned"

	intakeAssignedCondition   = "size(root().data.issue.assignees) > 0"
	intakeUnassignedCondition = "size(root().data.issue.assignees) == 0"

	intakeAuthorAccessCondition    = `root().data.issue.author_association in ["COLLABORATOR", "MEMBER", "OWNER"]`
	intakeAssignedToAgentCondition = `root().data.action != "assigned" || (root().data.issue.state == "open" && root().data.assignee.login == "superplaneagent")`
)

// intakeSettings is what a user can change about an intake without editing the
// canvas by hand. Filter fields are stored in, and read back from, the filter
// expression: the graph is what the workers run, so nothing is kept twice.
type intakeSettings struct {
	ConfidencePct     int
	Labels            []string
	LabelFilterMode   string
	Assignment        string
	AuthorsWithAccess bool
	NewIssues         bool
	AssignedToAgent   bool
}

func defaultIntakeSettings() intakeSettings {
	return intakeSettings{
		ConfidencePct:   DefaultIntakeConfidencePct,
		Labels:          []string{},
		LabelFilterMode: intakeLabelFilterInclude,
		Assignment:      intakeAssignmentAny,
		NewIssues:       true,
		// AuthorsWithAccess is off by default: false.
	}
}

func (s intakeSettings) normalized() intakeSettings {
	s.ConfidencePct = clampIntakeConfidence(s.ConfidencePct)

	if s.LabelFilterMode != intakeLabelFilterExclude {
		s.LabelFilterMode = intakeLabelFilterInclude
	}
	if s.Assignment != intakeAssignmentAssigned && s.Assignment != intakeAssignmentUnassigned {
		s.Assignment = intakeAssignmentAny
	}

	labels := make([]string, 0, len(s.Labels))
	for _, label := range s.Labels {
		label = strings.TrimSpace(label)
		if label != "" && !slices.Contains(labels, label) {
			labels = append(labels, label)
		}
	}
	s.Labels = labels

	return s
}

// intakeFilterExpressionFor builds the gate in front of the work order from
// the filters the source supports. An empty filter set is `true`, so every
// matching event still creates a work order.
func intakeFilterExpressionFor(source string, settings intakeSettings) string {
	settings = settings.normalized()
	if source != models.FactoryIntakeSourceGitHubIssues {
		return "true"
	}

	conditions := []string{}
	if len(settings.Labels) > 0 {
		if labels, err := json.Marshal(settings.Labels); err == nil {
			matches := fmt.Sprintf("root().data.issue.labels.exists(label, label.name in %s)", labels)
			if settings.LabelFilterMode == intakeLabelFilterExclude {
				matches = fmt.Sprintf("!(%s)", matches)
			}
			conditions = append(conditions, matches)
		}
	}

	switch settings.Assignment {
	case intakeAssignmentAssigned:
		conditions = append(conditions, intakeAssignedCondition)
	case intakeAssignmentUnassigned:
		conditions = append(conditions, intakeUnassignedCondition)
	}

	if settings.AuthorsWithAccess {
		conditions = append(conditions, intakeAuthorAccessCondition)
	}
	if settings.AssignedToAgent {
		conditions = append(conditions, intakeAssignedToAgentCondition)
	}

	if len(conditions) == 0 {
		return "true"
	}

	return strings.Join(conditions, " && ")
}

func intakeTriggerActionsFor(settings intakeSettings) []any {
	actions := []any{}
	if settings.NewIssues {
		actions = append(actions, "opened", "reopened")
	}
	if settings.AssignedToAgent {
		actions = append(actions, "assigned")
	}
	return actions
}

func intakeSettingsChangeTrigger(current, updated intakeSettings) bool {
	return current.NewIssues != updated.NewIssues || current.AssignedToAgent != updated.AssignedToAgent
}

func intakeSettingsChangeFilters(current, updated intakeSettings) bool {
	if current.AssignedToAgent != updated.AssignedToAgent {
		return true
	}
	if current.LabelFilterMode != updated.LabelFilterMode {
		return true
	}
	if current.Assignment != updated.Assignment {
		return true
	}
	if current.AuthorsWithAccess != updated.AuthorsWithAccess {
		return true
	}
	if len(current.Labels) != len(updated.Labels) {
		return true
	}
	for i := range current.Labels {
		if current.Labels[i] != updated.Labels[i] {
			return true
		}
	}
	return false
}

var intakeLabelsPattern = regexp.MustCompile(`(!\()?root\(\)\.data\.issue\.labels\.exists\(label, label\.name in (\[[^\]]*\])\)`)

// intakeSettingsFromGraph reads the settings back out of the filter
// expression. A hand-edited expression that no longer matches reports defaults
// rather than a wrong value.
func intakeSettingsFromGraph(graph intakeGraph, spec models.LiveCanvasSpec) intakeSettings {
	settings := defaultIntakeSettings()
	settings.ConfidencePct = graph.ConfidencePct

	trigger := findIntakeNode(spec.Nodes, graph.TriggerNodeID)
	if trigger != nil {
		actions := configurationStrings(trigger.Configuration["actions"])
		settings.NewIssues = slices.Contains(actions, "opened") || slices.Contains(actions, "reopened")
		settings.AssignedToAgent = slices.Contains(actions, "assigned")
	}

	filter := findIntakeNode(spec.Nodes, graph.FilterNodeID)
	if filter == nil {
		return settings
	}

	expression, _ := filter.Configuration["expression"].(string)
	if expression == "" {
		return settings
	}

	if match := intakeLabelsPattern.FindStringSubmatch(expression); match != nil {
		var labels []string
		if err := json.Unmarshal([]byte(match[2]), &labels); err == nil {
			settings.Labels = labels
			if match[1] != "" {
				settings.LabelFilterMode = intakeLabelFilterExclude
			}
		}
	}

	switch {
	case strings.Contains(expression, intakeUnassignedCondition):
		settings.Assignment = intakeAssignmentUnassigned
	case strings.Contains(expression, intakeAssignedCondition):
		settings.Assignment = intakeAssignmentAssigned
	}

	settings.AuthorsWithAccess = strings.Contains(expression, intakeAuthorAccessCondition)

	return settings.normalized()
}

func serializeIntakeSettings(settings intakeSettings) *pb.FactoryIntake_Settings {
	return &pb.FactoryIntake_Settings{
		ConfidencePct:     int32(settings.ConfidencePct),
		Labels:            settings.Labels,
		LabelFilterMode:   serializeIntakeLabelFilterMode(settings.LabelFilterMode),
		Assignment:        serializeIntakeAssignment(settings.Assignment),
		AuthorsWithAccess: settings.AuthorsWithAccess,
		NewIssues:         proto.Bool(settings.NewIssues),
		AssignedToAgent:   proto.Bool(settings.AssignedToAgent),
	}
}

// parseIntakeSettings merges a request over what the graph already says, so a
// caller that leaves an enum unspecified does not reset it.
func parseIntakeSettings(current intakeSettings, requested *pb.FactoryIntake_Settings) intakeSettings {
	if requested == nil {
		return current
	}

	updated := current
	updated.ConfidencePct = int(requested.GetConfidencePct())
	updated.Labels = requested.GetLabels()

	if mode := requested.GetLabelFilterMode(); mode != pb.FactoryIntake_Settings_LABEL_FILTER_MODE_UNSPECIFIED {
		updated.LabelFilterMode = parseIntakeLabelFilterMode(mode)
	}
	if assignment := requested.GetAssignment(); assignment != pb.FactoryIntake_Settings_ASSIGNMENT_UNSPECIFIED {
		updated.Assignment = parseIntakeAssignment(assignment)
	}
	updated.AuthorsWithAccess = requested.GetAuthorsWithAccess()
	if requested.NewIssues != nil {
		updated.NewIssues = requested.GetNewIssues()
	}
	if requested.AssignedToAgent != nil {
		updated.AssignedToAgent = requested.GetAssignedToAgent()
	}

	return updated.normalized()
}

func configurationStrings(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func serializeIntakeLabelFilterMode(mode string) pb.FactoryIntake_Settings_LabelFilterMode {
	if mode == intakeLabelFilterExclude {
		return pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE
	}
	return pb.FactoryIntake_Settings_LABEL_FILTER_MODE_INCLUDE
}

func parseIntakeLabelFilterMode(mode pb.FactoryIntake_Settings_LabelFilterMode) string {
	if mode == pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE {
		return intakeLabelFilterExclude
	}
	return intakeLabelFilterInclude
}

func serializeIntakeAssignment(assignment string) pb.FactoryIntake_Settings_Assignment {
	switch assignment {
	case intakeAssignmentAssigned:
		return pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED
	case intakeAssignmentUnassigned:
		return pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED
	default:
		return pb.FactoryIntake_Settings_ASSIGNMENT_ANY
	}
}

func parseIntakeAssignment(assignment pb.FactoryIntake_Settings_Assignment) string {
	switch assignment {
	case pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED:
		return intakeAssignmentAssigned
	case pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED:
		return intakeAssignmentUnassigned
	default:
		return intakeAssignmentAny
	}
}

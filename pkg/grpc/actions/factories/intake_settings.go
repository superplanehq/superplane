package factories

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

const (
	intakeLabelFilterInclude = "include"
	intakeLabelFilterExclude = "exclude"

	intakeAssignmentAny        = "any"
	intakeAssignmentAssigned   = "assigned"
	intakeAssignmentUnassigned = "unassigned"

	intakeAssignedCondition   = "size(root().data.issue.assignees) > 0"
	intakeUnassignedCondition = "size(root().data.issue.assignees) == 0"
)

// intakeSettings is what a user can change about an intake without editing the
// canvas by hand. Every field is stored in, and read back from, the threshold
// expression: the graph is what the workers run, so nothing is kept twice.
type intakeSettings struct {
	ConfidencePct   int
	Labels          []string
	LabelFilterMode string
	Assignment      string
}

func defaultIntakeSettings() intakeSettings {
	return intakeSettings{
		ConfidencePct:   DefaultIntakeConfidencePct,
		Labels:          []string{},
		LabelFilterMode: intakeLabelFilterInclude,
		Assignment:      intakeAssignmentAny,
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

// intakeThresholdExpressionFor builds the gate in front of the work order: the
// score, plus the filters the source supports.
func intakeThresholdExpressionFor(source string, settings intakeSettings) string {
	settings = settings.normalized()
	conditions := []string{intakeThresholdExpression(settings.ConfidencePct)}

	// Only GitHub issues carry labels and assignees in their payload.
	if source != models.FactoryIntakeSourceGitHubIssues {
		return conditions[0]
	}

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

	return strings.Join(conditions, " && ")
}

var intakeLabelsPattern = regexp.MustCompile(`(!\()?root\(\)\.data\.issue\.labels\.exists\(label, label\.name in (\[[^\]]*\])\)`)

// intakeSettingsFromGraph reads the settings back out of the threshold
// expression. A hand-edited expression that no longer matches reports defaults
// rather than a wrong value.
func intakeSettingsFromGraph(graph intakeGraph, spec models.LiveCanvasSpec) intakeSettings {
	settings := defaultIntakeSettings()
	settings.ConfidencePct = graph.ConfidencePct

	threshold := findIntakeNode(spec.Nodes, graph.ThresholdNodeID)
	if threshold == nil {
		return settings
	}

	expression, _ := threshold.Configuration["expression"].(string)
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

	return settings.normalized()
}

func serializeIntakeSettings(settings intakeSettings) *pb.FactoryIntake_Settings {
	return &pb.FactoryIntake_Settings{
		ConfidencePct:   int32(settings.ConfidencePct),
		Labels:          settings.Labels,
		LabelFilterMode: serializeIntakeLabelFilterMode(settings.LabelFilterMode),
		Assignment:      serializeIntakeAssignment(settings.Assignment),
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

	return updated.normalized()
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

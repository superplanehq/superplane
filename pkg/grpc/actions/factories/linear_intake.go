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
	settings.LinearProjectIDs = []string{}
	settings.LinearLabels = []string{}
	return settings
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
	if len(settings.LinearProjectIDs) > 0 {
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

package runner

import (
	"encoding/json"
	"github.com/google/uuid"
	"strings"
)

const EnvSuperplaneMergeConfidenceOrderID = "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID"

func parseMergeConfidenceCheckLabels(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil
	}
	return labels
}

func parseMergeConfidenceChecksClaim(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	enabled := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		enabled = append(enabled, name)
	}
	return enabled
}

func environmentValue(environment []TaskEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name != name {
			continue
		}
		return strings.TrimSpace(item.Value), true
	}
	return "", false
}

func withoutEnvironmentName(environment []TaskEnvironmentVariable, name string) []TaskEnvironmentVariable {
	filtered := make([]TaskEnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		if item.Name == name {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// MergeConfidenceScope is the check list for one merge confidence agent run.
// EnabledChecks is the list from the agent prompt. The server reports a check
// only when its name is in this list.
type MergeConfidenceScope struct {
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	WorkOrderID     uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	EnabledChecks   []string
	// CheckLabels maps a check id to the step name. Built-in checks use their
	// own names when this map has no entry.
	CheckLabels map[string]string
	// Repository is the pull request repository from the run trigger. A task
	// can have one pull request in each of several repositories, and each
	// run scores only its own. Empty when the trigger names no repository.
	Repository string
}

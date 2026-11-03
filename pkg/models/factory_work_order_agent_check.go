package models

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/models/factory"
)

const (
	MaxAgentChecks            = 20
	MaxAgentCheckReasonLength = 1000
)

var agentCheckLevels = []string{
	FactoryWorkOrderCheckLevelPositive,
	FactoryWorkOrderCheckLevelCaution,
	FactoryWorkOrderCheckLevelCritical,
}

// AgentCheckInput is one check that an agent or a person reports.
type AgentCheckInput struct {
	Key      string
	Name     string
	Score    *float64
	MaxScore *float64
	Level    string
	Reason   string
}

// AgentCheckParams validates checks for ReportAgentChecks. Errors name the
// input field, so callers can return them as they are.
func AgentCheckParams(inputs []AgentCheckInput, run *factory.RunRef) ([]FactoryWorkOrderCheckParams, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("checks is required")
	}
	if len(inputs) > MaxAgentChecks {
		return nil, fmt.Errorf("checks must have at most %d items", MaxAgentChecks)
	}
	params := make([]FactoryWorkOrderCheckParams, 0, len(inputs))
	seen := make([]string, 0, len(inputs))
	for index, input := range inputs {
		label := fmt.Sprintf("checks[%d]", index)
		key := strings.TrimSpace(input.Key)
		if slices.Contains(seen, key) {
			return nil, fmt.Errorf("%s.key repeats an earlier key", label)
		}
		seen = append(seen, key)
		if input.Score == nil || input.MaxScore == nil {
			return nil, fmt.Errorf("%s.score and %s.max_score are required", label, label)
		}
		if *input.Score > *input.MaxScore {
			return nil, fmt.Errorf("%s.score must be from 0 to max_score", label)
		}
		level := strings.ToLower(strings.TrimSpace(input.Level))
		if !slices.Contains(agentCheckLevels, level) {
			return nil, fmt.Errorf("%s.level must be positive, caution, or critical", label)
		}
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			return nil, fmt.Errorf("%s.reason is required", label)
		}
		if utf8.RuneCountInString(reason) > MaxAgentCheckReasonLength {
			return nil, fmt.Errorf("%s.reason must be at most %d characters", label, MaxAgentCheckReasonLength)
		}
		params = append(params, FactoryWorkOrderCheckParams{
			Key:      key,
			Name:     input.Name,
			Score:    *input.Score,
			MaxScore: *input.MaxScore,
			Level:    level,
			Summary:  reason,
			Run:      run,
		})
	}
	return params, nil
}

// AgentCheckKeys trims and deduplicates keys for RemoveAgentChecks.
func AgentCheckKeys(raw []string) ([]string, error) {
	keys := []string{}
	for _, value := range raw {
		key := strings.TrimSpace(value)
		if key != "" && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("keys is required")
	}
	if len(keys) > MaxAgentChecks {
		return nil, fmt.Errorf("keys must have at most %d items", MaxAgentChecks)
	}
	return keys, nil
}

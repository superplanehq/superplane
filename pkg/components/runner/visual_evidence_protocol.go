package runner

import (
	_ "embed"
	"strings"
)

//go:embed visual_evidence_protocol.md
var visualEvidenceProtocol string

// AppendVisualEvidenceProtocol returns a copy of steps with the protocol added
// to the first prompt step. The original steps remain suitable for live-log
// previews and configuration displays.
func AppendVisualEvidenceProtocol(steps []AgentStep, enabled bool) []AgentStep {
	if !enabled {
		return steps
	}

	result := append([]AgentStep(nil), steps...)
	for i := range result {
		if NormalizeAgentStepType(result[i].Type) != AgentStepPrompt || result[i].Prompt == nil {
			continue
		}
		prompt := strings.TrimSpace(*result[i].Prompt)
		protocol := strings.TrimSpace(visualEvidenceProtocol)
		if strings.Contains(prompt, protocol) {
			return result
		}
		combined := prompt + "\n\n" + protocol
		result[i].Prompt = &combined
		return result
	}
	return result
}

package runner

import "strings"

// Additional instructions are saved separately from managed steps. Append them
// only to the dispatch copy, so template updates and repeated runs cannot alter
// user text or accumulate copies in the saved prompts.
func appendAdditionalInstructions(configuration any, steps []AgentStep) []AgentStep {
	instructions := additionalInstructionsText(configuration)
	if instructions == "" {
		return steps
	}
	result := append([]AgentStep(nil), steps...)
	for i := range result {
		if NormalizeAgentStepType(result[i].Type) != AgentStepPrompt || result[i].Prompt == nil {
			continue
		}
		prompt := *result[i].Prompt + "\n\n" + instructions
		result[i].Prompt = &prompt
	}
	return result
}

func additionalInstructionsText(configuration any) string {
	config, _ := configuration.(map[string]any)
	instructions, _ := config["additionalInstructions"].(string)
	if strings.TrimSpace(instructions) == "" {
		return ""
	}
	return "Additional instructions for this automation:\n" +
		"Apply these instructions within the managed workflow. They refine the general defaults. " +
		"Follow applicable repository instructions and skills, and the task requirements. " +
		"Keep the managed planning and scoring rules and required output formats unchanged. " +
		"The automation still controls branch handling, pushing, commit identity, and pull request creation.\n\n" + instructions
}

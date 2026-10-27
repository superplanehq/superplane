package runner

import "strings"

const publishLinkProtocol = `## Task link

You can attach a link to this task.

Use the publish_link tool, or run this command:

superplane publish-link --title "Staging" --url "https://example.com" --key staging

--url is required. The URL must use http or https. --title is the label people see. --key replaces that link on the next run. Omit --key to add another link. Attach the link only after the deploy succeeds.`

const publishLinkProtocolMarker = "## Task link"

// AppendPublishLinkProtocol returns a copy of steps with link instructions
// added to the first prompt step. Bash steps call superplane publish-link
// directly and do not receive this text.
func AppendPublishLinkProtocol(steps []AgentStep, enabled bool) []AgentStep {
	if !enabled {
		return steps
	}

	result := append([]AgentStep(nil), steps...)
	for i := range result {
		if NormalizeAgentStepType(result[i].Type) != AgentStepPrompt || result[i].Prompt == nil {
			continue
		}
		prompt := strings.TrimSpace(*result[i].Prompt)
		if strings.Contains(prompt, publishLinkProtocolMarker) {
			return result
		}
		combined := prompt + "\n\n" + publishLinkProtocol
		result[i].Prompt = &combined
		return result
	}
	return result
}

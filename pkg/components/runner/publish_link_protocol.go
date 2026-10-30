package runner

import "strings"

const publishLinkProtocol = `## Task link

You can attach a link, a markdown note, or a file to this task.

Use the publish_link tool, or run one of these commands:

superplane task artifact add-link --url "https://example.com" --name "Staging" --key staging

superplane task artifact add-markdown --file ./notes.md --name "Preview login" --key preview-login

superplane task artifact add-file --file "$SUPERPLANE_TASK_DIR/evidence/login.png" --name "Login"

--name is the label people see. --key replaces a link or a markdown note on the next run. Omit --key to add another artifact. Attach a link only after the deploy succeeds. add-markdown reads the note from --file. add-file uploads a PNG, JPEG, WebP, WebM, or MP4 file from the task evidence directory. Call inspect_screenshot before add-file for a screenshot. That tool returns the image. A video accepts --poster. For a UI change, inspect the screenshot and use upload_artifact.`

const publishLinkProtocolMarker = "## Task link"

// AppendPublishLinkProtocol returns a copy of steps with link instructions
// added to the first prompt step. Bash steps call superplane task artifact
// add-link, add-markdown, or add-file directly and do not receive this text.
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

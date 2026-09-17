package sentry

import (
	"encoding/json"
	"fmt"
)

// IssueDescription renders a Sentry issue as a task body: the permalink,
// then a fenced JSON block of the delivered payload.
func IssueDescription(issue any) string {
	issueMap, ok := issue.(map[string]any)
	if !ok {
		return ""
	}

	permalink, _ := issueMap["permalink"].(string)

	encoded, err := json.MarshalIndent(issueMap, "", "  ")
	if err != nil {
		return permalink
	}

	block := fmt.Sprintf("```json\n%s\n```", encoded)
	if permalink == "" {
		return block
	}

	return permalink + "\n\n" + block
}

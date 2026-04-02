package jira

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}

//go:embed example_output_change_jira_ticket_status.json
var exampleOutputChangeJiraTicketStatusBytes []byte

var exampleOutputChangeJiraTicketStatusOnce sync.Once
var exampleOutputChangeJiraTicketStatus map[string]any

func (t *ChangeJiraTicketStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputChangeJiraTicketStatusOnce, exampleOutputChangeJiraTicketStatusBytes, &exampleOutputChangeJiraTicketStatus)
}

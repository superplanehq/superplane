package factories

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

// BuiltInAutomation is one resettable factory automation.
type BuiltInAutomation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// The ids match the template id stored when an automation is reset.
var builtInAutomations = []BuiltInAutomation{
	{ID: models.FactoryAppTemplateBacklogID, Name: "Task analysis"},
	{ID: scheduledIntakeTemplateID, Name: "Scheduled intake"},
	{ID: "line-implementation", Name: "Implement"},
	{ID: "issue-intake", Name: "Issue label intake"},
	{ID: "pr-closure", Name: "Pull request closure"},
	{ID: "risk-score", Name: "Merge confidence"},
	{ID: "visual-evidence", Name: "Visual Evidence"},
	{ID: prFeedbackDiscussionTemplateID, Name: "Pull request discussion"},
	{ID: prFeedbackChecksTemplateID, Name: "Pull request checks"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceGitHubIssues), Name: "GitHub issues"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceDependabotAlerts), Name: "Dependabot alerts"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceJiraIssues), Name: "Jira issues"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceSentryExceptions), Name: "Sentry exceptions"},
	{ID: intakeTemplateID(models.FactoryIntakeSourcePagerDutyIncidents), Name: "PagerDuty incidents"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceProductiveTasks), Name: "Productive tasks"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceDatadog), Name: "Datadog errors"},
	{ID: intakeTemplateID(models.FactoryIntakeSourceLinearIssues), Name: "Linear issues"},
}

func intakeTemplateID(source string) string {
	return "intake:" + source
}

// ListBuiltInAutomations returns the automations an installation admin can reset.
func ListBuiltInAutomations() []BuiltInAutomation {
	items := slices.Clone(builtInAutomations)
	slices.SortFunc(items, func(left, right BuiltInAutomation) int {
		return strings.Compare(left.Name, right.Name)
	})
	return items
}

func lookupBuiltInAutomation(id string) (BuiltInAutomation, bool) {
	for _, item := range builtInAutomations {
		if item.ID == id {
			return item, true
		}
	}
	return BuiltInAutomation{}, false
}

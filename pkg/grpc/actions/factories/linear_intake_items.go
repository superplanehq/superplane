package factories

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func newLinearIntakeItemSource(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error) {
	scope := linearIssueScope(trigger.Configuration)
	if scope.Validate() != nil {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeLinearClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &linearIntakeItemSource{
		linear: client,
		scope:  scope,
		labels: linearLabelsFromConfiguration(trigger.Configuration["labels"]),
	}, nil
}

func (s *linearIntakeItemSource) Search(_ context.Context, query string, limit int) ([]IntakeItem, error) {
	issues, err := s.linear.SearchIssues(s.scope, query, limit)
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(issues))
	for _, result := range issues {
		reason := s.issueDisabledReason(result.Issue)
		if reason != "" && !result.ExplicitIdentifier {
			continue
		}
		item := linearIssueItem(result.Issue)
		item.DisabledReason = reason
		items = append(items, item)
		if limit > 0 && len(items) >= limit {
			return items, nil
		}
	}
	return items, nil
}

func (s *linearIntakeItemSource) issueDisabledReason(issue linear.Issue) string {
	reasons := []string{}
	if reason := s.scopeDisabledReason(issue); reason != "" {
		reasons = append(reasons, reason)
	}
	if issue.State != nil && (issue.State.Type == "completed" || issue.State.Type == "canceled") {
		reasons = append(reasons, "This issue is closed or canceled.")
	}
	if !linearIssueMatchesLabels(issue, s.labels) {
		reasons = append(reasons, "This issue has no matching intake labels.")
	}
	return strings.Join(reasons, " ")
}

func (s *linearIntakeItemSource) scopeDisabledReason(issue linear.Issue) string {
	if s.scope.Contains(issue) {
		return ""
	}
	if s.scope.TeamID != "" && (issue.Team == nil || issue.Team.ID != s.scope.TeamID) {
		return "This issue is not in the selected team."
	}
	return "This issue is not in the selected projects."
}

func (s *linearIntakeItemSource) IssueFiles(ctx context.Context, issueID, description string) ([]linear.IssueFile, []linear.IssueLink, error) {
	return s.linear.IssueFiles(ctx, issueID, description)
}

func (s *linearIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	issueID := strings.TrimSpace(id)
	if issueID == "" {
		return nil, errIntakeItemNotFound
	}

	issue, err := s.linear.GetIssue(issueID)
	if err != nil || issue == nil || s.issueDisabledReason(*issue) != "" {
		return nil, errIntakeItemNotFound
	}

	item := linearIssueItem(*issue)
	return &item, nil
}

func (s *linearIntakeItemSource) AvailabilityScope() string {
	return fmt.Sprintf(
		"linear:%s:%s:%t",
		s.scope.TeamID,
		strings.Join(s.scope.ProjectIDs, ","),
		s.scope.IncludeNoProject,
	)
}

func (s *linearIntakeItemSource) ItemIDFromOriginURL(rawURL string) (string, bool) {
	ref, ok := linear.IssueRefFromURL(rawURL)
	if !ok {
		return "", false
	}
	return ref.Identifier, true
}

func (s *linearIntakeItemSource) IsItemAvailable(_ context.Context, id string) (bool, error) {
	issue, err := s.linear.GetIssue(strings.TrimSpace(id))
	if err != nil || issue == nil || !s.scope.Contains(*issue) {
		return false, nil
	}
	if issue.State != nil && (issue.State.Type == "completed" || issue.State.Type == "canceled") {
		return false, nil
	}
	return true, nil
}

func linearIssueItem(issue linear.Issue) IntakeItem {
	return IntakeItem{
		ID:    issue.ID,
		Key:   issue.Identifier,
		Title: issue.Title,
		Body:  issue.Description,
		URL:   issue.URL,
	}
}

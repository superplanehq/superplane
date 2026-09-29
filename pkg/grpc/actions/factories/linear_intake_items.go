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
	projectIDs := configurationStrings(trigger.Configuration["projects"])
	if len(projectIDs) == 0 {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeLinearClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &linearIntakeItemSource{
		linear:     client,
		projectIDs: projectIDs,
		labels:     linearLabelsFromConfiguration(trigger.Configuration["labels"]),
	}, nil
}

func (s *linearIntakeItemSource) Search(_ context.Context, query string, limit int) ([]IntakeItem, error) {
	issues, err := s.linear.SearchOpenProjectIssues(s.projectIDs, query, limit)
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(issues))
	for _, issue := range issues {
		if !linearIssueMatchesLabels(issue, s.labels) || !s.ownsIssue(issue) {
			continue
		}
		items = append(items, linearIssueItem(issue))
	}
	return items, nil
}

func (s *linearIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	issueID := strings.TrimSpace(id)
	if issueID == "" {
		return nil, errIntakeItemNotFound
	}

	issue, err := s.linear.GetIssue(issueID)
	if err != nil || issue == nil || !s.ownsIssue(*issue) {
		return nil, errIntakeItemNotFound
	}

	item := linearIssueItem(*issue)
	return &item, nil
}

func (s *linearIntakeItemSource) AvailabilityScope() string {
	return "linear:" + strings.Join(s.projectIDs, ",")
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
	if err != nil || issue == nil || !s.ownsIssue(*issue) {
		return false, nil
	}
	if issue.State != nil && (issue.State.Type == "completed" || issue.State.Type == "canceled") {
		return false, nil
	}
	return true, nil
}

func (s *linearIntakeItemSource) ownsIssue(issue linear.Issue) bool {
	if issue.Project == nil {
		return false
	}
	for _, projectID := range s.projectIDs {
		if issue.Project.ID == projectID {
			return true
		}
	}
	return false
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

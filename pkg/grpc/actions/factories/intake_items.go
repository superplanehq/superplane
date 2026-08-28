package factories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	defaultLatestIntakeItems = 5
	defaultSearchIntakeItems = 10
	maxIntakeItems           = 50
)

var (
	errIntakeNotConnected      = errors.New("intake is not connected")
	errIntakeItemNotFound      = errors.New("intake item not found")
	errIntakeSearchUnsupported = errors.New("this intake cannot search items yet")
	intakeItemSourceByTrigger  = map[string]intakeItemSourceBuilder{}
)

type intakeItemSourceBuilder func(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error)

// registerIntakeItemSource binds live search and import to an intake trigger
// component. Unknown triggers return unsupportedIntakeItemSource.
func registerIntakeItemSource(triggerComponent string, builder intakeItemSourceBuilder) {
	intakeItemSourceByTrigger[triggerComponent] = builder
}

func init() {
	registerIntakeItemSource("github.onIssue", newGitHubIntakeItemSource)
}

type gitHubIntakeItemSource struct {
	github     *common.Client
	repository string
}

type unsupportedIntakeItemSource struct{}

func newLiveIntakeItemSource(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	intake *models.FactoryIntake,
) (intakeItemSource, error) {
	trigger, integration, err := resolveLiveIntakeTrigger(tx, intake)
	if err != nil {
		return nil, err
	}

	builder := intakeItemSourceByTrigger[trigger.ComponentName()]
	if builder == nil {
		return unsupportedIntakeItemSource{}, nil
	}

	return builder(ctx, deps, tx, trigger, integration)
}

func newGitHubIntakeItemSource(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error) {
	repository, _ := trigger.Configuration["repository"].(string)
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeGitHubClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &gitHubIntakeItemSource{github: client, repository: repository}, nil
}

func (unsupportedIntakeItemSource) Search(context.Context, string, int) ([]IntakeItem, error) {
	return nil, errIntakeSearchUnsupported
}

func (unsupportedIntakeItemSource) Get(context.Context, string) (*IntakeItem, error) {
	return nil, errIntakeSearchUnsupported
}

func (s *gitHubIntakeItemSource) Search(ctx context.Context, query string, limit int) ([]IntakeItem, error) {
	result, _, err := s.github.SearchIssues(ctx, gitHubIssueSearchQuery(s.repository, query), &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: limit},
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return gitHubIssueItems(result.Issues, limit), nil
}

func (s *gitHubIntakeItemSource) Get(ctx context.Context, id string) (*IntakeItem, error) {
	number, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(id), "#"))
	if err != nil || number <= 0 {
		return nil, errIntakeItemNotFound
	}

	issue, _, err := s.github.GetIssue(ctx, s.repository, number)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.IsPullRequest() {
		return nil, errIntakeItemNotFound
	}

	item := gitHubIssueItem(issue)
	return &item, nil
}

func resolveLiveIntakeTrigger(tx *gorm.DB, intake *models.FactoryIntake) (*models.Node, *models.Integration, error) {
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, []uuid.UUID{intake.CanvasID})
	if err != nil {
		return nil, nil, err
	}

	spec, ok := specs[intake.CanvasID]
	if !ok {
		return nil, nil, errIntakeNotConnected
	}

	graph := resolveIntakeGraph(intake.Source, spec)
	trigger := findIntakeNode(spec.Nodes, graph.TriggerNodeID)
	if trigger == nil || trigger.IntegrationID == nil || strings.TrimSpace(*trigger.IntegrationID) == "" {
		return nil, nil, errIntakeNotConnected
	}

	integrationID, err := uuid.Parse(strings.TrimSpace(*trigger.IntegrationID))
	if err != nil {
		return nil, nil, errIntakeNotConnected
	}

	integration, err := models.FindIntegrationInTransaction(tx, intake.OrganizationID, integrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errIntakeNotConnected
		}
		return nil, nil, err
	}
	if integration.State != models.IntegrationStateReady {
		return nil, nil, errIntakeNotConnected
	}

	return trigger, integration, nil
}

func gitHubIssueItems(issues []*github.Issue, limit int) []IntakeItem {
	items := make([]IntakeItem, 0, limit)
	for _, issue := range issues {
		if len(items) == limit {
			break
		}
		if issue == nil || issue.IsPullRequest() {
			continue
		}
		items = append(items, gitHubIssueItem(issue))
	}
	return items
}

func gitHubIssueItem(issue *github.Issue) IntakeItem {
	return IntakeItem{
		ID:    strconv.Itoa(issue.GetNumber()),
		Key:   fmt.Sprintf("#%d", issue.GetNumber()),
		Title: issue.GetTitle(),
		Body:  issue.GetBody(),
		URL:   issue.GetHTMLURL(),
	}
}

func gitHubIssueSearchQuery(repository, query string) string {
	base := fmt.Sprintf("repo:%s is:issue is:open", repository)
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return base
	}
	return fmt.Sprintf("%s %q", base, trimmed)
}

func intakeItemLimit(query string, requested int) int {
	if requested > 0 {
		if requested > maxIntakeItems {
			return maxIntakeItems
		}
		return requested
	}
	if strings.TrimSpace(query) == "" {
		return defaultLatestIntakeItems
	}
	return defaultSearchIntakeItems
}

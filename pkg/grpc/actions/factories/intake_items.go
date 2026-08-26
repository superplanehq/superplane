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
	"github.com/superplanehq/superplane/pkg/integrations/pagerduty"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	defaultLatestIntakeItems = 5
	defaultSearchIntakeItems = 10
	maxIntakeItems           = 50
)

var (
	errIntakeNotConnected = errors.New("intake is not connected")
	errIntakeItemNotFound = errors.New("intake item not found")
)

type liveIntakeItemSource struct {
	source        string
	github        *common.Client
	repository    string
	sentry        *sentry.Client
	sentryProject string
	pagerduty     *pagerduty.Client
	serviceID     string
}

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

	source := &liveIntakeItemSource{source: intake.Source}
	switch intake.Source {
	case models.FactoryIntakeSourceGitHubIssues:
		repository, _ := trigger.Configuration["repository"].(string)
		repository = strings.TrimSpace(repository)
		if repository == "" {
			return nil, errIntakeNotConnected
		}
		client, err := newIntakeGitHubClient(deps, tx, integration)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
		}
		source.github = client
		source.repository = repository
	case models.FactoryIntakeSourceSentryExceptions:
		client, err := newIntakeSentryClient(deps, tx, integration)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
		}
		source.sentry = client
		if project, ok := trigger.Configuration["project"].(string); ok {
			source.sentryProject = strings.TrimSpace(project)
		}
	case models.FactoryIntakeSourcePagerDutyIncidents:
		serviceID, _ := trigger.Configuration["service"].(string)
		serviceID = strings.TrimSpace(serviceID)
		if serviceID == "" {
			return nil, errIntakeNotConnected
		}
		client, err := newIntakePagerDutyClient(deps, tx, integration)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
		}
		source.pagerduty = client
		source.serviceID = serviceID
	default:
		return nil, models.ErrFactoryIntakeSourceInvalid
	}

	return source, nil
}

func (s *liveIntakeItemSource) Search(ctx context.Context, query string, limit int) ([]IntakeItem, error) {
	switch s.source {
	case models.FactoryIntakeSourceGitHubIssues:
		return s.searchGitHub(ctx, query, limit)
	case models.FactoryIntakeSourceSentryExceptions:
		return s.searchSentry(query, limit)
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return s.searchPagerDuty(query, limit)
	default:
		return nil, models.ErrFactoryIntakeSourceInvalid
	}
}

func (s *liveIntakeItemSource) Get(ctx context.Context, id string) (*IntakeItem, error) {
	switch s.source {
	case models.FactoryIntakeSourceGitHubIssues:
		return s.getGitHub(ctx, id)
	case models.FactoryIntakeSourceSentryExceptions:
		return s.getSentry(id)
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return s.getPagerDuty(id)
	default:
		return nil, models.ErrFactoryIntakeSourceInvalid
	}
}

func (s *liveIntakeItemSource) searchGitHub(ctx context.Context, query string, limit int) ([]IntakeItem, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		issues, _, err := s.github.ListIssues(ctx, s.repository, &github.IssueListByRepoOptions{
			State:       "open",
			Sort:        "created",
			Direction:   "desc",
			ListOptions: github.ListOptions{PerPage: gitHubIntakePageSize(limit)},
		})
		if err != nil {
			return nil, err
		}
		return gitHubIssueItems(issues, limit), nil
	}

	result, _, err := s.github.SearchIssues(ctx, fmt.Sprintf("repo:%s is:issue is:open %s", s.repository, trimmed), &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: gitHubIntakePageSize(limit)},
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return gitHubIssueItems(result.Issues, limit), nil
}

func (s *liveIntakeItemSource) getGitHub(ctx context.Context, id string) (*IntakeItem, error) {
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

func (s *liveIntakeItemSource) searchSentry(query string, limit int) ([]IntakeItem, error) {
	issues, err := s.sentry.ListIssues()
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(issues))
	for _, issue := range issues {
		if s.sentryProject != "" && issue.Project != nil && issue.Project.Slug != s.sentryProject {
			continue
		}
		items = append(items, sentryIssueItem(issue))
	}

	page, _ := pageIntakeItems(items, query, limit, 0)
	return page, nil
}

func (s *liveIntakeItemSource) getSentry(id string) (*IntakeItem, error) {
	issue, err := s.sentry.GetIssue(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.ID == "" {
		return nil, errIntakeItemNotFound
	}

	item := sentryIssueItem(*issue)
	return &item, nil
}

func (s *liveIntakeItemSource) searchPagerDuty(query string, limit int) ([]IntakeItem, error) {
	incidents, err := s.pagerduty.ListIncidents([]string{s.serviceID})
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(incidents))
	for _, incident := range incidents {
		items = append(items, pagerDutyIncidentItem(incident))
	}

	page, _ := pageIntakeItems(items, query, limit, 0)
	return page, nil
}

func (s *liveIntakeItemSource) getPagerDuty(id string) (*IntakeItem, error) {
	incidents, err := s.pagerduty.ListIncidents([]string{s.serviceID})
	if err != nil {
		return nil, err
	}

	for _, incident := range incidents {
		if incident.ID == strings.TrimSpace(id) {
			item := pagerDutyIncidentItem(incident)
			return &item, nil
		}
	}

	return nil, errIntakeItemNotFound
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

func newIntakeSentryClient(deps IntakeDependencies, tx *gorm.DB, integration *models.Integration) (*sentry.Client, error) {
	integrationContext, err := newIntakeIntegrationContext(deps, tx, integration)
	if err != nil {
		return nil, err
	}
	return sentry.NewClient(deps.Registry.HTTPContext(), integrationContext)
}

func newIntakePagerDutyClient(deps IntakeDependencies, tx *gorm.DB, integration *models.Integration) (*pagerduty.Client, error) {
	integrationContext, err := newIntakeIntegrationContext(deps, tx, integration)
	if err != nil {
		return nil, err
	}
	return pagerduty.NewClient(deps.Registry.HTTPContext(), integrationContext)
}

func newIntakeIntegrationContext(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*contexts.IntegrationContext, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	return contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil), nil
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

func sentryIssueItem(issue sentry.Issue) IntakeItem {
	url := strings.TrimSpace(issue.Permalink)
	if url == "" {
		url = strings.TrimSpace(issue.WebURL)
	}

	key := strings.TrimSpace(issue.ShortID)
	if key == "" {
		key = strings.TrimSpace(issue.ID)
	}

	return IntakeItem{
		ID:    strings.TrimSpace(issue.ID),
		Key:   key,
		Title: strings.TrimSpace(issue.Title),
		Body:  url,
		URL:   url,
	}
}

func pagerDutyIncidentItem(incident pagerduty.Incident) IntakeItem {
	key := strings.TrimSpace(incident.ID)
	if incident.IncidentNumber > 0 {
		key = fmt.Sprintf("#%d", incident.IncidentNumber)
	}

	body := strings.TrimSpace(incident.Description)
	if body == "" {
		body = strings.TrimSpace(incident.HTMLURL)
	}

	return IntakeItem{
		ID:    strings.TrimSpace(incident.ID),
		Key:   key,
		Title: strings.TrimSpace(incident.Title),
		Body:  body,
		URL:   strings.TrimSpace(incident.HTMLURL),
	}
}

func filterIntakeItems(items []IntakeItem, query string, limit int) []IntakeItem {
	page, _ := pageIntakeItems(items, query, limit, 0)
	return page
}

func pageIntakeItems(items []IntakeItem, query string, limit, offset int) ([]IntakeItem, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	matched := make([]IntakeItem, 0, len(items))
	for _, item := range items {
		if trimmed != "" && !intakeItemMatches(item, trimmed) {
			continue
		}
		matched = append(matched, item)
	}

	if offset < 0 {
		offset = 0
	}
	if offset >= len(matched) {
		return []IntakeItem{}, false
	}

	end := len(matched)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return matched[offset:end], end < len(matched)
}

func gitHubIntakePageSize(limit int) int {
	if limit > intakeSeedPageSize {
		return limit
	}
	return intakeSeedPageSize
}

func intakeItemMatches(item IntakeItem, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{item.Key, item.Title, item.Body, item.ID}, " "))
	return strings.Contains(haystack, query)
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

package factories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	ghdependabot "github.com/superplanehq/superplane/pkg/integrations/github/dependabot"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	defaultLatestIntakeItems = 5
	defaultSearchIntakeItems = 10
	maxIntakeItems           = 50
)

var (
	errIntakeNotConnected       = errors.New("intake is not connected")
	errIntakeItemNotFound       = errors.New("intake item not found")
	errIntakeSearchUnsupported  = errors.New("this intake cannot search items yet")
	errIntakeRefreshUnsupported = errors.New("no intake supports backlog refresh")
	intakeItemSourceByTrigger   = map[string]intakeItemSourceBuilder{}
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
	registerIntakeItemSource("github.onDependabotAlert", newDependabotIntakeItemSource)
	registerIntakeItemSource("jira.onIssue", newJiraIntakeItemSource)
	registerIntakeItemSource("productive.onTask", newProductiveIntakeItemSource)
	registerIntakeItemSource("sentry.onIssue", newSentryIntakeItemSource)
}

type gitHubIntakeItemSource struct {
	github                   *common.Client
	repository               string
	repositoryProbe          sync.Once
	repositoryReadabilityErr error
}

type jiraIntakeItemSource struct {
	jira               *jira.Client
	projectKey         string
	siteURL            string
	projectProbe       sync.Once
	projectReadableErr error
}

type productiveIntakeItemSource struct {
	productive            *productive.Client
	projectID             string
	organizationID        string
	taskListIDs           []string
	excludeKeyTasks       bool
	projectProbe          sync.Once
	projectReadabilityErr error
}

type sentryIntakeItemSource struct {
	sentry  *sentry.Client
	project string
}

type unsupportedIntakeItemSource struct{}

type intakeItemAvailabilitySource interface {
	AvailabilityScope() string
	ItemIDFromOriginURL(rawURL string) (string, bool)
	IsItemAvailable(ctx context.Context, id string) (bool, error)
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

	builder := intakeItemSourceByTrigger[trigger.ComponentName()]
	if builder == nil {
		return unsupportedIntakeItemSource{}, nil
	}

	source, err := builder(ctx, deps, tx, trigger, integration)
	if err != nil {
		return nil, err
	}
	if productiveSource, ok := source.(*productiveIntakeItemSource); ok {
		settings := productiveIntakeSettings(tx, intake.CanvasID)
		productiveSource.excludeKeyTasks = settings.ExcludeKeyTasks
		productiveSource.taskListIDs = settings.TaskListIDs
	}
	return source, nil
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

func (s *gitHubIntakeItemSource) RemoteFetch(ctx context.Context, req *http.Request) (*http.Response, error) {
	return s.github.HTTPDo(req.WithContext(ctx))
}

func newJiraIntakeItemSource(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error) {
	projectKey, _ := trigger.Configuration["project"].(string)
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeJiraClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &jiraIntakeItemSource{
		jira:       client,
		projectKey: projectKey,
		siteURL:    jira.SiteURLFromMetadata(integration.Metadata.Data()),
	}, nil
}

func (s *jiraIntakeItemSource) Search(_ context.Context, query string, limit int) ([]IntakeItem, error) {
	jql := fmt.Sprintf(`project = "%s" AND resolution = Unresolved`, jiraQuotedProjectKey(s.projectKey))
	query = strings.TrimSpace(query)
	if query != "" {
		jql += fmt.Sprintf(` AND text ~ "%s"`, jiraQuotedProjectKey(query))
	}
	jql += " ORDER BY updated DESC"

	hits, err := s.jira.SearchIssues(jql, limit)
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(hits))
	for _, hit := range hits {
		items = append(items, jiraIssueItem(hit, s.siteURL))
	}
	return items, nil
}

func (s *jiraIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	issueKey := strings.TrimSpace(id)
	if issueKey == "" {
		return nil, errIntakeItemNotFound
	}

	issue, err := s.jira.GetIssue(issueKey)
	if err != nil {
		return nil, err
	}
	// A Jira site holds every project the connection can read, so an issue
	// key alone does not say that the issue belongs to this intake. Without
	// this check a caller could import an issue from another project.
	if issue == nil || !s.ownsIssue(issue.Fields) {
		return nil, errIntakeItemNotFound
	}

	item := jiraIssueFromFullIssue(issue, s.siteURL)
	return &item, nil
}

// ownsIssue reports whether the issue belongs to the project this intake
// listens on. An issue without a readable project key is rejected, so a
// missing field cannot widen the boundary.
func (s *jiraIntakeItemSource) ownsIssue(fields map[string]any) bool {
	project, _ := fields["project"].(map[string]any)
	key, _ := project["key"].(string)
	return key != "" && strings.EqualFold(strings.TrimSpace(key), s.projectKey)
}

func (s *jiraIntakeItemSource) AvailabilityScope() string {
	return "jira:" + strings.ToUpper(s.projectKey)
}

func (s *jiraIntakeItemSource) ItemIDFromOriginURL(rawURL string) (string, bool) {
	issueKey, ok := parseJiraIssueURL(rawURL, s.siteURL)
	// Every project of the site shares one browse path, so the host match
	// alone would let a backlog item of another project look like it came
	// from this intake.
	if !ok || !strings.EqualFold(jiraIssueProjectKey(issueKey), s.projectKey) {
		return "", false
	}

	return issueKey, true
}

func (s *jiraIntakeItemSource) IssueFiles(ctx context.Context, issueKey, description string) ([]jira.IssueFile, error) {
	return s.jira.IssueFiles(ctx, issueKey, description)
}

func (s *jiraIntakeItemSource) IsItemAvailable(_ context.Context, id string) (bool, error) {
	issueKey := strings.TrimSpace(id)
	if issueKey == "" {
		return false, errIntakeItemNotFound
	}

	issue, err := s.jira.GetIssueWithOptions(issueKey, jira.GetIssueOptions{Fields: "status,project"})
	if err != nil {
		s.projectProbe.Do(func() {
			_, s.projectReadableErr = s.jira.SearchIssues(
				fmt.Sprintf(`project = "%s"`, jiraQuotedProjectKey(s.projectKey)),
				1,
			)
		})
		if s.projectReadableErr != nil {
			return false, s.projectReadableErr
		}
		return false, nil
	}
	if issue == nil || !s.ownsIssue(issue.Fields) {
		return false, nil
	}

	status, _ := issue.Fields["status"].(map[string]any)
	if status == nil {
		return true, nil
	}
	category, _ := status["statusCategory"].(map[string]any)
	key, _ := category["key"].(string)
	return !strings.EqualFold(key, "done"), nil
}

// jiraIssueProjectKey reads the project of an issue key such as ENG-42. A key
// without the "<project>-<number>" shape reports an empty project.
func jiraIssueProjectKey(issueKey string) string {
	return jira.ProjectKeyFromIssueKey(issueKey)
}

func jiraIssueItem(hit jira.IssueSearchHit, siteURL string) IntakeItem {
	title := jiraIssueSummary(hit.Fields)
	return IntakeItem{
		ID:    hit.Key,
		Key:   hit.Key,
		Title: title,
		URL:   jira.IssueURL(siteURL, hit.Key),
	}
}

func jiraIssueFromFullIssue(issue *jira.Issue, siteURL string) IntakeItem {
	return IntakeItem{
		ID:    issue.Key,
		Key:   issue.Key,
		Title: jiraIssueSummary(issue.Fields),
		Body:  jira.IssueDescriptionText(issue),
		URL:   jira.IssueURL(siteURL, issue.Key),
	}
}

func jiraIssueSummary(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	summary, _ := fields["summary"].(string)
	return summary
}

func parseJiraIssueURL(rawURL, siteURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "browse") {
		return "", false
	}

	issueKey := strings.TrimSpace(parts[1])
	if issueKey == "" {
		return "", false
	}

	if siteURL != "" {
		expectedHost := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(siteURL)), "https://")
		expectedHost = strings.TrimPrefix(expectedHost, "http://")
		expectedHost = strings.TrimSuffix(expectedHost, "/")
		if expectedHost != "" && !strings.EqualFold(parsed.Hostname(), strings.Split(expectedHost, "/")[0]) {
			return "", false
		}
	}

	return issueKey, true
}

func newProductiveIntakeItemSource(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error) {
	projectID, _ := trigger.Configuration["project"].(string)
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeProductiveClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &productiveIntakeItemSource{
		productive:      client,
		projectID:       projectID,
		organizationID:  client.OrganizationID,
		excludeKeyTasks: true,
	}, nil
}

func newSentryIntakeItemSource(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	trigger *models.Node,
	integration *models.Integration,
) (intakeItemSource, error) {
	project, _ := trigger.Configuration["project"].(string)
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, errIntakeNotConnected
	}

	client, err := newIntakeSentryClient(deps, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &sentryIntakeItemSource{sentry: client, project: project}, nil
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

func (s *gitHubIntakeItemSource) AvailabilityScope() string {
	return "github:" + strings.ToLower(s.repository)
}

func (s *gitHubIntakeItemSource) ItemIDFromOriginURL(rawURL string) (string, bool) {
	repository, number, ok := parseGitHubIssueURL(rawURL)
	if !ok || !strings.EqualFold(repository, s.repository) {
		return "", false
	}
	return strconv.Itoa(number), true
}

func (s *gitHubIntakeItemSource) IsItemAvailable(ctx context.Context, id string) (bool, error) {
	number, err := strconv.Atoi(id)
	if err != nil || number <= 0 {
		return false, errIntakeItemNotFound
	}

	issue, _, err := s.github.GetIssue(ctx, s.repository, number)
	if common.IsNotFoundError(err) {
		s.repositoryProbe.Do(func() {
			_, s.repositoryReadabilityErr = s.github.FindRepository(s.repository)
		})
		if s.repositoryReadabilityErr != nil {
			return false, s.repositoryReadabilityErr
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if issue == nil || issue.IsPullRequest() {
		return false, nil
	}
	return !strings.EqualFold(issue.GetState(), "closed"), nil
}

type dependabotIntakeItemSource struct {
	github     *common.Client
	repository string
}

func newDependabotIntakeItemSource(
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

	return &dependabotIntakeItemSource{github: client, repository: repository}, nil
}

func (s *dependabotIntakeItemSource) Search(ctx context.Context, query string, limit int) ([]IntakeItem, error) {
	alerts, err := s.github.ListAllOpenDependabotAlerts(ctx, s.repository)
	if err != nil {
		return nil, ghdependabot.UnavailableError(err)
	}

	query = strings.ToLower(strings.TrimSpace(query))
	items := make([]IntakeItem, 0, len(alerts))
	for _, alert := range alerts {
		item, ok := dependabotAlertItem(alert)
		if !ok {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.Body), query) {
			continue
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *dependabotIntakeItemSource) Get(ctx context.Context, id string) (*IntakeItem, error) {
	number, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil || number <= 0 {
		return nil, errIntakeItemNotFound
	}

	alert, _, err := s.github.GetDependabotAlert(ctx, s.repository, number)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, errIntakeItemNotFound
		}
		return nil, ghdependabot.UnavailableError(err)
	}

	item, ok := dependabotAlertItem(alert)
	if !ok {
		return nil, errIntakeItemNotFound
	}
	return &item, nil
}

func dependabotAlertItem(alert *github.DependabotAlert) (IntakeItem, bool) {
	if alert == nil || alert.GetNumber() <= 0 || !strings.EqualFold(alert.GetState(), "open") {
		return IntakeItem{}, false
	}
	copy := ghdependabot.TaskCopyFromAlert(alert)
	page := strings.TrimSpace(alert.GetHTMLURL())
	if page == "" {
		return IntakeItem{}, false
	}
	number := strconv.Itoa(alert.GetNumber())
	return IntakeItem{
		ID:    number,
		Key:   "#" + number,
		Title: copy.Title,
		Body:  copy.Description,
		URL:   page,
	}, true
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
	if issue.GetComments() > 0 {
		comments, err := s.github.ListIssueComments(ctx, s.repository, number)
		if err != nil {
			return nil, err
		}
		item.Body = composeImportedDescription(item.Body, comments)
	}

	return &item, nil
}

func (s *productiveIntakeItemSource) Search(ctx context.Context, query string, limit int) ([]IntakeItem, error) {
	tasks, err := s.productive.ListTasks(s.projectID, query, limit, s.excludeKeyTasks, s.taskListIDs)
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, productiveTaskItem(task, s.organizationID))
	}
	return items, nil
}

func (s *productiveIntakeItemSource) TaskFiles(ctx context.Context, taskID, description string) ([]productive.TaskFile, error) {
	return s.productive.TaskFiles(ctx, taskID, description)
}

func (s *productiveIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	task, err := s.productive.GetTask(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if task.ProjectID != "" && task.ProjectID != s.projectID {
		return nil, errIntakeItemNotFound
	}
	if !s.acceptsTask(*task) {
		return nil, errIntakeItemNotFound
	}

	item := productiveTaskItem(*task, s.organizationID)
	return &item, nil
}

func (s *productiveIntakeItemSource) AvailabilityScope() string {
	return "productive:" + s.organizationID + ":" + s.projectID
}

func (s *productiveIntakeItemSource) ItemIDFromOriginURL(rawURL string) (string, bool) {
	organizationID, taskID, ok := parseProductiveTaskURL(rawURL)
	if !ok || organizationID != s.organizationID {
		return "", false
	}
	return taskID, true
}

func (s *productiveIntakeItemSource) IsItemAvailable(_ context.Context, id string) (bool, error) {
	task, err := s.productive.GetTask(strings.TrimSpace(id))
	if productive.IsNotFoundError(err) {
		s.projectProbe.Do(func() {
			_, s.projectReadabilityErr = s.productive.GetProject(s.projectID)
		})
		if s.projectReadabilityErr != nil {
			return false, s.projectReadabilityErr
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !s.acceptsTask(*task) {
		return false, nil
	}
	return !task.Closed, nil
}

func (s *productiveIntakeItemSource) acceptsTask(task productive.Task) bool {
	if s.excludeKeyTasks && task.IsKeyTask() {
		return false
	}
	if len(s.taskListIDs) == 0 {
		return true
	}
	return slices.Contains(s.taskListIDs, task.TaskListID)
}

func productiveTaskItem(task productive.Task, organizationID string) IntakeItem {
	key := task.Number
	if key != "" {
		key = "#" + key
	}
	return IntakeItem{
		ID:    task.ID,
		Key:   key,
		Title: task.Title,
		Body:  task.Description,
		URL:   fmt.Sprintf("https://app.productive.io/%s/tasks/%s", organizationID, task.ID),
	}
}

func (s *sentryIntakeItemSource) Search(_ context.Context, query string, limit int) ([]IntakeItem, error) {
	issues, err := s.sentry.SearchUnresolvedIssues(s.project, query, limit)
	if err != nil {
		return nil, err
	}

	items := make([]IntakeItem, 0, len(issues))
	for _, issue := range issues {
		items = append(items, sentryIssueItem(issue))
	}
	return items, nil
}

func (s *sentryIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	issueID := strings.TrimSpace(id)
	if issueID == "" {
		return nil, errIntakeItemNotFound
	}

	issue, err := s.sentry.GetIssue(issueID)
	if err != nil {
		return nil, err
	}
	if !s.ownsIssue(issue) {
		return nil, errIntakeItemNotFound
	}

	item := sentryIssueItem(*issue)
	event, err := s.sentry.GetPreferredIssueEvent(issueID)
	if err != nil {
		event = nil
	}
	item.Body = sentry.IssueDescription(issue, event)
	return &item, nil
}

// ownsIssue reports whether the issue belongs to the project this intake
// listens on. An issue without a readable project slug is rejected, so a
// missing field cannot widen the boundary.
func (s *sentryIntakeItemSource) ownsIssue(issue *sentry.Issue) bool {
	if issue == nil || issue.Project == nil {
		return false
	}
	slug := strings.TrimSpace(issue.Project.Slug)
	return slug != "" && strings.EqualFold(slug, s.project)
}

func sentryIssueItem(issue sentry.Issue) IntakeItem {
	issueURL := strings.TrimSpace(issue.Permalink)
	if issueURL == "" {
		issueURL = strings.TrimSpace(issue.WebURL)
	}
	return IntakeItem{
		ID:    issue.ID,
		Key:   issue.ShortID,
		Title: issue.Title,
		URL:   issueURL,
	}
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

func parseGitHubIssueURL(rawURL string) (repository string, number int, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", 0, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "issues" {
		return "", 0, false
	}

	number, err = strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return "", 0, false
	}
	return parts[0] + "/" + parts[1], number, true
}

func parseProductiveTaskURL(rawURL string) (organizationID string, taskID string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "app.productive.io") {
		return "", "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "tasks" || parts[2] == "" {
		return "", "", false
	}
	return parts[0], parts[2], true
}

const (
	importedCommentsSeparator     = "____"
	importedCommentsHeader        = "Imported Comments:"
	importedCommentsUnknownAuthor = "unknown"
)

// composeImportedDescription appends an issue's comments to its body when
// creating a draft work order from an imported GitHub issue. Comments are
// expected oldest first (see Client.ListIssueComments); this function trusts
// that order and does not re-sort. If there are no usable comments, the body
// is returned unchanged.
func composeImportedDescription(body string, comments []*github.IssueComment) string {
	usable := make([]*github.IssueComment, 0, len(comments))
	for _, comment := range comments {
		if comment != nil {
			usable = append(usable, comment)
		}
	}
	if len(usable) == 0 {
		return body
	}

	var builder strings.Builder
	if body != "" {
		builder.WriteString(body)
		builder.WriteString("\n")
	}
	builder.WriteString(importedCommentsSeparator)
	builder.WriteString("\n")
	builder.WriteString(importedCommentsHeader)
	builder.WriteString("\n")

	for i, comment := range usable {
		builder.WriteString(importedCommentAuthor(comment))
		builder.WriteString(" ")
		builder.WriteString(comment.GetCreatedAt().UTC().Format(time.RFC3339))
		builder.WriteString("\n")
		builder.WriteString(comment.GetBody())
		if i < len(usable)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

func importedCommentAuthor(comment *github.IssueComment) string {
	login := comment.GetUser().GetLogin()
	if login == "" {
		return importedCommentsUnknownAuthor
	}
	return login
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

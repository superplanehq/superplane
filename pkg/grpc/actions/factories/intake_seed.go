package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	ghdependabot "github.com/superplanehq/superplane/pkg/integrations/github/dependabot"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const (
	// intakeSeedSize is how many items a new intake analyzes at once. A source
	// with few open items gives fewer, so the seed is an upper bound.
	intakeSeedSize = 30

	// intakeSentrySeedSize is how many unresolved Sentry issues a new intake
	// imports. The wizard tells the user this number.
	intakeSentrySeedSize = 10

	// intakeSentrySeedEventWindow is how many recent trigger events a reseed
	// reads so it can drop issues that already sit on the intake.
	intakeSentrySeedEventWindow = 200

	// intakeProductiveSeedSize is how many open Productive.io tasks a new
	// intake imports. The wizard tells the user this number.
	intakeProductiveSeedSize = 10

	// intakeJiraSeedSize is how many unresolved Jira issues a new intake
	// imports. The wizard tells the user this number.
	intakeJiraSeedSize = 10

	// intakeJiraSeedEventWindow is how many recent trigger events a reseed
	// reads so it can drop issues that already sit on the intake.
	intakeJiraSeedEventWindow = 200

	// intakeGitHubIssuePayloadType is the payload type the GitHub trigger emits.
	// A seeded item uses the same one, so the graph reads it the same way.
	intakeGitHubIssuePayloadType = "github.issue"

	// intakeSentryIssuePayloadType is the payload type the Sentry trigger emits.
	intakeSentryIssuePayloadType = "sentry.issue"

	// intakeJiraIssuePayloadType is the payload type the Jira trigger emits.
	intakeJiraIssuePayloadType = jira.IssueEventPayloadType
)

type intakeSeedResult struct {
	itemCount int
	skipped   bool
}

// seedIntake gives a new intake work at once: the newest open items of the
// source enter the graph as if they had just arrived. Without a seed the intake
// stays empty until the source sends its next event, which can take days.
func seedIntake(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	source string,
	binding *intakeBinding,
) (intakeSeedResult, error) {
	// An unbound intake has nothing to read from. Its items arrive through the
	// webhook alone.
	installation := binding.installation()
	if installation == nil {
		return intakeSeedResult{skipped: true}, nil
	}

	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return seedGitHubIssues(ctx, deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceProductiveTasks:
		return seedProductiveTasks(deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceSentryExceptions:
		return seedSentryIssues(deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceJiraIssues:
		return seedJiraIssues(deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceDependabotAlerts:
		return seedDependabotAlerts(ctx, deps, tx, canvasID, binding, installation)
	}

	// The remaining sources cannot be read yet, so they start empty.
	return intakeSeedResult{skipped: true}, nil
}

// SeedExistingIntake reseeds one intake from its live trigger binding, so a
// later canvas edit still reads the repository the trigger listens on.
func SeedExistingIntake(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	intake *models.FactoryIntake,
) error {
	binding, err := liveIntakeBinding(tx, intake)
	if err != nil {
		return err
	}

	_, err = seedIntake(ctx, deps, tx, intake.CanvasID, intake.Source, binding)
	return err
}

// SeedFactoryIntakes reseeds every intake of a workspace. A source that
// cannot be read now is logged and skipped, matching intake create.
func SeedFactoryIntakes(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	factory *models.Factory,
) error {
	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return fmt.Errorf("list intakes: %w", err)
	}

	for i := range intakes {
		intake := &intakes[i]
		if err := SeedExistingIntake(ctx, deps, tx, intake); err != nil {
			log.Warnf("factory %s: intake %s starts without a first batch: %v", factory.ID, intake.ID, err)
		}
	}

	return nil
}

func liveIntakeBinding(tx *gorm.DB, intake *models.FactoryIntake) (*intakeBinding, error) {
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, intake.CanvasID)
	if err != nil {
		return nil, fmt.Errorf("load intake canvas: %w", err)
	}

	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, fmt.Errorf("load intake graph: %w", err)
	}

	var trigger *models.Node
	for i := range version.Nodes {
		if version.Nodes[i].ID == intakeTriggerNodeID {
			trigger = &version.Nodes[i]
			break
		}
	}
	if trigger == nil || trigger.IntegrationID == nil {
		return nil, nil
	}

	integrationID, err := uuid.Parse(*trigger.IntegrationID)
	if err != nil {
		log.Warnf("intake %s: seed left unbound, invalid integration id %q", intake.ID, *trigger.IntegrationID)
		return nil, nil
	}

	integration, err := models.FindIntegrationInTransaction(tx, intake.OrganizationID, integrationID)
	if err != nil {
		log.Warnf("intake %s: seed left unbound, integration %s not found: %v", intake.ID, integrationID, err)
		return nil, nil
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: trigger.Configuration,
		Installation:  integration,
	}, nil
}

func seedGitHubIssues(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeGitHubClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	repository, _ := binding.Configuration["repository"].(string)
	payloads, err := newestGitHubIssueEvents(ctx, client, repository, intakeSeedSize)
	if err != nil {
		return intakeSeedResult{}, err
	}

	if err := emitIntakeEvents(tx, canvasID, intakeGitHubIssuePayloadType, payloads); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(payloads)}, nil
}

func seedDependabotAlerts(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeGitHubClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	repository, _ := binding.Configuration["repository"].(string)
	alerts, _, err := client.ListOpenDependabotAlerts(ctx, repository, intakeSeedSize)
	if err != nil {
		return intakeSeedResult{}, ghdependabot.UnavailableError(fmt.Errorf("failed to list Dependabot alerts of %s: %w", repository, err))
	}

	payloads := make([]map[string]any, 0, len(alerts))
	for _, alert := range alerts {
		event, err := ghdependabot.AlertEvent(alert)
		if err != nil {
			return intakeSeedResult{}, err
		}
		payloads = append(payloads, event)
	}
	slices.Reverse(payloads)

	if err := emitIntakeEvents(tx, canvasID, ghdependabot.AlertPayloadType, payloads); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(payloads)}, nil
}

func seedJiraIssues(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeJiraClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	projectKey, _ := binding.Configuration["project"].(string)
	siteURL := jira.SiteURLFromMetadata(installation.Metadata.Data())
	hits, err := newestJiraIssueHits(client, projectKey, intakeJiraSeedSize)
	if err != nil {
		return intakeSeedResult{}, err
	}

	return seedKnownJiraIssues(tx, canvasID, client.GetIssue, hits, siteURL)
}

func seedKnownJiraIssues(
	tx *gorm.DB,
	canvasID uuid.UUID,
	load jiraIssueLoader,
	hits []jira.IssueSearchHit,
	siteURL string,
) (intakeSeedResult, error) {
	hits, err := filterJiraIssuesForSeed(tx, canvasID, hits, siteURL)
	if err != nil {
		return intakeSeedResult{}, err
	}

	payloads, err := jiraIssueEvents(load, hits, siteURL)
	if err != nil {
		return intakeSeedResult{}, err
	}

	if err := emitIntakeEvents(tx, canvasID, intakeJiraIssuePayloadType, payloads); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(payloads)}, nil
}

func filterJiraIssuesForSeed(tx *gorm.DB, canvasID uuid.UUID, hits []jira.IssueSearchHit, siteURL string) ([]jira.IssueSearchHit, error) {
	if len(hits) == 0 {
		return hits, nil
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("load intake canvas: %w", err)
	}
	if canvas.FactoryID == nil {
		return nil, fmt.Errorf("intake canvas is not owned by a factory")
	}

	factory, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	seen, err := jiraIssueKeysOnTrigger(tx, canvasID)
	if err != nil {
		return nil, err
	}

	kept := make([]jira.IssueSearchHit, 0, len(hits))
	for _, hit := range hits {
		issueKey := strings.TrimSpace(hit.Key)
		if issueKey == "" {
			continue
		}
		if seen[issueKey] {
			log.Infof("skipping Jira issue %s: already on intake", issueKey)
			continue
		}

		ref, ok := jira.IssueRefFromSite(siteURL, issueKey)
		if !ok {
			kept = append(kept, hit)
			continue
		}

		hasOrder, err := jira.IssueHasWorkOrder(tx, factory, ref)
		if err != nil {
			return nil, err
		}
		if hasOrder {
			log.Infof("skipping Jira issue %s on %s: work order already exists", ref.Key, ref.Host)
			continue
		}

		kept = append(kept, hit)
	}

	return kept, nil
}

func jiraIssueKeysOnTrigger(tx *gorm.DB, canvasID uuid.UUID) (map[string]bool, error) {
	events, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, intakeJiraSeedEventWindow, nil)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for i := range events {
		issueKey, ok := jira.IssueKeyFromEventData(events[i].Data.Data())
		if ok {
			seen[issueKey] = true
		}
	}
	return seen, nil
}

func seedProductiveTasks(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeProductiveClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	project, _ := binding.Configuration["project"].(string)
	settings := productiveIntakeSettings(tx, canvasID)
	documents, err := newestProductiveSeedDocuments(
		client,
		project,
		settings.ExcludeKeyTasks,
		settings.TaskListIDs,
	)
	if err != nil {
		return intakeSeedResult{}, fmt.Errorf("failed to list the tasks of project %s: %w", project, err)
	}

	if err := emitIntakeEvents(tx, canvasID, productive.TaskPayloadType, productiveTaskEvents(documents)); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(documents)}, nil
}

func newestProductiveSeedDocuments(
	client *productive.Client,
	project string,
	regularOnly bool,
	taskListIDs []string,
) ([]map[string]any, error) {
	return client.ListNewestOpenTaskDocuments(project, intakeProductiveSeedSize, regularOnly, taskListIDs)
}

// productiveTaskEvents shapes each task of a newest-first page like the event
// the trigger emits when it polls, so the rest of the graph cannot tell a
// seeded task from a polled one.
func productiveTaskEvents(documents []map[string]any) []map[string]any {
	events := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		events = append(events, productive.TaskEnvelope(productive.TaskCreatedEvent, document))
	}

	// The intake lists its runs newest first. Emitting the oldest task first
	// keeps the newest task at the top, where the reader expects it.
	slices.Reverse(events)

	return events
}

func productiveIntakeSettings(tx *gorm.DB, canvasID uuid.UUID) intakeSettings {
	return liveIntakeSettings(tx, models.FactoryIntakeSourceProductiveTasks, canvasID, defaultProductiveIntakeSettings())
}

// intakeInstructions reads the per-intake instructions off the live canvas.
// An unreadable canvas gives none rather than a source default.
func intakeInstructions(tx *gorm.DB, intake *models.FactoryIntake) string {
	return liveIntakeSettings(tx, intake.Source, intake.CanvasID, intakeSettings{}).Instructions
}

// liveIntakeSettings reads the settings out of the intake's live canvas. It
// returns fallback when the canvas cannot be read.
func liveIntakeSettings(tx *gorm.DB, source string, canvasID uuid.UUID, fallback intakeSettings) intakeSettings {
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, []uuid.UUID{canvasID})
	if err != nil {
		return fallback
	}
	spec, ok := specs[canvasID]
	if !ok {
		return fallback
	}
	return intakeSettingsFromGraph(source, resolveIntakeGraph(source, spec), spec)
}

func seedSentryIssues(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeSentryClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	project, _ := binding.Configuration["project"].(string)
	issues, err := client.ListNewestUnresolvedIssues(project, intakeSentrySeedSize)
	if err != nil {
		return intakeSeedResult{}, fmt.Errorf("failed to list the issues of project %s: %w", project, err)
	}

	return seedKnownSentryIssues(tx, canvasID, client, issues)
}

func seedKnownSentryIssues(
	tx *gorm.DB,
	canvasID uuid.UUID,
	client *sentry.Client,
	issues []sentry.Issue,
) (intakeSeedResult, error) {
	issues, err := filterSentryIssuesForSeed(tx, canvasID, issues)
	if err != nil {
		return intakeSeedResult{}, err
	}

	if err := emitIntakeEvents(tx, canvasID, intakeSentryIssuePayloadType, sentryIssueEvents(client, issues)); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(issues)}, nil
}

func filterSentryIssuesForSeed(tx *gorm.DB, canvasID uuid.UUID, issues []sentry.Issue) ([]sentry.Issue, error) {
	if len(issues) == 0 {
		return issues, nil
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("load intake canvas: %w", err)
	}
	if canvas.FactoryID == nil {
		return nil, fmt.Errorf("intake canvas is not owned by a factory")
	}

	factory, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	seen, err := sentryIssueIDsOnTrigger(tx, canvasID)
	if err != nil {
		return nil, err
	}

	kept := make([]sentry.Issue, 0, len(issues))
	for _, issue := range issues {
		if seen[issue.ID] {
			log.Infof("skipping Sentry issue %s: already on intake", issue.ID)
			continue
		}

		hasOrder, err := sentry.IssueHasWorkOrder(tx, factory, issue.ID)
		if err != nil {
			return nil, err
		}
		if hasOrder {
			log.Infof("skipping Sentry issue %s: work order already exists", issue.ID)
			continue
		}

		kept = append(kept, issue)
	}

	return kept, nil
}

func sentryIssueIDsOnTrigger(tx *gorm.DB, canvasID uuid.UUID) (map[string]bool, error) {
	events, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, intakeSentrySeedEventWindow, nil)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for i := range events {
		issueID, ok := sentry.IssueIDFromEventData(events[i].Data.Data())
		if ok {
			seen[issueID] = true
		}
	}
	return seen, nil
}

// sentryIssueEvents shapes each issue of a newest-first page like the webhook
// the trigger emits, so the rest of the graph cannot tell a seeded issue from
// a received one.
func sentryIssueEvents(client *sentry.Client, issues []sentry.Issue) []map[string]any {
	events := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		events = append(events, sentryIssueEvent(client, issue))
	}
	slices.Reverse(events)
	return events
}

func sentryIssueEvent(client *sentry.Client, issue sentry.Issue) map[string]any {
	encoded, err := json.Marshal(issue)
	if err != nil {
		payload := map[string]any{"id": issue.ID, "title": issue.Title}
		return map[string]any{
			"resource":    "issue",
			"action":      "created",
			"data":        map[string]any{"issue": payload},
			"description": sentry.FetchedIssueDescription(client, payload, log.StandardLogger()),
		}
	}

	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		payload = map[string]any{"id": issue.ID, "title": issue.Title}
	}

	timestamp := strings.TrimSpace(issue.LastSeen)
	if timestamp == "" {
		timestamp = strings.TrimSpace(issue.FirstSeen)
	}

	return map[string]any{
		"resource":    "issue",
		"action":      "created",
		"data":        map[string]any{"issue": payload},
		"timestamp":   timestamp,
		"description": sentry.FetchedIssueDescription(client, payload, log.StandardLogger()),
	}
}

func newIntakeSentryClient(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*sentry.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := sentry.NewClient(deps.Registry.HTTPContext(), integrationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build Sentry client: %w", err)
	}

	return client, nil
}

func newIntakeGitHubClient(deps IntakeDependencies, tx *gorm.DB, integration *models.Integration) (*common.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := common.NewClient(integrationContext, deps.Registry.HTTPContext())
	if err != nil {
		return nil, fmt.Errorf("failed to build GitHub client: %w", err)
	}

	return client, nil
}

func newIntakeJiraClient(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*jira.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := jira.NewClient(deps.Registry.HTTPContext(), integrationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build Jira client: %w", err)
	}

	return client, nil
}

func newIntakeProductiveClient(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*productive.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := productive.NewClient(deps.Registry.HTTPContext(), integrationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build Productive client: %w", err)
	}

	return client, nil
}

// newestGitHubIssueEvents reads the newest open issues of the repository.
func newestGitHubIssueEvents(
	ctx context.Context,
	client *common.Client,
	repository string,
	limit int,
) ([]map[string]any, error) {
	issues, err := client.ListNewestOpenIssues(ctx, repository, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list the issues of %s: %w", repository, err)
	}

	return gitHubIssueEvents(issues, repository)
}

// gitHubIssueEvents shapes each issue of a newest-first page like the webhook
// body the trigger would have delivered, so the rest of the graph cannot tell a
// seeded item from a received one.
func gitHubIssueEvents(issues []*github.Issue, repository string) ([]map[string]any, error) {
	events := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		event, err := gitHubIssueEvent(issue, repository)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	// The intake lists its runs newest first. Emitting the oldest issue first
	// keeps the newest issue at the top, where the reader expects it.
	slices.Reverse(events)

	return events, nil
}

// gitHubIssueEvent converts an issue from the API into the body of an "issues"
// webhook. The generated graph reads titles, bodies, labels, and assignees out
// of that shape.
func newestJiraIssueHits(client *jira.Client, projectKey string, limit int) ([]jira.IssueSearchHit, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, fmt.Errorf("project is required")
	}

	jql := fmt.Sprintf(`project = "%s" AND resolution = Unresolved ORDER BY created DESC`, jiraQuotedProjectKey(projectKey))
	hits, err := client.SearchIssues(jql, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list the issues of project %s: %w", projectKey, err)
	}

	return hits, nil
}

// jiraIssueLoader reads one issue by key. The seed takes the read as a
// function so a test can fail a single issue of a batch.
type jiraIssueLoader func(issueKey string) (*jira.Issue, error)

// jiraIssueEvents loads the full issue behind each search hit. One unreadable
// issue - deleted between the search and the fetch, or hidden from the
// connection - must not discard the rest of the first batch, so a failure is
// logged and that issue is left out. A batch where every issue failed still
// reports an error, because that points at the connection rather than at one
// issue.
func jiraIssueEvents(load jiraIssueLoader, hits []jira.IssueSearchHit, siteURL string) ([]map[string]any, error) {
	events := make([]map[string]any, 0, len(hits))
	var lastErr error
	for _, hit := range hits {
		event, err := jiraIssueEvent(load, hit.Key, siteURL)
		if err != nil {
			lastErr = err
			log.Warnf("intake seed: issue %s stays out of the first batch: %v", hit.Key, err)
			continue
		}

		events = append(events, event)
	}

	if len(events) == 0 && lastErr != nil {
		return nil, lastErr
	}

	slices.Reverse(events)
	return events, nil
}

func jiraIssueEvent(load jiraIssueLoader, issueKey, siteURL string) (map[string]any, error) {
	issue, err := load(issueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load issue %s: %w", issueKey, err)
	}

	event := jira.NewIssueEvent("created", issue, nil, nil, siteURL)

	encoded, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to encode issue event: %w", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, fmt.Errorf("failed to read issue event: %w", err)
	}

	return payload, nil
}

func jiraQuotedProjectKey(projectKey string) string {
	escaped := strings.ReplaceAll(projectKey, `\`, `\\`)
	return strings.ReplaceAll(escaped, `"`, `\"`)
}

func gitHubIssueEvent(issue *github.Issue, repository string) (map[string]any, error) {
	encoded, err := json.Marshal(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to encode issue: %w", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, fmt.Errorf("failed to read issue: %w", err)
	}

	// A webhook always sends both lists. The API leaves them out when they are
	// empty, and the intake filters read them without a guard.
	for _, field := range []string{"labels", "assignees"} {
		if _, ok := payload[field]; !ok {
			payload[field] = []any{}
		}
	}

	return map[string]any{
		"action":     "opened",
		"issue":      payload,
		"repository": map[string]any{"full_name": repository},
	}, nil
}

// emitIntakeEvents feeds the payloads into the intake's trigger on the path a
// webhook takes: one pending root event for each item, and one run for each
// event.
func emitIntakeEvents(tx *gorm.DB, canvasID uuid.UUID, payloadType string, payloads []map[string]any) error {
	if len(payloads) == 0 {
		return nil
	}

	node, err := models.FindCanvasNode(tx, canvasID, intakeTriggerNodeID)
	if err != nil {
		return fmt.Errorf("failed to load the intake trigger: %w", err)
	}

	emitted := []models.CanvasEvent{}
	events := contexts.NewEventContext(tx, node, nil, func(created []models.CanvasEvent) {
		emitted = append(emitted, created...)
	})

	for _, payload := range payloads {
		if err := events.Emit(payloadType, payload); err != nil {
			return fmt.Errorf("failed to emit an intake event: %w", err)
		}
	}

	for i := range emitted {
		// The event router also polls for pending events, so a lost message
		// only delays the analysis.
		if err := messages.PublishCanvasEventCreatedMessage(&emitted[i]); err != nil {
			log.Warnf("failed to publish intake event %s: %v", emitted[i].ID, err)
		}
	}

	return nil
}

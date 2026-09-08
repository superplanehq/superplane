package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/notion"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	// intakeSeedSize is how many items a new intake analyzes at once. A source
	// with few open items gives fewer, so the seed is an upper bound.
	intakeSeedSize = 30

	// intakeGitHubIssuePayloadType is the payload type the GitHub trigger emits.
	// A seeded item uses the same one, so the graph reads it the same way.
	intakeGitHubIssuePayloadType = "github.issue"
)

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
) error {
	// An unbound intake has nothing to read from. Its items arrive through the
	// webhook alone.
	installation := binding.installation()
	if installation == nil {
		return nil
	}

	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return seedGitHubIssues(ctx, deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceProductiveTasks:
		return seedProductiveTasks(deps, tx, canvasID, binding, installation)
	case models.FactoryIntakeSourceNotionPages:
		return seedNotionPages(deps, tx, canvasID, binding, installation)
	}

	// The remaining sources cannot be read yet, so they start empty.
	return nil
}

func seedGitHubIssues(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) error {
	client, err := newIntakeGitHubClient(deps, tx, installation)
	if err != nil {
		return err
	}

	repository, _ := binding.Configuration["repository"].(string)
	payloads, err := newestGitHubIssueEvents(ctx, client, repository, intakeSeedSize)
	if err != nil {
		return err
	}

	return emitIntakeEvents(tx, canvasID, intakeGitHubIssuePayloadType, payloads)
}

func seedProductiveTasks(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) error {
	client, err := newIntakeProductiveClient(deps, tx, installation)
	if err != nil {
		return err
	}

	project, _ := binding.Configuration["project"].(string)
	documents, err := client.ListNewestOpenTaskDocuments(project, intakeSeedSize)
	if err != nil {
		return fmt.Errorf("failed to list the tasks of project %s: %w", project, err)
	}

	return emitIntakeEvents(tx, canvasID, productive.TaskPayloadType, productiveTaskEvents(documents))
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

func seedNotionPages(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) error {
	client, err := newIntakeNotionClient(deps, tx, installation)
	if err != nil {
		return err
	}

	database, _ := binding.Configuration["database"].(string)
	pages, err := client.ListNewestPages(database, intakeSeedSize)
	if err != nil {
		return fmt.Errorf("failed to list the pages of database %s: %w", database, err)
	}

	events, err := notionPageEvents(client, pages)
	if err != nil {
		return err
	}

	return emitIntakeEvents(tx, canvasID, notion.PagePayloadType, events)
}

// notionPageEvents shapes each page of a newest-first read like the event the
// trigger emits when it polls, so the rest of the graph cannot tell a seeded
// page from a polled one.
func notionPageEvents(client *notion.Client, pages []map[string]any) ([]map[string]any, error) {
	events := make([]map[string]any, 0, len(pages))
	for _, page := range pages {
		enriched, err := notion.BuildPageEvent(client, page)
		if err != nil {
			return nil, fmt.Errorf("failed to read the content of page %s: %w", page["id"], err)
		}

		events = append(events, notion.PageEnvelope(notion.PageCreatedEvent, enriched))
	}

	// The intake lists its runs newest first. Emitting the oldest page first
	// keeps the newest page at the top of the intake list.
	slices.Reverse(events)

	return events, nil
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
		return nil, fmt.Errorf("failed to build Productive.io client: %w", err)
	}

	return client, nil
}

func newIntakeNotionClient(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*notion.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := notion.NewClient(deps.Registry.HTTPContext(), integrationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build Notion client: %w", err)
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

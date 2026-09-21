package factories

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__IntakeSeed(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	t.Run("seeded issues analyze at once", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		intake := response.GetIntake()

		titles := []string{
			"Handle duplicate refunds on retry",
			"Return 409 when the invoice is already paid",
			"Show a clearer empty state on the billing page",
			"Upgrade the Node 20 base image",
			"Add a flake retry to the checkout e2e suite",
		}

		events, err := gitHubIssueEvents(gitHubIssuePage(titles), "acme/backlog")
		require.NoError(t, err)
		require.NoError(t, emitIntakeEvents(
			database.DB(t.Context()),
			uuid.MustParse(intake.GetCanvasId()),
			intakeGitHubIssuePayloadType,
			events,
		))

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)
		require.Len(t, runs.GetRuns(), len(titles))

		// The newest issue leads the list. The work order is not created yet.
		reported := []string{}
		for _, run := range runs.GetRuns() {
			assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_ANALYZING, run.GetPlacement())
			reported = append(reported, run.GetTitle())
		}
		assert.Equal(t, titles, reported)
	})

	t.Run("an intake without a connection starts empty", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  response.GetIntake().GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, runs.GetRuns())
	})

	t.Run("reseeding an unbound intake stays empty", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		intake, err := factory.FindIntake(database.DB(t.Context()), uuid.MustParse(response.GetIntake().GetId()))
		require.NoError(t, err)
		require.NoError(t, SeedExistingIntake(ctx, deps, database.DB(t.Context()), intake))

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.NoError(t, err)
		assert.Empty(t, runs.GetRuns())
	})

	t.Run("live binding reads the canvas trigger not onboarding config", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		backlogRepository := "acme/backlog"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID:  &integrationID,
			BacklogRepository: &backlogRepository,
		}))

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		otherRepository := "acme/other"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			BacklogRepository: &otherRepository,
		}))

		intake, err := factory.FindIntake(database.DB(t.Context()), uuid.MustParse(response.GetIntake().GetId()))
		require.NoError(t, err)

		binding, err := liveIntakeBinding(database.DB(t.Context()), intake)
		require.NoError(t, err)
		require.NotNil(t, binding)
		assert.Equal(t, backlogRepository, binding.Configuration["repository"])
		require.NotNil(t, binding.Installation)
		assert.Equal(t, integrationID, binding.Installation.ID.String())
	})
}

func Test__GitHubIssueEvents(t *testing.T) {
	t.Run("the newest issue ends up on top of the intake", func(t *testing.T) {
		events, err := gitHubIssueEvents(gitHubIssuePage([]string{"Newest issue", "Older issue"}), "acme/backlog")
		require.NoError(t, err)
		require.Len(t, events, 2)

		// Events are emitted oldest first, so the newest issue ends up on top
		// of the intake list.
		assert.Equal(t, "Older issue", issueEventTitle(t, events[0]))
		assert.Equal(t, "Newest issue", issueEventTitle(t, events[1]))
	})

	t.Run("an event carries what the graph reads", func(t *testing.T) {
		issue := &github.Issue{
			Number: github.Ptr(42),
			Title:  github.Ptr("Handle duplicate refunds on retry"),
			Body:   github.Ptr("A retried refund charges the customer twice."),
		}

		events, err := gitHubIssueEvents([]*github.Issue{issue}, "acme/backlog")
		require.NoError(t, err)
		require.Len(t, events, 1)

		event := events[0]
		assert.Equal(t, "opened", event["action"])
		assert.Equal(t, map[string]any{"full_name": "acme/backlog"}, event["repository"])

		payload, ok := event["issue"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Handle duplicate refunds on retry", payload["title"])
		assert.Equal(t, "A retried refund charges the customer twice.", payload["body"])

		// The filters read both lists without a guard, and GitHub leaves them
		// out of the response when the issue has none.
		assert.Equal(t, []any{}, payload["labels"])
		assert.Equal(t, []any{}, payload["assignees"])
	})
}

func Test__JiraIssueEvents(t *testing.T) {
	issue := func(key, description string) *jira.Issue {
		return &jira.Issue{
			Key: key,
			Fields: map[string]any{
				"summary":     "Summary of " + key,
				"description": jira.WrapInADF(description),
			},
		}
	}

	t.Run("the newest issue ends up on top of the intake", func(t *testing.T) {
		load := func(issueKey string) (*jira.Issue, error) {
			return issue(issueKey, "Description of "+issueKey), nil
		}

		events, err := jiraIssueEvents(load, []jira.IssueSearchHit{{Key: "ENG-2"}, {Key: "ENG-1"}}, "")
		require.NoError(t, err)
		require.Len(t, events, 2)

		// Events are emitted oldest first, so the newest issue ends up on top
		// of the intake list.
		assert.Equal(t, "ENG-1", jiraEventIssueKey(t, events[0]))
		assert.Equal(t, "ENG-2", jiraEventIssueKey(t, events[1]))
	})

	t.Run("an event carries the description as plain text", func(t *testing.T) {
		load := func(issueKey string) (*jira.Issue, error) {
			return issue(issueKey, "A retried refund charges twice."), nil
		}

		events, err := jiraIssueEvents(load, []jira.IssueSearchHit{{Key: "ENG-1"}}, "")
		require.NoError(t, err)
		require.Len(t, events, 1)

		assert.Equal(t, "created", events[0]["action"])
		assert.Equal(t, "A retried refund charges twice.", events[0]["description"])
	})

	t.Run("an event stores the issue page as origin", func(t *testing.T) {
		load := func(issueKey string) (*jira.Issue, error) {
			return issue(issueKey, "Description of "+issueKey), nil
		}

		events, err := jiraIssueEvents(load, []jira.IssueSearchHit{{Key: "ENG-1"}}, "https://acme.atlassian.net")
		require.NoError(t, err)
		require.Len(t, events, 1)

		assert.Equal(t, "https://acme.atlassian.net/browse/ENG-1", events[0]["url"])
		assert.Equal(t, &models.WorkOrderOrigin{
			URL:   "https://acme.atlassian.net/browse/ENG-1",
			Label: "ENG-1",
		}, models.OriginFromIntakePayload(events[0]))
	})

	t.Run("one unreadable issue does not discard the batch", func(t *testing.T) {
		load := func(issueKey string) (*jira.Issue, error) {
			if issueKey == "ENG-2" {
				return nil, fmt.Errorf("issue was deleted")
			}
			return issue(issueKey, "Description of "+issueKey), nil
		}

		events, err := jiraIssueEvents(load, []jira.IssueSearchHit{{Key: "ENG-3"}, {Key: "ENG-2"}, {Key: "ENG-1"}}, "")
		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, "ENG-1", jiraEventIssueKey(t, events[0]))
		assert.Equal(t, "ENG-3", jiraEventIssueKey(t, events[1]))
	})

	t.Run("a batch where every issue fails reports the failure", func(t *testing.T) {
		load := func(string) (*jira.Issue, error) {
			return nil, fmt.Errorf("the connection lost its access")
		}

		_, err := jiraIssueEvents(load, []jira.IssueSearchHit{{Key: "ENG-1"}}, "")
		require.ErrorContains(t, err, "the connection lost its access")
	})

	t.Run("an empty search reports no events", func(t *testing.T) {
		events, err := jiraIssueEvents(nil, nil, "")
		require.NoError(t, err)
		assert.Empty(t, events)
	})
}

func jiraEventIssueKey(t *testing.T, event map[string]any) string {
	t.Helper()

	payload, ok := event["issue"].(map[string]any)
	require.True(t, ok)
	key, ok := payload["key"].(string)
	require.True(t, ok)

	return key
}

func Test__ProductiveTaskEvents(t *testing.T) {
	t.Run("the newest task ends up on top of the intake", func(t *testing.T) {
		events := productiveTaskEvents(productiveTaskPage([]string{"Newest task", "Older task"}))
		require.Len(t, events, 2)

		// Events are emitted oldest first, so the newest task ends up on top
		// of the intake list.
		assert.Equal(t, "Older task", taskEventTitle(t, events[0]))
		assert.Equal(t, "Newest task", taskEventTitle(t, events[1]))
	})

	t.Run("an event carries what the graph reads", func(t *testing.T) {
		document := map[string]any{
			"id":   "91",
			"type": "tasks",
			"attributes": map[string]any{
				"title":       "Fix payment retries",
				"description": "Retries fail silently after the third attempt.",
			},
		}

		events := productiveTaskEvents([]map[string]any{document})
		require.Len(t, events, 1)

		// The graph reads root().data.data.attributes, and a created task is
		// what the intake filters on.
		assert.Equal(t, map[string]any{"event": "task.created"}, events[0]["meta"])
		assert.Equal(t, document, events[0]["data"])
	})
}

// productiveTaskPage builds a page as the API returns it, so the titles are
// given newest first.
func productiveTaskPage(titles []string) []map[string]any {
	documents := make([]map[string]any, 0, len(titles))
	for i, title := range titles {
		documents = append(documents, map[string]any{
			"id":   strconv.Itoa(len(titles) - i),
			"type": "tasks",
			"attributes": map[string]any{
				"title":       title,
				"description": fmt.Sprintf("Description of %s", title),
			},
		})
	}

	return documents
}

func taskEventTitle(t *testing.T, event map[string]any) string {
	t.Helper()

	document, ok := event["data"].(map[string]any)
	require.True(t, ok)
	attributes, ok := document["attributes"].(map[string]any)
	require.True(t, ok)
	title, ok := attributes["title"].(string)
	require.True(t, ok)

	return title
}

// gitHubIssuePage builds a page as the API returns it, so the titles are given
// newest first.
func gitHubIssuePage(titles []string) []*github.Issue {
	issues := make([]*github.Issue, 0, len(titles))
	for i, title := range titles {
		issues = append(issues, &github.Issue{
			Number: github.Ptr(len(titles) - i),
			Title:  github.Ptr(title),
			Body:   github.Ptr(fmt.Sprintf("Body of %s", title)),
		})
	}

	return issues
}

func issueEventTitle(t *testing.T, event map[string]any) string {
	t.Helper()

	payload, ok := event["issue"].(map[string]any)
	require.True(t, ok)
	title, ok := payload["title"].(string)
	require.True(t, ok)

	return title
}

func Test__SentryIssueEvents(t *testing.T) {
	now := time.Now().UTC()
	issues := []sentry.Issue{
		{ID: "1", Title: "Newest timeout", LastSeen: now.Format(time.RFC3339)},
		{ID: "2", Title: "Older null pointer", LastSeen: now.Add(-time.Hour).Format(time.RFC3339)},
	}

	events := sentryIssueEvents(nil, issues)
	require.Len(t, events, 2)
	assert.Equal(t, "created", events[0]["action"])
	assert.Equal(t, "issue", events[0]["resource"])

	firstData, ok := events[0]["data"].(map[string]any)
	require.True(t, ok)
	firstIssue, ok := firstData["issue"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Older null pointer", firstIssue["title"])
	assert.Equal(t, sentry.IssueDescription(firstIssue, nil), events[0]["description"])

	secondData, ok := events[1]["data"].(map[string]any)
	require.True(t, ok)
	secondIssue, ok := secondData["issue"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Newest timeout", secondIssue["title"])
	assert.Equal(t, sentry.IssueDescription(secondIssue, nil), events[1]["description"])
}

func Test__SentrySeedSkipsKnownIssues(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId: factoryModel.ID.String(),
		Source:    pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
	})
	require.NoError(t, err)
	canvasID := uuid.MustParse(response.GetIntake().GetCanvasId())
	tx := database.DB(t.Context())

	createOriginOrder := func(issueID, title, state, result string) {
		t.Helper()
		origin := models.WorkOrderOrigin{
			URL:   "https://acme.sentry.io/issues/" + issueID + "/",
			Label: title,
		}
		order, createErr := factoryModel.CreateWorkOrderWithOrigin(tx, title, "", nil, nil, nil, origin)
		require.NoError(t, createErr)
		switch {
		case state == models.FactoryWorkOrderStateDraft:
			return
		case state == models.FactoryWorkOrderStateClosed && result == models.FactoryWorkOrderResultRejected:
			_, updateErr := order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
				ToState: models.FactoryWorkOrderStateClosed,
				Result:  models.FactoryWorkOrderResultRejected,
			})
			require.NoError(t, updateErr)
		case state == models.FactoryWorkOrderStateClosed:
			_, updateErr := order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
				ToState: models.FactoryWorkOrderStateOpen,
			})
			require.NoError(t, updateErr)
			_, closeErr := order.Close(tx, result, nil)
			require.NoError(t, closeErr)
		}
	}

	createOriginOrder("11", "Draft timeout", models.FactoryWorkOrderStateDraft, "")
	createOriginOrder("22", "Closed timeout", models.FactoryWorkOrderStateClosed, models.FactoryWorkOrderResultCompleted)
	createOriginOrder("33", "Archived timeout", models.FactoryWorkOrderStateClosed, models.FactoryWorkOrderResultRejected)

	queued := []sentry.Issue{{
		ID:        "44",
		Title:     "Queued timeout",
		Permalink: "https://acme.sentry.io/issues/44/",
	}}
	first, err := seedKnownSentryIssues(tx, canvasID, nil, queued)
	require.NoError(t, err)
	assert.Equal(t, 1, first.itemCount)

	before, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	require.Len(t, before, 1)

	result, err := seedKnownSentryIssues(tx, canvasID, nil, []sentry.Issue{
		{ID: "11", Title: "Draft timeout", Permalink: "https://acme.sentry.io/issues/11/"},
		{ID: "22", Title: "Closed timeout", Permalink: "https://acme.sentry.io/issues/22/"},
		{ID: "33", Title: "Archived timeout", Permalink: "https://acme.sentry.io/issues/33/"},
		{ID: "44", Title: "Queued timeout", Permalink: "https://acme.sentry.io/issues/44/"},
		{ID: "55", Title: "Fresh timeout", Permalink: "https://acme.sentry.io/issues/55/"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.itemCount)

	after, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	require.Len(t, after, 2)

	freshIDs := map[string]bool{}
	for i := range after {
		issueID, ok := sentry.IssueIDFromEventData(after[i].Data.Data())
		require.True(t, ok)
		freshIDs[issueID] = true
	}
	assert.Equal(t, map[string]bool{"44": true, "55": true}, freshIDs)

	repeat, err := seedKnownSentryIssues(tx, canvasID, nil, []sentry.Issue{
		{ID: "11", Title: "Draft timeout"},
		{ID: "22", Title: "Closed timeout"},
		{ID: "33", Title: "Archived timeout"},
		{ID: "44", Title: "Queued timeout"},
		{ID: "55", Title: "Fresh timeout"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, repeat.itemCount)

	finalEvents, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	assert.Len(t, finalEvents, 2)
}

func Test__JiraSeedSkipsKnownIssues(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId: factoryModel.ID.String(),
		Source:    pb.FactoryIntake_SOURCE_JIRA_ISSUES,
	})
	require.NoError(t, err)
	canvasID := uuid.MustParse(response.GetIntake().GetCanvasId())
	tx := database.DB(t.Context())
	siteURL := "https://acme.atlassian.net"

	createOriginOrder := func(issueKey, title, state, result string) {
		t.Helper()
		origin := models.WorkOrderOrigin{
			URL:   jira.IssueURL(siteURL, issueKey),
			Label: issueKey,
		}
		order, createErr := factoryModel.CreateWorkOrderWithOrigin(tx, title, "", nil, nil, nil, origin)
		require.NoError(t, createErr)
		switch {
		case state == models.FactoryWorkOrderStateDraft:
			return
		case state == models.FactoryWorkOrderStateClosed && result == models.FactoryWorkOrderResultRejected:
			_, updateErr := order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
				ToState: models.FactoryWorkOrderStateClosed,
				Result:  models.FactoryWorkOrderResultRejected,
			})
			require.NoError(t, updateErr)
		case state == models.FactoryWorkOrderStateClosed:
			_, updateErr := order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
				ToState: models.FactoryWorkOrderStateOpen,
			})
			require.NoError(t, updateErr)
			_, closeErr := order.Close(tx, result, nil)
			require.NoError(t, closeErr)
		}
	}

	load := func(issueKey string) (*jira.Issue, error) {
		return &jira.Issue{
			Key: issueKey,
			Fields: map[string]any{
				"summary": "Summary of " + issueKey,
			},
		}, nil
	}

	createOriginOrder("ENG-11", "Draft issue", models.FactoryWorkOrderStateDraft, "")
	createOriginOrder("ENG-22", "Closed issue", models.FactoryWorkOrderStateClosed, models.FactoryWorkOrderResultCompleted)
	createOriginOrder("ENG-33", "Archived issue", models.FactoryWorkOrderStateClosed, models.FactoryWorkOrderResultRejected)

	queued := []jira.IssueSearchHit{{Key: "ENG-44"}}
	first, err := seedKnownJiraIssues(tx, canvasID, load, queued, siteURL)
	require.NoError(t, err)
	assert.Equal(t, 1, first.itemCount)

	before, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	require.Len(t, before, 1)

	result, err := seedKnownJiraIssues(tx, canvasID, load, []jira.IssueSearchHit{
		{Key: "ENG-11"},
		{Key: "ENG-22"},
		{Key: "ENG-33"},
		{Key: "ENG-44"},
		{Key: "ENG-55"},
	}, siteURL)
	require.NoError(t, err)
	assert.Equal(t, 1, result.itemCount)

	after, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	require.Len(t, after, 2)

	freshKeys := map[string]bool{}
	for i := range after {
		issueKey, ok := jira.IssueKeyFromEventData(after[i].Data.Data())
		require.True(t, ok)
		freshKeys[issueKey] = true
	}
	assert.Equal(t, map[string]bool{"ENG-44": true, "ENG-55": true}, freshKeys)

	repeat, err := seedKnownJiraIssues(tx, canvasID, load, []jira.IssueSearchHit{
		{Key: "ENG-11"},
		{Key: "ENG-22"},
		{Key: "ENG-33"},
		{Key: "ENG-44"},
		{Key: "ENG-55"},
	}, siteURL)
	require.NoError(t, err)
	assert.Equal(t, 0, repeat.itemCount)

	finalEvents, err := models.ListCanvasEvents(tx, canvasID, intakeTriggerNodeID, 20, nil)
	require.NoError(t, err)
	assert.Len(t, finalEvents, 2)
}

package factories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestGitHubReplaySelectionAndLimits(t *testing.T) {
	r, factory := resetTestFactory(t)
	defer r.Close()
	db := database.DB(t.Context())
	id := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	request := intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Settings: defaultIntakeSettings(), Binding: &intakeBinding{
		Integration: &yaml.IntegrationRef{ID: id}, Configuration: map[string]any{"repository": "acme/app"},
	}}
	doc, err := buildIntakeCanvas(request)
	require.NoError(t, err)
	nodes, edges, err := applyDefaultAuthorAccess(request, doc.Nodes(), doc.Edges())
	require.NoError(t, err)
	// Historical gate and live trigger identities must not depend on fixed IDs.
	renames := map[string]string{intakeTriggerNodeID: "legacy-trigger", intakeAuthorPermissionNodeID: "legacy-permission", intakeAuthorFilterNodeID: "legacy-access"}
	for i := range nodes {
		if renamed := renames[nodes[i].ID]; renamed != "" {
			nodes[i].ID = renamed
		}
	}
	for i := range edges {
		if renamed := renames[edges[i].SourceID]; renamed != "" {
			edges[i].SourceID = renamed
		}
		if renamed := renames[edges[i].TargetID]; renamed != "" {
			edges[i].TargetID = renamed
		}
	}
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
	publishGitHubTestGraph(t, r, canvas, nodes, edges)
	intake, err := factory.CreateIntake(db, canvas.ID, request.Source)
	require.NoError(t, err)
	emit := func(number int, repository string) models.CanvasEvent {
		payload, err := gitHubIssueEvent(&github.Issue{Number: github.Ptr(number), Title: github.Ptr("Issue"), User: &github.User{Login: github.Ptr("community")}, HTMLURL: github.Ptr(fmt.Sprintf("https://github.com/%s/issues/%d", repository, number))}, repository)
		require.NoError(t, err)
		require.NoError(t, emitIntakeEvents(db, canvas.ID, "github.issue", []map[string]any{payload}))
		events, err := models.ListCanvasEvents(db, canvas.ID, "legacy-trigger", 1, nil)
		require.NoError(t, err)
		return events[0]
	}
	for number := 1; number <= 25; number++ {
		event := emit(number, "acme/app")
		executeGitHubIntakeRun(t, r, canvas, &event, gitHubTestResponses(`{"permission":"none"}`))
	}
	_ = emit(25, "acme/app") // Newest pending event suppresses the older drop.
	wrong := emit(26, "acme/other")
	executeGitHubIntakeRun(t, r, canvas, &wrong, gitHubTestResponses(`{"permission":"none"}`))
	unrelated := emit(27, "acme/app")
	executeGitHubIntakeRun(t, r, canvas, &unrelated, gitHubTestResponses(`{"permission":"none"}`))
	require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("run_id = ? AND node_id = ?", unrelated.RunID, intakeFilterNodeID).Update("result", models.CanvasNodeExecutionResultFailed).Error)
	failedPermission := emit(28, "acme/app")
	executeGitHubIntakeRun(t, r, canvas, &failedPermission, gitHubTestResponses(`invalid`))
	eligible, err := gitHubIssueRunWasSkipped(db, failedPermission)
	require.NoError(t, err)
	require.True(t, eligible, "failed author lookup can be reconsidered")
	// Suppress this eligible failure with a newer pending event for the batch.
	_ = emit(28, "acme/app")
	active := emit(29, "acme/app")
	executeGitHubIntakeRun(t, r, canvas, &active, gitHubTestResponses(`{"permission":"read"}`))
	active = emit(29, "acme/app")
	executeGitHubIntakeRun(t, r, canvas, &active, gitHubTestResponses(`{"permission":"none"}`))
	host := gitHubTestResponses()
	for number := 24; number >= 5; number-- {
		body := fmt.Sprintf(`{"number":%d,"html_url":"https://github.com/acme/app/issues/%d","title":"Fresh","state":"open"}`, number, number)
		switch number {
		case 24:
			body = `invalid`
		case 23:
			body = `{"state":"closed"}`
		case 22:
			body = `{"state":"open","pull_request":{"url":"https://api.github.com/repos/acme/app/pulls/22"}}`
		}
		response := gitHubTestResponses(body)
		host.Responses = append(host.Responses, response.Responses...)
	}
	client, err := common.NewClient(gitHubPermissionTestIntegration(), host)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), intakeGitHubReplayBudget)
	defer cancel()
	payloads, err := skippedGitHubIssueEvents(ctx, db, intake, client, "acme/app")
	require.NoError(t, err)
	require.Len(t, host.Requests, 20, "failed refreshes count toward the issue limit")
	require.Len(t, payloads, 17)
	require.Equal(t, "opened", payloads[0]["action"])
	for _, request := range host.Requests {
		require.NotContains(t, request.URL.Path, "/issues/25")
	}
	expired, stop := context.WithCancel(t.Context())
	stop()
	payloads, err = skippedGitHubIssueEvents(expired, db, intake, client, "acme/app")
	require.NoError(t, err)
	require.Empty(t, payloads)
	require.Len(t, host.Requests, 20)
	for _, table := range []string{"workflow_runs", "factory_work_orders"} {
		t.Run("timeout during "+table, func(t *testing.T) {
			host := gitHubTestResponses(`{"number":24,"html_url":"https://github.com/acme/app/issues/24","title":"Fresh","state":"open"}`)
			client, err := common.NewClient(gitHubPermissionTestIntegration(), host)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			callback := "test:replay_read_timeout"
			require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
				if len(host.Requests) == 1 && tx.Statement.Table == table {
					<-ctx.Done()
				}
			}))
			defer func() { require.NoError(t, db.Callback().Query().Remove(callback)) }()
			payloads, err := skippedGitHubIssueEvents(ctx, db.WithContext(ctx), intake, client, "acme/app")
			require.NoError(t, err)
			require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			require.Len(t, payloads, 1, "a later database timeout must retain the refreshed issue")
			require.EqualValues(t, 24, payloads[0]["issue"].(map[string]any)["number"])
		})
	}
	// Fill the scan with newer pending subjects: older drops outside the newest
	// 200 events must not be reconsidered.
	for number := 100; number < 300; number++ {
		_ = emit(number, "acme/app")
	}
	payloads, err = skippedGitHubIssueEvents(t.Context(), db, intake, client, "acme/app")
	require.NoError(t, err)
	require.Empty(t, payloads)
	require.Len(t, host.Requests, 20)
	old := time.Now().Add(-intakeGitHubReplayWindow - time.Hour)
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Update("created_at", old).Error)
	retained, err := models.ListRecentCanvasRootEvents(db, canvas.ID, "github.issue", time.Now().Add(-intakeGitHubReplayWindow), intakeGitHubReplayEventLimit)
	require.NoError(t, err)
	require.Empty(t, retained)
}

func TestGitHubReplaySkipsPausedIntake(t *testing.T) {
	r, factory := resetTestFactory(t)
	defer r.Close()
	db := database.DB(t.Context())
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
	intake, err := factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
	require.NoError(t, err)
	stored, err := factory.FindIntake(db, intake.ID)
	require.NoError(t, err)
	require.NoError(t, stored.SetPaused(db, true))
	// The caller's intake is stale. Read the saved pause state before any GitHub setup.
	require.NoError(t, replaySkippedGitHubIssues(t.Context(), IntakeDependencies{}, db, intake))
}

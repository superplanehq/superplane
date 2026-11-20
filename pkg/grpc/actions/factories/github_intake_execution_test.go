package factories

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func gitHubPermissionTestIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{NewSetupFlow: true,
		CurrentProperties: map[string]any{"authMethod": "pat", "owner": "acme", "ownerType": "Organization"},
		CurrentSecrets:    map[string]core.IntegrationSecret{"pat": {Name: "pat", Value: []byte("test-token")}},
	}
}

func gitHubTestResponses(bodies ...string) *contexts.HTTPContext {
	host := &contexts.HTTPContext{}
	for _, body := range bodies {
		host.Responses = append(host.Responses, &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))})
	}
	return host
}

func publishGitHubTestGraph(t *testing.T, r *support.ResourceRegistry, canvas *models.Canvas, nodes []models.Node, edges []models.Edge) {
	t.Helper()
	encoded, err := yaml.VersionToCanvasYAML(canvas.Name, "GitHub intake", &models.CanvasVersion{Nodes: nodes, Edges: edges})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(encoded))
	require.NoError(t, err)
	resetTestPublishCanvas(t, r, canvas, doc)
}

// Execute the published components with production expression, state, and task
// contexts. Follow their real output channels rather than synthesizing drops.
func executeGitHubIntakeRun(t *testing.T, r *support.ResourceRegistry, canvas *models.Canvas, root *models.CanvasEvent, host core.HTTPContext) {
	t.Helper()
	db := database.DB(t.Context())
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, root.RunID)
	require.NoError(t, err)
	version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, run.VersionID)
	require.NoError(t, err)
	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
	nodeID := graph.FilterNodeID
	incoming := root
	result := models.CanvasRunResultPassed
	for nodeID != "" {
		node := findIntakeNode(spec.Nodes, nodeID)
		require.NotNil(t, node)
		action, err := r.Registry.GetAction(node.ComponentName())
		require.NoError(t, err)
		execution := support.CreateNextNodeExecution(t, canvas.ID, nodeID, root.ID, incoming.ID, incoming.ExecutionID)
		builder := workercontexts.NewNodeConfigurationBuilder(db, canvas.ID).WithRunID(run.ID).WithNodeID(nodeID).
			WithRootEvent(&root.ID).WithPreviousExecution(incoming.ExecutionID).WithIncomingEventID(&incoming.ID).
			WithInput(map[string]any{incoming.NodeID: incoming.Data.Data()}).WithConfigurationFields(action.Configuration())
		resolved, err := builder.Build(node.Configuration)
		require.NoError(t, err)
		execution.Configuration = datatypes.NewJSONType(resolved)
		require.NoError(t, db.Save(execution).Error)
		err = action.Execute(core.ExecutionContext{Configuration: resolved, HTTP: host,
			Integration: gitHubPermissionTestIntegration(), Metadata: &contexts.MetadataContext{},
			Logger: log.NewEntry(log.StandardLogger()), Expressions: workercontexts.NewExpressionContext(builder),
			ExecutionState: workercontexts.NewExecutionStateContext(db, execution, nil),
			Factory:        workercontexts.NewFactoryContext(db, canvas, execution),
		})
		if err != nil {
			t.Logf("execution %s failed: %v", nodeID, err)
			result = models.CanvasRunResultFailed
			require.NoError(t, db.Model(execution).Updates(map[string]any{"state": models.CanvasNodeExecutionStateFinished, "result": models.CanvasNodeExecutionResultFailed}).Error)
			break
		}
		var outputs []models.CanvasEvent
		require.NoError(t, db.Where("execution_id = ?", execution.ID).Find(&outputs).Error)
		if len(outputs) == 0 {
			break // The task duplicate guard passes without an output.
		}
		require.Len(t, outputs, 1)
		incoming = &outputs[0]
		nodeID = ""
		for _, edge := range spec.Edges {
			if edge.SourceID == incoming.NodeID && edge.Channel == incoming.Channel {
				nodeID = edge.TargetID
				break
			}
		}
	}
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("run_id = ?", run.ID).Update("state", models.CanvasEventStateRouted).Error)
	require.NoError(t, db.Model(run).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": result}).Error)
}

func TestGitHubIntakeActorApproval(t *testing.T) {
	for _, tc := range []struct {
		name, action, author, sender string
		wantTask                     bool
	}{
		{"maintainer labels community issue", "labeled", "read", "write", true},
		{"triage sender labels community issue", "labeled", "read", "triage", true},
		{"maintainer reopens community issue", "reopened", "read", "admin", true},
		{"author remains sufficient", "labeled", "triage", "read", true},
		{"community author and actor", "labeled", "read", "read", false},
		{"opened cannot use actor", "opened", "read", "write", false},
		{"missing sender", "reopened", "read", "", false},
		{"author error actor approves", "labeled", "invalid", "write", true},
		{"lookup failure cannot approve", "labeled", "invalid", "none", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			defer r.Close()
			db := database.DB(t.Context())
			integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			request := intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Settings: defaultIntakeSettings(), Binding: &intakeBinding{
				Integration: &yaml.IntegrationRef{ID: integrationID}, Configuration: map[string]any{"repository": "acme/app"},
			}}
			request.Settings.Labels = []string{"superplane"}
			request.Settings = deriveGitHubIssueListening(request.Settings)
			doc, err := buildIntakeCanvas(request)
			require.NoError(t, err)
			nodes, edges, err := applyDefaultAuthorAccess(request, doc.Nodes(), doc.Edges())
			require.NoError(t, err)
			canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
			publishGitHubTestGraph(t, r, canvas, nodes, edges)
			issue := &github.Issue{Number: github.Ptr(42), HTMLURL: github.Ptr("https://github.com/acme/app/issues/42"),
				Title: github.Ptr("Community issue"), State: github.Ptr("open"), User: &github.User{Login: github.Ptr("community")}, Labels: []*github.Label{{Name: github.Ptr("superplane")}}}
			payload, err := gitHubIssueEvent(issue, "acme/app")
			require.NoError(t, err)
			payload["action"] = tc.action
			payload["label"] = map[string]any{"name": "superplane"}
			if tc.sender != "" {
				payload["sender"] = map[string]any{"login": "maintainer"}
			}
			require.NoError(t, emitIntakeEvents(db, canvas.ID, "github.issue", []map[string]any{payload}))
			events, err := models.ListRecentCanvasRootEvents(db, canvas.ID, "github.issue", *canvas.CreatedAt, 1)
			require.NoError(t, err)
			require.Len(t, events, 1)
			permissionBody := func(permission string) string {
				if permission == "invalid" {
					return "invalid"
				}
				if permission == "triage" {
					return `{"permission":"read","role_name":"triage"}`
				}
				return fmt.Sprintf(`{"permission":%q}`, permission)
			}
			executeGitHubIntakeRun(t, r, canvas, &events[0], gitHubTestResponses(permissionBody(tc.author), permissionBody(tc.sender)))
			orders, err := models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{events[0].RunID})
			require.NoError(t, err)
			require.Equal(t, tc.wantTask, len(orders) == 1)
		})
	}
}

func TestGitHubIntakeSettingsSaveRecoversAfterCommit(t *testing.T) {
	r, factory := resetTestFactory(t)
	defer r.Close()
	db := database.DB(t.Context())
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	request := intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Settings: defaultIntakeSettings(), Binding: &intakeBinding{
		Integration: &yaml.IntegrationRef{ID: integrationID}, Configuration: map[string]any{"repository": "acme/app"},
	}}
	doc, err := buildIntakeCanvas(request)
	require.NoError(t, err)
	nodes, edges, err := applyDefaultAuthorAccess(request, doc.Nodes(), doc.Edges())
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
	publishGitHubTestGraph(t, r, canvas, nodes, edges)
	intake, err := factory.CreateIntake(db, canvas.ID, request.Source)
	require.NoError(t, err)
	payload, err := gitHubIssueEvent(&github.Issue{Number: github.Ptr(42), HTMLURL: github.Ptr("https://github.com/acme/app/issues/42"), Title: github.Ptr("Old title"), User: &github.User{Login: github.Ptr("community")}}, "acme/app")
	require.NoError(t, err)
	require.NoError(t, emitIntakeEvents(db, canvas.ID, "github.issue", []map[string]any{payload}))
	roots, err := models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 1, nil)
	require.NoError(t, err)
	executeGitHubIntakeRun(t, r, canvas, &roots[0], gitHubTestResponses(`{"permission":"read"}`))
	orders, err := models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{roots[0].RunID})
	require.NoError(t, err)
	require.Empty(t, orders)

	host := gitHubTestResponses(`{"number":42,"html_url":"https://github.com/acme/app/issues/42","title":"Refreshed title","state":"open","user":{"login":"community"}}`)
	client, err := common.NewClient(gitHubPermissionTestIntegration(), host)
	require.NoError(t, err)
	release := make(chan struct{}, 1)
	defer close(release)
	deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000",
		NewItemSource: func(ctx context.Context, _ *gorm.DB, _ *models.FactoryIntake) (intakeItemSource, error) {
			select {
			case <-release:
				return &gitHubIntakeItemSource{github: client, repository: "acme/app"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	ctx, cancel := context.WithCancel(authentication.SetUserIdInMetadata(context.Background(), r.User.String()))
	defer cancel()
	saved := make(chan error, 1)
	go func() {
		_, err := UpdateFactoryIntake(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(), IntakeId: intake.ID.String(), Settings: &pb.FactoryIntake_Settings{AuthorsWithAccess: false},
		})
		saved <- err
	}()
	select {
	case err := <-saved:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("settings save waited for recovery")
	}
	cancel() // A completed request must not cancel the background GitHub read.
	version := resetTestLiveVersion(t, canvas)
	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	require.False(t, intakeSettingsFromGraph(request.Source, resolveIntakeGraph(request.Source, spec), spec).AuthorsWithAccess)
	release <- struct{}{}
	require.Eventually(t, func() bool {
		roots, err = models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 10, nil)
		return err == nil && len(roots) == 2
	}, 5*time.Second, 10*time.Millisecond)
	executeGitHubIntakeRun(t, r, canvas, &roots[0], gitHubTestResponses())
	orders, err = models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{roots[0].RunID})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "Refreshed title", orders[roots[0].RunID].Title)
}

func TestGitHubIntakeDisableRecoversSkippedRun(t *testing.T) {
	r, factory := resetTestFactory(t)
	defer r.Close()
	db := database.DB(t.Context())
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	request := intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Settings: defaultIntakeSettings(), Binding: &intakeBinding{
		Integration: &yaml.IntegrationRef{ID: integrationID}, Configuration: map[string]any{"repository": "acme/app"},
	}}
	doc, err := buildIntakeCanvas(request)
	require.NoError(t, err)
	nodes, edges, err := applyDefaultAuthorAccess(request, doc.Nodes(), doc.Edges())
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
	publishGitHubTestGraph(t, r, canvas, nodes, edges)
	intake, err := factory.CreateIntake(db, canvas.ID, request.Source)
	require.NoError(t, err)
	payload, err := gitHubIssueEvent(&github.Issue{Number: github.Ptr(42), HTMLURL: github.Ptr("https://github.com/acme/app/issues/42"), Title: github.Ptr("Old title"), User: &github.User{Login: github.Ptr("community")}}, "acme/app")
	require.NoError(t, err)
	require.NoError(t, emitIntakeEvents(db, canvas.ID, "github.issue", []map[string]any{payload}))
	events, err := models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 1, nil)
	require.NoError(t, err)
	executeGitHubIntakeRun(t, r, canvas, &events[0], gitHubTestResponses(`{"permission":"read"}`))
	orders, err := models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{events[0].RunID})
	require.NoError(t, err)
	require.Empty(t, orders)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	// This integration cannot read GitHub. Recovery failure must not fail save.
	response, err := UpdateFactoryIntake(ctx, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}, r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), IntakeId: intake.ID.String(), Settings: &pb.FactoryIntake_Settings{AuthorsWithAccess: false, Labels: []string{"superplane"}, NewIssues: github.Ptr(true), ReopenedIssues: github.Ptr(true)},
	})
	require.NoError(t, err)
	require.False(t, response.Intake.Settings.AuthorsWithAccess)
	host := gitHubTestResponses(`{"number":42,"html_url":"https://github.com/acme/app/issues/42","title":"Refreshed title","state":"open","labels":[{"name":"superplane"}],"assignees":[{"login":"maintainer"}],"user":{"login":"community"}}`)
	client, err := common.NewClient(gitHubPermissionTestIntegration(), host)
	require.NoError(t, err)
	refreshed, err := skippedGitHubIssueEvents(t.Context(), db, intake, client, "acme/app")
	require.NoError(t, err)
	require.Len(t, refreshed, 1)
	require.Nil(t, refreshed[0]["sender"])
	issue := refreshed[0]["issue"].(map[string]any)
	require.Equal(t, "maintainer", issue["assignees"].([]any)[0].(map[string]any)["login"])
	stored, err := factory.FindIntake(db, intake.ID)
	require.NoError(t, err)
	require.NoError(t, stored.SetPaused(db, true))
	// A pause saved during the refresh must block emission, even with a stale intake.
	require.NoError(t, emitRecoveredGitHubIssueEvents(db, intake, refreshed))
	roots, err := models.ListRecentCanvasRootEvents(db, canvas.ID, "github.issue", *canvas.CreatedAt, 10)
	require.NoError(t, err)
	require.Len(t, roots, 1)
	require.NoError(t, stored.SetPaused(db, false))
	require.NoError(t, emitRecoveredGitHubIssueEvents(db, intake, refreshed))
	events, err = models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 1, nil)
	require.NoError(t, err)
	version := resetTestLiveVersion(t, canvas)
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, events[0].RunID)
	require.NoError(t, err)
	require.Equal(t, version.ID, run.VersionID)
	executeGitHubIntakeRun(t, r, canvas, &events[0], gitHubTestResponses())
	orders, err = models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{events[0].RunID})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "Refreshed title", orders[events[0].RunID].Title)
	refreshed, err = skippedGitHubIssueEvents(t.Context(), db, intake, client, "acme/app")
	require.NoError(t, err)
	require.Empty(t, refreshed)
}

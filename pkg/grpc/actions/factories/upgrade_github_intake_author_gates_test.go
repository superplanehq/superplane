package factories

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
)

func TestUpgradeGitHubIntakeAuthorGates(t *testing.T) {
	for _, expression := range []string{
		`root().data.permission != "none"`,
		`previous().data.permission != "none"`,
		`previous().data.permission in ["read", "triage", "write", "maintain", "admin"]`,
		"disabled",
	} {
		t.Run(expression, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			defer r.Close()
			db := database.DB(t.Context())
			integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			request := intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Settings: defaultIntakeSettings(), Binding: &intakeBinding{
				Integration: &yaml.IntegrationRef{ID: integrationID}, Configuration: map[string]any{"repository": "acme/app"},
			}}
			request.Settings.AuthorsWithAccess = expression != "disabled"
			request.Settings.Labels = []string{"bug"}
			request.Settings = deriveGitHubIssueListening(request.Settings)
			doc, err := buildIntakeCanvas(request)
			require.NoError(t, err)
			nodes, edges, err := applyDefaultAuthorAccess(request, doc.Nodes(), doc.Edges())
			require.NoError(t, err)
			for i := range nodes {
				switch nodes[i].ID {
				case intakeAuthorPermissionNodeID:
					nodes[i].ID = "legacy-permission"
					delete(nodes[i].Configuration, "fallbackUsername")
				case intakeAuthorFilterNodeID:
					nodes[i].ID = "legacy-access"
					nodes[i].Configuration["expression"] = expression
				}
			}
			for i := range edges {
				for old, renamed := range map[string]string{intakeAuthorPermissionNodeID: "legacy-permission", intakeAuthorFilterNodeID: "legacy-access"} {
					if edges[i].SourceID == old {
						edges[i].SourceID = renamed
					}
					if edges[i].TargetID == old {
						edges[i].TargetID = renamed
					}
				}
			}
			canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
			publishGitHubTestGraph(t, r, canvas, nodes, edges)
			intake, err := factory.CreateIntake(db, canvas.ID, request.Source)
			require.NoError(t, err)
			require.NoError(t, intake.SetPaused(db, true))
			before := resetTestLiveVersion(t, canvas)
			beforeSpec := models.LiveCanvasSpec{Nodes: before.Nodes, Edges: before.Edges}
			settings := intakeSettingsFromGraph(request.Source, resolveIntakeGraph(request.Source, beforeSpec), beforeSpec)
			deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
			require.NoError(t, UpgradeGitHubIntakeAuthorGates(t.Context(), deps))
			after := resetTestLiveVersion(t, canvas)
			afterSpec := models.LiveCanvasSpec{Nodes: after.Nodes, Edges: after.Edges}
			require.Equal(t, settings, intakeSettingsFromGraph(request.Source, resolveIntakeGraph(request.Source, afterSpec), afterSpec))
			require.Equal(t, before.Edges, after.Edges)
			stored, err := factory.FindIntake(db, intake.ID)
			require.NoError(t, err)
			require.True(t, stored.Paused())
			roots, err := models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 10, nil)
			require.NoError(t, err)
			require.Empty(t, roots, "startup upgrade must not replay issues")
			require.NoError(t, UpgradeGitHubIntakeAuthorGates(t.Context(), deps))
			require.Equal(t, after.ID, resetTestLiveVersion(t, canvas).ID, "upgrade is idempotent")
			if expression == "disabled" {
				require.Equal(t, before.ID, after.ID, "disabled gates stay disabled")
				return
			}
			require.NotEqual(t, before.ID, after.ID)
			require.NoError(t, stored.SetPaused(db, false))
			payload, err := gitHubIssueEvent(&github.Issue{Number: github.Ptr(42), HTMLURL: github.Ptr("https://github.com/acme/app/issues/42"), Title: github.Ptr("Community issue"), State: github.Ptr("open"), User: &github.User{Login: github.Ptr("community")}, Labels: []*github.Label{{Name: github.Ptr("bug")}}}, "acme/app")
			require.NoError(t, err)
			for _, action := range []string{"opened", "labeled"} {
				payload["action"] = action
				payload["label"] = map[string]any{"name": "bug"}
				payload["sender"] = map[string]any{"login": "maintainer"}
				require.NoError(t, emitIntakeEvents(db, canvas.ID, "github.issue", []map[string]any{payload}))
				roots, err := models.ListCanvasEvents(db, canvas.ID, intakeTriggerNodeID, 1, nil)
				require.NoError(t, err)
				executeGitHubIntakeRun(t, r, canvas, &roots[0], gitHubTestResponses(`{"permission":"read","role_name":"read"}`, `{"permission":"read","role_name":"triage"}`))
				orders, err := models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{roots[0].RunID})
				require.NoError(t, err)
				require.Equal(t, action == "labeled", len(orders) == 1, "upgraded gate rejects outsiders until an authorized sender approves")
			}
		})
	}
}

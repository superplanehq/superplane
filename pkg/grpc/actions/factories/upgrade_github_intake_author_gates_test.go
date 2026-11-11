package factories

import (
	"slices"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

func TestUpgradeGitHubIntakeAuthorGatesContinuesAfterInvalidIntake(t *testing.T) {
	for _, failure := range []string{"missing permission node", "missing filter node", "missing live version"} {
		t.Run(failure, func(t *testing.T) {
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
			for i := range nodes {
				if nodes[i].ID == intakeAuthorPermissionNodeID {
					delete(nodes[i].Configuration, "fallbackUsername")
				}
				if nodes[i].ID == intakeAuthorFilterNodeID {
					nodes[i].Configuration["expression"] = `root().data.permission != "none"`
				}
			}
			var canvases []*models.Canvas
			var versions []*models.CanvasVersion
			for _, id := range []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"} {
				canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub")
				publishGitHubTestGraph(t, r, canvas, nodes, edges)
				intake, err := factory.CreateIntake(db, canvas.ID, request.Source)
				require.NoError(t, err)
				// Fix scan order so the invalid intake always precedes the valid one.
				require.NoError(t, db.Model(intake).Update("id", uuid.MustParse(id)).Error)
				canvases = append(canvases, canvas)
				versions = append(versions, resetTestLiveVersion(t, canvas))
			}
			if failure == "missing live version" {
				require.NoError(t, db.Model(canvases[0]).Update("live_version_id", versions[1].ID).Error)
			} else {
				missing := intakeAuthorPermissionNodeID
				if failure == "missing filter node" {
					missing = intakeAuthorFilterNodeID
				}
				versions[0].Nodes = slices.DeleteFunc(versions[0].Nodes, func(node models.Node) bool { return node.ID == missing })
				require.NoError(t, db.Model(versions[0]).Update("nodes", versions[0].Nodes).Error)
			}
			deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
			hook := captureGlobalLogEntries(t)
			require.NoError(t, UpgradeGitHubIntakeAuthorGates(t.Context(), deps))
			warnings := slices.DeleteFunc(hook.AllEntries(), func(entry *log.Entry) bool { return entry.Level != log.WarnLevel })
			require.Len(t, warnings, 1, "invalid intake is logged once")
			require.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), warnings[0].Data["intake_id"])
			require.Equal(t, r.Organization.ID, warnings[0].Data["organization_id"])
			require.Equal(t, canvases[0].ID, warnings[0].Data["canvas_id"])
			if failure == "missing live version" {
				stored, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvases[0].ID)
				require.NoError(t, err)
				require.Equal(t, versions[1].ID, *stored.LiveVersionID)
			} else {
				require.Equal(t, versions[0].ID, resetTestLiveVersion(t, canvases[0]).ID, "incomplete intake stays unchanged")
			}
			after := resetTestLiveVersion(t, canvases[1])
			require.NotEqual(t, versions[1].ID, after.ID, "later intake is still upgraded")
			require.Equal(t, intakeAuthorPermissionCondition, findModelNode(t, after.Nodes, intakeAuthorFilterNodeID).Configuration["expression"])
			require.Equal(t, intakeAuthorFallbackUsername, findModelNode(t, after.Nodes, intakeAuthorPermissionNodeID).Configuration["fallbackUsername"])
		})
	}
}

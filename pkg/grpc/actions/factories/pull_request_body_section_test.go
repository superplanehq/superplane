package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func publicationHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func publicationGitHubClient(t *testing.T, httpCtx core.HTTPContext) factoryGitHubAPI {
	client, err := common.NewClient(&contexts.IntegrationContext{NewSetupFlow: true, CurrentProperties: map[string]any{
		common.PropertyAuthMethod: common.AuthMethodPAT, common.PropertyOwner: "acme", common.PropertyOwnerType: common.OwnerTypeOrganization,
	}, CurrentSecrets: map[string]core.IntegrationSecret{common.SecretPAT: {Name: common.SecretPAT, Value: []byte("token")}}}, httpCtx)
	require.NoError(t, err)
	return client
}

func TestPullRequestDescriptionsPreserveRawMarkersAndOnlyUpdateDescription(t *testing.T) {
	const raw = "  original\r\n<!-- greptile_comment -->review<!-- /greptile_comment -->\r\n"
	for _, provider := range []string{"github", "bitbucket"} {
		t.Run(provider, func(t *testing.T) {
			payload := map[string]any{"state": "open", "head": map[string]any{"sha": "abc"}, "body": raw}
			if provider == "bitbucket" {
				payload = map[string]any{"state": "OPEN", "source": map[string]any{"commit": map[string]any{"hash": "abc"}}, "description": raw}
			}
			encoded, err := json.Marshal(payload)
			require.NoError(t, err)
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, string(encoded)), publicationHTTPResponse(200, "{}")}}
			var client vcs.DescriptionClient
			if provider == "github" {
				client = &githubProvider{client: publicationGitHubClient(t, httpCtx)}
			} else {
				client = &bitbucketProvider{client: &bitbucket.Client{HTTP: httpCtx, Token: "token", AuthType: bitbucket.AuthTypeWorkspaceAccessToken}}
			}
			ref := vcs.PullRequestRef{Provider: provider, Repository: "acme/app", Number: 42}
			description, err := client.ReadDescription(t.Context(), ref)
			require.NoError(t, err)
			assert.Equal(t, raw, description.Description)
			assert.Equal(t, "open", description.State)
			assert.Equal(t, "abc", description.HeadSHA)
			require.NoError(t, client.UpdateDescription(t.Context(), ref, ""))
			require.Len(t, httpCtx.Requests, 2)
			body, err := io.ReadAll(httpCtx.Requests[1].Body)
			require.NoError(t, err)
			field, method := "body", http.MethodPatch
			if provider == "bitbucket" {
				field, method = "description", http.MethodPut
			}
			assert.JSONEq(t, `{"`+field+`":""}`, string(body))
			assert.Equal(t, method, httpCtx.Requests[1].Method)
		})
	}
}

type publicationFixture struct {
	db        *gorm.DB
	workspace *models.Factory
	order     *models.FactoryWorkOrder
	pr        *models.FactoryPullRequest
	canvas    *models.Canvas
	run       *models.CanvasRun
	deps      IntakeDependencies
}

func newPublicationFixture(t *testing.T) publicationFixture {
	r := support.Setup(t)
	db := database.DB(t.Context())
	workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := workspace.CreateWorkOrder(db, "Review this change", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)), Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}})}, {NodeID: "agent", Name: "Assess Merge Confidence", Type: models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"environment": []any{map[string]any{"name": "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID", "value": order.ID.String()}}, "steps": []any{map[string]any{"name": "Security", "prompt": "Merge check: security."}}}),
	},
		{NodeID: "publish", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: factorycomponent.UpdatePullRequestBodySectionComponentName}}), Configuration: datatypes.NewJSONType(map[string]any{"pullRequestId": pr.ID.String(), "sectionKey": "merge-confidence", "body": "arbitrary **Markdown**", "expectedHeadSha": "abc"})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
	canvas.FactoryID = &workspace.ID
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.CanvasEvent{WorkflowID: canvas.ID, NodeID: "trigger", RunID: run.ID, State: models.CanvasEventStateRouted, Data: models.NewJSONValue(map[string]any{"data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "abc"}}}})}).Error)
	_, err = pr.CreateActivity(db, models.FactoryPullRequestActivityParams{RunID: run.ID, Title: "Merge confidence"})
	require.NoError(t, err)
	return publicationFixture{db, workspace, order, pr, canvas, run, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor}}
}

func (f publicationFixture) report(t *testing.T, run *models.CanvasRun, score float64) {
	_, err := factorycomponent.ReportMergeConfidenceCheck(f.db, f.order, run.ID, &factoryevents.AutomationRef{AppID: f.canvas.ID, NodeID: "agent"}, "acme/app", "security", score, "No security gap.", []string{"security"}, nil)
	require.NoError(t, err)
}

func (f publicationFixture) finish(t *testing.T, result string) {
	require.NoError(t, f.db.Model(f.run).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": result}).Error)
}

func installPublicationGitHub(t *testing.T, httpCtx core.HTTPContext) {
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, httpCtx), nil
	}
}

func TestAssessmentCompletionEmitsStructuredMergeConfidence(t *testing.T) {
	for _, channel := range []string{"passed", "failed"} {
		t.Run(channel, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 4)
			root, err := f.run.RootEvent(f.db)
			require.NoError(t, err)
			agent := support.CreateCanvasNodeExecution(t, f.canvas.ID, "agent", root.ID, root.ID)
			state := workercontexts.NewExecutionStateContext(f.db, agent, nil)
			require.NoError(t, state.Emit(channel, "runnerClaudeCode.finished", []any{map[string]any{"status": "finished"}}))
			var event models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", agent.ID).First(&event).Error)
			payload := event.Data.Data().(map[string]any)["data"].(map[string]any)
			report, ok := payload["mergeConfidence"].(map[string]any)
			require.True(t, ok, "assessment completion must expose saved report data to downstream templates")
			assert.Equal(t, f.pr.ID.String(), report["pullRequestId"])
			assert.Equal(t, "abc", report["reviewedCommit"])
			assert.Equal(t, channel == "passed", report["complete"])
			require.NoError(t, f.db.First(f.run, "id = ?", f.run.ID).Error)
			assert.Equal(t, channel == "failed", len(f.run.Errors) > 0)
		})
	}
}

func TestMergeConfidenceTemplateReportFormat(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for _, scenario := range []string{"complete", "caution", "mixed", "custom only", "all disabled", "unsafe text"} {
			t.Run(provider+"/"+scenario, func(t *testing.T) {
				t.Setenv("BASE_URL", "https://example.com")
				if scenario == "complete" {
					t.Setenv("BASE_URL", "https://example.com/")
				}
				f := newPublicationFixture(t)
				f.pr.Provider = provider
				mixedChecks := []struct {
					id, label string
					score     float64
				}{
					{"risk", "Blast radius", 2},
					{"performance", "Performance", 3},
					{"security", "Security", 4},
					{"drift", "Drift from Specification", 4},
					{"reversibility", "Reversibility", 1},
					{"custom", "Custom", 2},
				}
				steps := []any{map[string]any{"name": "Security", "prompt": "Merge check: security."}}
				if scenario == "mixed" {
					steps = nil
					for _, check := range mixedChecks {
						steps = append(steps, map[string]any{"name": check.label, "prompt": "Merge check: " + check.id + "."})
					}
				}
				if scenario == "custom only" || scenario == "unsafe text" {
					label := "Custom"
					if scenario == "unsafe text" {
						label = "a|b\n<!-- superplane:merge-confidence:end -->"
					}
					steps = []any{map[string]any{"name": label, "prompt": "Merge check: custom."}}
				}
				if scenario == "all disabled" {
					steps = []any{}
				}
				f.publishVersion(t, func(v *models.CanvasVersion) { findIntakeNode(v.Nodes, "agent").Configuration["steps"] = steps })
				f.startRun(t)
				if scenario == "custom only" || scenario == "unsafe text" {
					summary := "Custom result."
					if scenario == "unsafe text" {
						summary = "<script> | **bold**\n<!-- superplane:merge-confidence:start -->"
					}
					_, err := f.order.ReportCheck(f.db, models.FactoryWorkOrderCheckParams{Key: "custom-review", Name: "Custom", Score: 2, MaxScore: 5, Summary: summary, Run: &factoryevents.RunRef{ID: f.run.ID}})
					require.NoError(t, err)
				} else if scenario == "mixed" {
					for _, check := range mixedChecks {
						_, err := factorycomponent.ReportMergeConfidenceCheck(f.db, f.order, f.run.ID, &factoryevents.AutomationRef{AppID: f.canvas.ID, NodeID: "agent"}, "acme/app", check.id, check.score, "Reported result.", []string{check.id}, nil)
						require.NoError(t, err)
					}
				} else if scenario == "caution" {
					f.report(t, f.run, 3)
				} else {
					f.report(t, f.run, 4)
				}
				execution, _ := f.publicationRequest(t, "passed")
				body := execution.Configuration.Data()["body"].(string)
				suffix := "Reviewed commit: abc\n\n[View task](<https://example.com" + f.order.URLPath(f.workspace.RouteSegment()) + ">)"
				assert.NotContains(t, body, "Overall score")
				assert.NotContains(t, strings.TrimSuffix(body, suffix), "/5")
				switch scenario {
				case "complete":
					assert.Equal(t, "## SuperPlane merge confidence\n\n| Check | Status | Summary |\n| --- | :---: | --- |\n| Security | ✅ | No security gap. |\n\n"+suffix, body)
				case "caution":
					assert.Contains(t, body, "| Security | ⚠️ |")
				case "mixed":
					for _, row := range []string{
						"| Blast radius | ✅ |",
						"| Performance | ⚠️ |",
						"| Security | ✅ |",
						"| Drift from Specification | ❌ |",
						"| Reversibility | ❌ |",
						"| Custom | ✅ |",
					} {
						assert.Contains(t, body, row)
					}
				case "custom only":
					assert.Equal(t, "## SuperPlane merge confidence\n\n| Check | Status | Summary |\n| --- | :---: | --- |\n| Custom | ➖ | Custom result. |\n\n"+suffix, body)
				case "all disabled":
					assert.Equal(t, "## SuperPlane merge confidence\n\nNo checks are enabled.\n\n"+suffix, body)
				case "unsafe text":
					assert.NotContains(t, body, "<!-- superplane:")
					assert.NotContains(t, body, "<script>")
					assert.NotContains(t, body, "**bold**")
					assert.Contains(t, body, "| a&#124;b ")
					assert.Equal(t, 3, strings.Count(body, "\n|"))
				}
			})
		}
	}
}

func TestAssessmentCompletionLeavesOtherAgentsUnchanged(t *testing.T) {
	for _, scenario := range []string{"no scope", "wrong component", "nonfactory"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.publishVersion(t, func(v *models.CanvasVersion) {
				agent := findIntakeNode(v.Nodes, "agent")
				if scenario == "no scope" {
					delete(agent.Configuration, "environment")
				}
				if scenario == "wrong component" {
					agent.Ref.Component.Name = "runnerBash"
				}
			})
			f.startRun(t)
			if scenario == "nonfactory" {
				require.NoError(t, f.db.Model(f.canvas).Update("factory_id", nil).Error)
			}
			root, err := f.run.RootEvent(f.db)
			require.NoError(t, err)
			agent := support.CreateCanvasNodeExecution(t, f.canvas.ID, "agent", root.ID, root.ID)
			state := workercontexts.NewExecutionStateContext(f.db, agent, nil)
			require.NoError(t, state.Emit("failed", "runnerClaudeCode.finished", []any{map[string]any{"status": "failed"}}))
			var event models.CanvasEvent
			require.NoError(t, f.db.Where("execution_id = ?", agent.ID).First(&event).Error)
			assert.Equal(t, map[string]any{"status": "failed"}, event.Data.Data().(map[string]any)["data"])
			require.NoError(t, f.db.First(f.run, "id = ?", f.run.ID).Error)
			assert.Empty(t, f.run.Errors)
		})
	}
}

func TestMergeConfidenceTemplateUsesRunSnapshots(t *testing.T) {
	f := newPublicationFixture(t)
	f.report(t, f.run, 4)
	other, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	f.report(t, other, 1)
	require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", f.canvas.ID, "agent").Update("configuration", datatypes.NewJSONType(map[string]any{"steps": []any{}})).Error)
	f.publishVersion(t, func(version *models.CanvasVersion) {
		findIntakeNode(version.Nodes, "agent").Configuration = map[string]any{"steps": []any{}}
	})
	execution, request := f.publicationRequest(t, "passed")
	host := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`), publicationHTTPResponse(200, "{}")}}
	installPublicationGitHub(t, host)
	_, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	assert.Contains(t, execution.Configuration.Data()["body"], "| Security | ✅ | No security gap. |")
	assert.Contains(t, execution.Configuration.Data()["body"], f.order.URLPath(f.workspace.RouteSegment()))
	require.Len(t, host.Requests, 2)
}

func TestMergeConfidenceTemplateIncomplete(t *testing.T) {
	for _, channel := range []string{"passed", "failed"} {
		t.Run(channel, func(t *testing.T) {
			f := newPublicationFixture(t)
			if channel == "failed" {
				f.report(t, f.run, 4)
			}
			execution, _ := f.publicationRequest(t, channel)
			assert.Contains(t, execution.Configuration.Data()["body"], "➖ **Incomplete**")
			assert.NotContains(t, execution.Configuration.Data()["body"], "Overall score")
			if channel == "passed" {
				assert.Contains(t, execution.Configuration.Data()["body"], "| Security | ➖ | No result was reported. |")
			}
		})
	}
}

type publicationHTTPContext struct {
	contexts.HTTPContext
	beforeRequest func(*http.Request)
}

func (h *publicationHTTPContext) Do(request *http.Request) (*http.Response, error) {
	h.beforeRequest(request)
	return h.HTTPContext.Do(request)
}

func TestUpdatePullRequestBodySectionReleasesConnectionsBeforeProviderCalls(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
			require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)
			pool, err := f.db.DB()
			require.NoError(t, err)
			checkConnections := func() {
				assert.Zero(t, pool.Stats().InUse, "provider calls must not reserve a database connection")
			}
			payload := `{"state":"open","head":{"sha":"abc"}}`
			if provider == "bitbucket" {
				payload = `{"state":"OPEN","source":{"commit":{"hash":"abc"}}}`
			}
			host := &publicationHTTPContext{
				HTTPContext:   contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, payload), publicationHTTPResponse(200, "{}")}},
				beforeRequest: func(*http.Request) { checkConnections() },
			}
			if provider == "github" {
				previous := newFactoryGitHubAPI
				t.Cleanup(func() { newFactoryGitHubAPI = previous })
				newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
					checkConnections()
					return publicationGitHubClient(t, host), nil
				}
			} else {
				previous := newFactoryBitbucketAPI
				t.Cleanup(func() { newFactoryBitbucketAPI = previous })
				newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucket.Client, error) {
					checkConnections()
					return &bitbucket.Client{HTTP: host, Token: "token", AuthType: bitbucket.AuthTypeWorkspaceAccessToken}, nil
				}
			}
			require.NoError(t, f.publish(t, IntakeDependencies{}))
			require.Len(t, host.Requests, 2)
		})
	}
}

func TestUpdatePullRequestBodySectionSkipsRunSupersededDuringDescriptionRead(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			host := &publicationHTTPContext{
				HTTPContext: contexts.HTTPContext{Responses: []*http.Response{descriptionResponse(t, provider, ""), publicationHTTPResponse(200, "{}")}},
				beforeRequest: func(request *http.Request) {
					if request.Method != http.MethodGet {
						return
					}
					newer, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
					require.NoError(t, err)
					// Bound the lock wait so this regression also fails promptly when
					// publication holds the PR row lock during the provider request.
					err = f.db.Transaction(func(tx *gorm.DB) error {
						if err := tx.Exec("SET LOCAL lock_timeout = '100ms'").Error; err != nil {
							return err
						}
						_, err := f.pr.CreateActivity(tx, models.FactoryPullRequestActivityParams{RunID: newer.ID})
						return err
					})
					require.NoError(t, err)
				},
			}
			f.useProvider(t, provider, host)
			require.NoError(t, f.publish(t, IntakeDependencies{}))
			require.Len(t, host.Requests, 1, "a newer assessment must suppress the pending write")

		})
	}
}

func TestUpdatePullRequestBodySectionRejectsStaleRuns(t *testing.T) {
	for _, reason := range []string{"newer run", "newer disabled publication", "stale commit", "closed PR", "other section", "wrong factory"} {
		t.Run(reason, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`)}}
			installPublicationGitHub(t, httpCtx)
			switch reason {
			case "newer run", "newer disabled publication":
				if reason == "newer disabled publication" {
					version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
					require.NoError(t, err)
					version.ID = uuid.New()
					for index := range version.Nodes {
						if version.Nodes[index].ID == "publish" {
							version.Nodes[index].Configuration["enabled"] = false
						}
					}
					require.NoError(t, f.db.Create(version).Error)
					require.NoError(t, f.db.Model(f.canvas).Update("live_version_id", version.ID).Error)
				}
				newer, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
				require.NoError(t, err)
				_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: newer.ID})
				require.NoError(t, err)
				if reason == "newer disabled publication" {
					// Keep the current setting on. Only the newer run's saved
					// section, rather than the opt-out guard, must stop this write.
					require.NoError(t, f.db.Model(f.canvas).Update("live_version_id", f.run.VersionID).Error)
				}
			case "stale commit":
				httpCtx.Responses = []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"changed"}}`)}
			case "closed PR":
				httpCtx.Responses = []*http.Response{publicationHTTPResponse(200, `{"state":"closed","head":{"sha":"abc"}}`)}
			case "other section":
				version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
				require.NoError(t, err)
				version.ID = uuid.New()
				findIntakeNode(version.Nodes, "publish").Configuration["sectionKey"] = "other"
				require.NoError(t, f.db.Create(version).Error)
				require.NoError(t, f.db.Model(f.canvas).Update("live_version_id", version.ID).Error)
			case "wrong factory":
				other, err := models.CreateFactory(f.db, f.workspace.OrganizationID, support.RandomName("factory"), "", "")
				require.NoError(t, err)
				require.NoError(t, f.db.Model(f.canvas).Update("factory_id", other.ID).Error)
			}
			require.NoError(t, f.publish(t, IntakeDependencies{}))
			if strings.HasPrefix(reason, "newer ") || reason == "other section" || reason == "wrong factory" {
				assert.Empty(t, httpCtx.Requests)
			}
			for _, request := range httpCtx.Requests {
				assert.Equal(t, http.MethodGet, request.Method)
			}
		})
	}
}

func TestUpdatePullRequestBodySectionRefetchesOnRetryAndSkipsUnchanged(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			const initial = "<!-- greptile_comment -->review<!-- /greptile_comment -->"
			read := func(body string) *http.Response {
				payload := map[string]any{"state": "open", "head": map[string]any{"sha": "abc"}, "body": body}
				if provider == "bitbucket" {
					payload = map[string]any{"state": "OPEN", "source": map[string]any{"commit": map[string]any{"hash": "abc"}}, "description": body}
				}
				encoded, err := json.Marshal(payload)
				require.NoError(t, err)
				return publicationHTTPResponse(200, string(encoded))
			}
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{read(initial), publicationHTTPResponse(503, `{"message":"unavailable"}`), read("user edit\r\n" + initial), publicationHTTPResponse(200, "{}")}}
			require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
			require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)
			if provider == "github" {
				installPublicationGitHub(t, httpCtx)
			} else {
				installPublicationBitbucket(t, httpCtx)
			}
			require.Error(t, f.publish(t, IntakeDependencies{}))
			require.NoError(t, f.publish(t, IntakeDependencies{}))
			require.Len(t, httpCtx.Requests, 4)
			body, err := io.ReadAll(httpCtx.Requests[3].Body)
			require.NoError(t, err)
			var update map[string]string
			require.NoError(t, json.Unmarshal(body, &update))
			field := "body"
			if provider == "bitbucket" {
				field = "description"
			}
			assert.True(t, strings.HasPrefix(update[field], "user edit\r\n"+initial))
			assert.Contains(t, update[field], "arbitrary **Markdown**")
			httpCtx.Responses = []*http.Response{read(update[field])}
			require.NoError(t, f.publish(t, IntakeDependencies{}))
			require.Len(t, httpCtx.Requests, 5)
		})
	}
}

func installPublicationBitbucket(t *testing.T, httpCtx core.HTTPContext) {
	previous := newFactoryBitbucketAPI
	t.Cleanup(func() { newFactoryBitbucketAPI = previous })
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucket.Client, error) {
		return &bitbucket.Client{HTTP: httpCtx, Token: "token", AuthType: bitbucket.AuthTypeWorkspaceAccessToken}, nil
	}
}

func TestUpdatePullRequestBodySectionHostFailuresAndMalformedMarkers(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
			t.Run(provider+http.StatusText(status), func(t *testing.T) {
				f := newPublicationFixture(t)
				f.report(t, f.run, 5)
				f.finish(t, models.CanvasRunResultPassed)
				require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
				require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)
				httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(status, `{"message":"host error"}`)}}
				if provider == "github" {
					installPublicationGitHub(t, httpCtx)
				} else {
					installPublicationBitbucket(t, httpCtx)
				}
				err := f.publish(t, IntakeDependencies{})
				if status == 408 || status == 429 || status >= 500 {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				var run models.CanvasRun
				require.NoError(t, f.db.First(&run, "id = ?", f.run.ID).Error)
				assert.Equal(t, models.CanvasRunResultPassed, run.Result)
			})
		}
	}
	f := newPublicationFixture(t)
	f.report(t, f.run, 5)
	f.finish(t, models.CanvasRunResultPassed)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"<!-- superplane:merge-confidence:start -->"}`)}}
	installPublicationGitHub(t, httpCtx)
	require.NoError(t, f.publish(t, IntakeDependencies{}))
	require.Len(t, httpCtx.Requests, 1)
}

func TestUpdatePullRequestBodySectionClassifiesAppTokenFailures(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			previous := newFactoryGitHubAPI
			t.Cleanup(func() { newFactoryGitHubAPI = previous })
			newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
				return &fakeFactoryGitHub{getPullErr: fmt.Errorf("token exchange: %w", &ghinstallation.HTTPError{Message: "token exchange failed", Response: &http.Response{StatusCode: status}})}, nil
			}
			err := f.publish(t, IntakeDependencies{})
			if status == http.StatusUnauthorized {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestUpdatePullRequestBodySectionRetriesVCSSetupFailures(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			integrationID := uuid.NewString()
			require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
			require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)

			// The integration does not exist, so the connection is missing.
			outcome, err := f.publishOutcome(t, f.deps)
			require.NoError(t, err)
			assert.Equal(t, "skipped", outcome.Status)

			// A failed integration lookup is temporary and must retry.
			failingLookup, cancel := context.WithCancel(t.Context())
			cancel()
			lookupDB := f.db.WithContext(failingLookup)
			if provider == models.ProviderGitHub {
				previous := newFactoryGitHubAPI
				t.Cleanup(func() { newFactoryGitHubAPI = previous })
				newFactoryGitHubAPI = func(_ *gorm.DB, deps IntakeDependencies, factory *models.Factory) (factoryGitHubAPI, error) {
					return githubClientForFactory(lookupDB, deps, factory)
				}
			} else {
				previous := newFactoryBitbucketAPI
				t.Cleanup(func() { newFactoryBitbucketAPI = previous })
				newFactoryBitbucketAPI = func(_ *gorm.DB, deps IntakeDependencies, factory *models.Factory) (*bitbucket.Client, error) {
					return buildFactoryBitbucketAPI(lookupDB, deps, factory)
				}
			}
			_, err = f.publishOutcome(t, f.deps)
			require.ErrorIs(t, err, errFactoryVCSSetupFailed)
		})
	}
}

func (f publicationFixture) publishOutcome(t *testing.T, deps IntakeDependencies) (bodySectionPublicationOutcome, error) {
	t.Helper()
	version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
	require.NoError(t, err)
	config, err := factorycomponent.DecodeBodySectionConfiguration(findIntakeNode(version.Nodes, "publish").Configuration)
	require.NoError(t, err)
	return publishBodySection(t.Context(), f.db, deps, sectionPublication{run: f.run, nodeID: "publish", configuration: config})
}
func (f publicationFixture) publish(t *testing.T, deps IntakeDependencies) error {
	_, err := f.publishOutcome(t, deps)
	return err
}

func TestBodySectionPublicationIgnoresNewerRunsForOtherSections(t *testing.T) {
	f := newPublicationFixture(t)
	source := f.run
	f.publishVersion(t, func(v *models.CanvasVersion) {
		findIntakeNode(v.Nodes, "publish").Configuration["sectionKey"] = "other-section"
	})
	f.startRun(t)
	f.publishVersion(t, func(v *models.CanvasVersion) {
		findIntakeNode(v.Nodes, "publish").Configuration["sectionKey"] = "merge-confidence"
	})
	f.run = source
	host := &contexts.HTTPContext{Responses: []*http.Response{descriptionResponse(t, models.ProviderGitHub, "original"), publicationHTTPResponse(200, `{}`)}}
	installPublicationGitHub(t, host)
	outcome, err := f.publishOutcome(t, IntakeDependencies{})
	require.NoError(t, err)
	assert.Equal(t, "published", outcome.Status)
	require.Len(t, host.Requests, 2)
}

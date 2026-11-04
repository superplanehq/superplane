package factories

import (
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (f publicationFixture) publicationRequest(t *testing.T, channel string) (*models.CanvasNodeExecution, *models.CanvasNodeRequest) {
	t.Helper()
	root, err := f.run.RootEvent(f.db)
	require.NoError(t, err)
	agent := support.CreateCanvasNodeExecution(t, f.canvas.ID, "agent", root.ID, root.ID)
	status := "succeeded"
	if channel == "failed" {
		status = "failed"
	}
	state := workercontexts.NewExecutionStateContext(f.db, agent, nil)
	require.NoError(t, state.Emit(channel, "runnerClaudeCode.finished", []any{map[string]any{"status": status}}))
	var output models.CanvasEvent
	require.NoError(t, f.db.Where("execution_id = ?", agent.ID).First(&output).Error)
	require.NoError(t, f.db.Model(&output).Update("state", models.CanvasEventStateRouted).Error)
	execution := support.CreateNextNodeExecution(t, f.canvas.ID, "publish", root.ID, output.ID, &agent.ID)
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: factorycomponent.UpdatePullRequestBodySectionComponentName})
	provider := f.pr.Provider
	if provider == "" {
		provider = models.ProviderGitHub
	}
	raw, err := factoryTemplateFiles.ReadFile("templates/" + provider + "/risk-score.canvas.yaml")
	require.NoError(t, err)
	template, err := yaml.CanvasFromYAML(raw)
	require.NoError(t, err)
	configuration := findYAMLNode(t, template, "publish-merge-confidence").Configuration
	resolved, err := workercontexts.NewNodeConfigurationBuilder(f.db, f.canvas.ID).
		WithRunID(f.run.ID).WithNodeID("publish").WithRootEvent(&root.ID).
		WithPreviousExecution(&agent.ID).WithIncomingEventID(&output.ID).
		WithInput(output.Data.Data()).
		WithConfigurationFields((&factorycomponent.UpdatePullRequestBodySection{}).Configuration()).Build(configuration)
	require.NoError(t, err)
	execution.Configuration = datatypes.NewJSONType(resolved)
	require.NoError(t, f.db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.UpdatePullRequestBodySection{}).Execute(core.ExecutionContext{
		RunID: execution.RunID, Configuration: execution.Configuration.Data(),
		Metadata: workercontexts.NewExecutionMetadataContext(f.db, execution), Requests: workercontexts.NewExecutionRequestContext(f.db, execution),
	}))
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	return execution, &request
}

func TestBodySectionActionRetriesAndRechecksEligibility(t *testing.T) {
	for _, superseded := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest description", true: "superseded assessment"}[superseded], func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 4)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`),
				publicationHTTPResponse(503, `{"message":"unavailable"}`),
				publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"latest user edit"}`),
				publicationHTTPResponse(200, `{}`),
			}}
			installPublicationGitHub(t, host)
			execution, request := f.publicationRequest(t, "passed")
			assert.Equal(t, f.run.ID, execution.RunID)
			_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, "retrying", execution.Metadata.Data()["status"])
			assert.EqualValues(t, 1, execution.Metadata.Data()["attempts"])
			assert.NotEmpty(t, execution.Metadata.Data()["lastError"])
			assert.Equal(t, factorycomponent.UpdatePullRequestBodySectionComponentName, execution.FrozenComponentName())
			assert.NotEqual(t, models.CanvasNodeExecutionStateFinished, execution.State)
			var retry models.CanvasNodeRequest
			require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&retry).Error)
			assert.True(t, retry.RunAt.After(time.Now()))
			if superseded {
				newer, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
				require.NoError(t, err)
				_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: newer.ID})
				require.NoError(t, err)
			}

			if !superseded {
				f.publishVersion(t, func(v *models.CanvasVersion) {
					findIntakeNode(v.Nodes, "publish").Configuration["body"] = "Changed body template"
				})
			}
			_, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)
			assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
			assert.EqualValues(t, 2, execution.Metadata.Data()["attempts"])
			if superseded {
				assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
				assert.NotEmpty(t, execution.Metadata.Data()["reason"])
				require.Len(t, host.Requests, 2)
			} else {
				assert.Equal(t, "published", execution.Metadata.Data()["status"])
				require.Len(t, host.Requests, 4)
				body, err := io.ReadAll(host.Requests[3].Body)
				require.NoError(t, err)
				var update map[string]string
				require.NoError(t, json.Unmarshal(body, &update))
				assert.True(t, strings.HasPrefix(update["body"], "latest user edit"))
				assert.Contains(t, update["body"], "| Security | ✅ | No security gap. |")
				assert.NotContains(t, update["body"], "Changed body template")
			}
			_, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, retry.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			var source models.CanvasRun
			require.NoError(t, f.db.First(&source, "id = ?", f.run.ID).Error)
			assert.Equal(t, models.CanvasRunStateStarted, source.State)
			result, err := source.CalculateResult(f.db)
			require.NoError(t, err)
			assert.Equal(t, models.CanvasRunResultPassed, result)
		})
	}
}

func TestBodySectionActionRecoversInterruptedAttempt(t *testing.T) {
	f := newPublicationFixture(t)
	f.report(t, f.run, 5)
	execution, request := f.publicationRequest(t, "failed")
	expired := time.Now().Add(-time.Minute)
	require.NoError(t, setBodySectionPublicationState(f.db, execution, bodySectionPublicationState{SourceRunID: f.run.ID.String(), Status: "publishing", Attempts: 1, LeaseUntil: &expired}))
	require.NoError(t, request.Reschedule(f.db, expired))
	host := &publicationHTTPContext{
		HTTPContext: contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`), publicationHTTPResponse(200, `{}`)}},
		beforeRequest: func(*http.Request) {
			// A duplicate delivery while the recovered attempt owns the lease
			// cannot start another host request.
			_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
		},
	}
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, host), nil
	}
	_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequest)
	require.NoError(t, err)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
	assert.Equal(t, "published", execution.Metadata.Data()["status"])
	assert.EqualValues(t, 2, execution.Metadata.Data()["attempts"])
	require.Len(t, host.Requests, 2)
	body, err := io.ReadAll(host.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "**Incomplete**")
	require.NoError(t, f.db.First(f.run, "id = ?", f.run.ID).Error)
	result, err := f.run.CalculateResult(f.db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func TestBodySectionActionSkipsWriteAfterCancellationDuringDescriptionRead(t *testing.T) {
	cancellations := map[string]func(t *testing.T, f publicationFixture, execution *models.CanvasNodeExecution){
		"run": func(t *testing.T, f publicationFixture, _ *models.CanvasNodeExecution) {
			run, err := models.FindCanvasRunInTransaction(f.db, f.canvas.ID, f.run.ID)
			require.NoError(t, err)
			require.NoError(t, run.MarkAsCancelling(f.db, nil))
		},
		"execution": func(t *testing.T, f publicationFixture, execution *models.CanvasNodeExecution) {
			current, err := models.FindNodeExecutionInTransaction(f.db, execution.WorkflowID, execution.ID)
			require.NoError(t, err)
			require.NoError(t, current.RequestCancellation(f.db, nil))
		},
	}
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for name, cancel := range cancellations {
			t.Run(provider+"/"+name, func(t *testing.T) {
				f := newPublicationFixture(t)
				f.report(t, f.run, 4)
				execution, request := f.publicationRequest(t, "passed")
				host := &publicationHTTPContext{
					HTTPContext: contexts.HTTPContext{Responses: []*http.Response{
						descriptionResponse(t, provider, "original"),
						publicationHTTPResponse(200, `{}`),
					}},
					beforeRequest: func(request *http.Request) {
						if request.Method == http.MethodGet {
							cancel(t, f, execution)
						}
					},
				}
				f.useProvider(t, provider, host)
				_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
				require.NoError(t, err)
				require.Len(t, host.Requests, 1, "a cancellation during the description read must stop the write")
			})
		}
	}
}

func (f publicationFixture) publishVersion(t *testing.T, change func(*models.CanvasVersion)) *models.CanvasVersion {
	t.Helper()
	canvas, err := models.FindCanvasInTransaction(f.db, f.canvas.OrganizationID, f.canvas.ID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(f.db, canvas)
	require.NoError(t, err)
	version.ID = uuid.New()
	change(version)
	require.NoError(t, f.db.Create(version).Error)
	require.NoError(t, f.db.Model(f.canvas).Update("live_version_id", version.ID).Error)
	return version
}

func (f *publicationFixture) startRun(t *testing.T) {
	t.Helper()
	run, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, f.db.Create(&models.CanvasEvent{WorkflowID: f.canvas.ID, NodeID: "trigger", RunID: run.ID, State: models.CanvasEventStateRouted, Data: models.NewJSONValue(map[string]any{"data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "abc"}}}})}).Error)
	_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: run.ID})
	require.NoError(t, err)
	f.run = run
}

func (f publicationFixture) arbitraryPublicationRequest(t *testing.T) (*models.CanvasNodeExecution, *models.CanvasNodeRequest) {
	t.Helper()
	root, err := f.run.RootEvent(f.db)
	require.NoError(t, err)
	version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
	require.NoError(t, err)
	execution := support.CreateCanvasNodeExecution(t, f.canvas.ID, "publish", root.ID, root.ID)
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Configuration = datatypes.NewJSONType(findIntakeNode(version.Nodes, "publish").Configuration)
	execution.Metadata = datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: factorycomponent.UpdatePullRequestBodySectionComponentName})
	require.NoError(t, f.db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.UpdatePullRequestBodySection{}).Execute(core.ExecutionContext{
		RunID: execution.RunID, Configuration: execution.Configuration.Data(),
		Metadata: workercontexts.NewExecutionMetadataContext(f.db, execution), Requests: workercontexts.NewExecutionRequestContext(f.db, execution),
	}))
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	return execution, &request
}

func (f publicationFixture) useProvider(t *testing.T, provider string, host core.HTTPContext) {
	t.Helper()
	require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
	require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)
	if provider == models.ProviderGitHub {
		installPublicationGitHub(t, host)
		return
	}
	installPublicationBitbucket(t, host)
}

func descriptionResponse(t *testing.T, provider, description string) *http.Response {
	t.Helper()
	payload := map[string]any{"state": "open", "head": map[string]any{"sha": "abc"}, "body": description}
	if provider == models.ProviderBitbucket {
		payload = map[string]any{"state": "OPEN", "source": map[string]any{"commit": map[string]any{"hash": "abc"}}, "description": description}
	}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return publicationHTTPResponse(200, string(encoded))
}

func TestBodySectionActionPublishesArbitraryFactoryContent(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			const markdown = "## Release notes\n\n- **Ready**\n\n<!-- arbitrary comment -->"
			f.publishVersion(t, func(v *models.CanvasVersion) {
				trigger := *findIntakeNode(v.Nodes, "trigger")
				trigger.Metadata = nil
				publication := *findIntakeNode(v.Nodes, "publish")
				publication.Configuration = map[string]any{"pullRequestId": f.pr.ID.String(), "sectionKey": "release-notes", "body": markdown}
				v.Nodes = []models.Node{trigger, publication}
			})
			f.startRun(t)
			require.NoError(t, f.db.Where("run_id = ?", f.run.ID).Delete(&models.FactoryPullRequestRun{}).Error)
			// This canvas has no assessment configuration, activity link, or agent input, and does not
			// require a reviewed commit. Other owned sections retain their exact bytes.
			const prefix = "User text\r\n<!-- superplane:other:start -->keep<!-- superplane:other:end -->\r\n"
			host := &contexts.HTTPContext{Responses: []*http.Response{descriptionResponse(t, provider, prefix), publicationHTTPResponse(200, `{}`)}}
			f.useProvider(t, provider, host)
			execution, request := f.arbitraryPublicationRequest(t)
			_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.Len(t, host.Requests, 2)
			body, err := io.ReadAll(host.Requests[1].Body)
			require.NoError(t, err)
			var update map[string]string
			require.NoError(t, json.Unmarshal(body, &update))
			field := "body"
			if provider == models.ProviderBitbucket {
				field = "description"
			}
			assert.Equal(t, prefix+"\n\n<!-- superplane:release-notes:start -->\n"+markdown+"\n<!-- superplane:release-notes:end -->", update[field])
			require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, "published", execution.Metadata.Data()["status"])
		})
	}
}

func TestBodySectionActionHonorsSavedAndCurrentOptOut(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		for _, timing := range []string{"saved run", "before attempt", "during read", "before retry"} {
			t.Run(provider+"/"+timing, func(t *testing.T) {
				f := newPublicationFixture(t)
				setEnabled := func(enabled bool) {
					f.publishVersion(t, func(v *models.CanvasVersion) { findIntakeNode(v.Nodes, "publish").Configuration["enabled"] = enabled })
				}
				if timing == "saved run" {
					setEnabled(false)
					f.startRun(t)
				}
				execution, request := f.arbitraryPublicationRequest(t)
				if timing == "saved run" {
					setEnabled(true)
				}
				if timing == "before attempt" {
					setEnabled(false)
				}
				host := &publicationHTTPContext{HTTPContext: contexts.HTTPContext{Responses: []*http.Response{descriptionResponse(t, provider, "existing report"), publicationHTTPResponse(503, `{"message":"unavailable"}`)}}, beforeRequest: func(request *http.Request) {
					if timing == "during read" && request.Method == http.MethodGet {
						setEnabled(false)
					}
				}}
				f.useProvider(t, provider, host)
				_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
				require.NoError(t, err)
				if timing == "before retry" {
					require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
					assert.Equal(t, "retrying", execution.Metadata.Data()["status"])
					setEnabled(false)
					request = &models.CanvasNodeRequest{}
					require.NoError(t, f.db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).First(request).Error)
					_, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
					require.NoError(t, err)
				}
				require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
				assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
				expectedCalls := map[string]int{"saved run": 0, "before attempt": 0, "during read": 1, "before retry": 2}[timing]
				assert.Len(t, host.Requests, expectedCalls)
				// Re-enabling applies to subsequent runs. A saved false never becomes true
				// for the old run, even after the published settings change again.
				setEnabled(true)
				f.startRun(t)
				host.beforeRequest = func(*http.Request) {}
				host.Responses = []*http.Response{descriptionResponse(t, provider, "existing report"), publicationHTTPResponse(200, `{}`)}
				nextExecution, nextRequest := f.arbitraryPublicationRequest(t)
				_, err = ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, nextRequest.ID, models.LockNodeRequestNow)
				require.NoError(t, err)
				require.NoError(t, f.db.First(nextExecution, "id = ?", nextExecution.ID).Error)
				assert.Equal(t, "published", nextExecution.Metadata.Data()["status"])
				assert.Len(t, host.Requests, expectedCalls+2)
			})
		}
	}
}

func TestMergeConfidenceOptOutKeepsAssessmentResults(t *testing.T) {
	f := newPublicationFixture(t)
	f.publishVersion(t, func(v *models.CanvasVersion) { findIntakeNode(v.Nodes, "publish").Configuration["enabled"] = false })
	f.startRun(t)
	f.report(t, f.run, 4)
	execution, request := f.publicationRequest(t, "failed")
	host := &contexts.HTTPContext{}
	installPublicationGitHub(t, host)
	_, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
	require.NoError(t, err)
	require.NoError(t, f.db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
	assert.Contains(t, execution.Configuration.Data()["body"], "**Incomplete**")
	checks, err := f.order.ChecksForRun(f.db, f.run.ID)
	require.NoError(t, err)
	assert.Equal(t, 4.0, checks["security-review"].Score)
	require.NoError(t, f.db.First(f.run, "id = ?", f.run.ID).Error)
	result, err := f.run.CalculateResult(f.db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
	assert.Empty(t, host.Requests)
}

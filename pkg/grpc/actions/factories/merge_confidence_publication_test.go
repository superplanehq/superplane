package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)), Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}})}, {NodeID: "agent", Type: models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"environment": []any{map[string]any{"name": "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID", "value": order.ID.String()}}, "steps": []any{map[string]any{"name": "Security", "prompt": "Merge check: security."}}}),
	}, {NodeID: "publication", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRunFinished"}})},
		{NodeID: "publish", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: factorycomponent.PublishMergeConfidenceComponentName}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.CanvasEvent{WorkflowID: canvas.ID, NodeID: "trigger", RunID: run.ID, State: models.CanvasEventStateRouted, Data: models.NewJSONValue(map[string]any{"data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "abc"}}}})}).Error)
	_, err = pr.CreateActivity(db, models.FactoryPullRequestActivityParams{RunID: run.ID, Title: "Merge confidence"})
	require.NoError(t, err)
	return publicationFixture{db, workspace, order, pr, canvas, run, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor}}
}

func (f publicationFixture) report(t *testing.T, run *models.CanvasRun, score float64) {
	_, err := f.order.ReportCheck(f.db, models.FactoryWorkOrderCheckParams{Key: "security-review", Name: "Security", Score: score, MaxScore: 5, Summary: "No security gap.", Run: &factoryevents.RunRef{ID: run.ID}})
	require.NoError(t, err)
}

func (f publicationFixture) finish(t *testing.T, result string) {
	require.NoError(t, f.db.Model(f.run).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": result}).Error)
}

func installPublicationGitHub(t *testing.T, httpCtx *contexts.HTTPContext) {
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, httpCtx), nil
	}
}

func TestPublishMergeConfidenceWaitsForTerminalStateAndUsesRunSnapshots(t *testing.T) {
	f := newPublicationFixture(t)
	httpCtx := &contexts.HTTPContext{}
	installPublicationGitHub(t, httpCtx)
	require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
	assert.Empty(t, httpCtx.Requests)
	f.report(t, f.run, 4)
	// A different run overwrites the mutable row. Only the immutable report for
	// this run must appear, even after the live configuration changes.
	other, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	f.report(t, other, 1)
	require.NoError(t, f.db.Model(&models.CanvasNode{}).Where("workflow_id = ?", f.canvas.ID).Update("configuration", datatypes.NewJSONType(map[string]any{"steps": []any{}})).Error)
	f.finish(t, models.CanvasRunResultPassed)
	httpCtx.Responses = []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"},"body":"original"}`), publicationHTTPResponse(200, "{}")}
	require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
	require.Len(t, httpCtx.Requests, 2)
	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	var update map[string]string
	require.NoError(t, json.Unmarshal(body, &update))
	assert.Contains(t, update["body"], "**Overall score: 4/5**")
	assert.Contains(t, update["body"], "original\n\n<!-- superplane:merge-confidence:start -->")
	assert.Contains(t, update["body"], f.order.URLPath(f.workspace.RouteSegment()))
}

func TestPublishMergeConfidenceIncomplete(t *testing.T) {
	for _, result := range []string{models.CanvasRunResultFailed, models.CanvasRunResultCancelled, models.CanvasRunResultPassed} {
		t.Run(result, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.finish(t, result)
			if result != models.CanvasRunResultPassed {
				f.report(t, f.run, 4)
			}
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`), publicationHTTPResponse(200, "{}")}}
			installPublicationGitHub(t, httpCtx)
			require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
			require.Len(t, httpCtx.Requests, 2)
			body, err := io.ReadAll(httpCtx.Requests[1].Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), "**Incomplete**")
			assert.NotContains(t, string(body), "Overall score")
			if result == models.CanvasRunResultPassed {
				assert.Contains(t, string(body), "Missing")
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

func TestPublishMergeConfidenceReleasesConnectionsBeforeProviderCalls(t *testing.T) {
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
			require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
			require.Len(t, host.Requests, 2)
		})
	}
}

func TestPublishMergeConfidenceSkipsRunSupersededDuringDescriptionRead(t *testing.T) {
	f := newPublicationFixture(t)
	f.report(t, f.run, 5)
	f.finish(t, models.CanvasRunResultPassed)
	host := &publicationHTTPContext{
		HTTPContext: contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`), publicationHTTPResponse(200, "{}")}},
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
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, host), nil
	}
	require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
	require.Len(t, host.Requests, 1, "a newer assessment must suppress the pending write")
}

func TestPublishMergeConfidenceRejectsStaleRuns(t *testing.T) {
	for _, reason := range []string{"newer run", "newer invalid configuration", "stale commit", "closed PR", "unlinked run", "wrong canvas"} {
		t.Run(reason, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`)}}
			installPublicationGitHub(t, httpCtx)
			canvasID := f.canvas.ID
			switch reason {
			case "newer run", "newer invalid configuration":
				if reason == "newer invalid configuration" {
					version, err := models.FindCanvasVersionInTransaction(f.db, f.canvas.ID, f.run.VersionID)
					require.NoError(t, err)
					version.ID = uuid.New()
					for index := range version.Nodes {
						version.Nodes[index].Configuration = map[string]any{}
					}
					require.NoError(t, f.db.Create(version).Error)
					require.NoError(t, f.db.Model(f.canvas).Update("live_version_id", version.ID).Error)
				}
				newer, err := models.CreateCanvasRunInTransaction(f.db, f.canvas.ID, "trigger", models.CanvasRunStateStarted, "")
				require.NoError(t, err)
				_, err = f.pr.CreateActivity(f.db, models.FactoryPullRequestActivityParams{RunID: newer.ID})
				require.NoError(t, err)
			case "stale commit":
				httpCtx.Responses = []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"changed"}}`)}
			case "closed PR":
				httpCtx.Responses = []*http.Response{publicationHTTPResponse(200, `{"state":"closed","head":{"sha":"abc"}}`)}
			case "unlinked run":
				require.NoError(t, f.db.Where("run_id = ?", f.run.ID).Delete(&models.FactoryPullRequestRun{}).Error)
			case "wrong canvas":
				canvasID = uuid.New()
			}
			require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, canvasID, f.run.ID, "https://example.com"))
			if strings.HasPrefix(reason, "newer ") || reason == "unlinked run" || reason == "wrong canvas" {
				assert.Empty(t, httpCtx.Requests)
			}
			for _, request := range httpCtx.Requests {
				assert.Equal(t, http.MethodGet, request.Method)
			}
		})
	}
}

func TestPublishMergeConfidenceRefetchesOnRetryAndSkipsUnchanged(t *testing.T) {
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
			require.Error(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
			require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
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
			assert.Contains(t, update[field], "**Overall score: 5/5**")
			httpCtx.Responses = []*http.Response{read(update[field])}
			require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
			require.Len(t, httpCtx.Requests, 5)
		})
	}
}

func installPublicationBitbucket(t *testing.T, httpCtx *contexts.HTTPContext) {
	previous := newFactoryBitbucketAPI
	t.Cleanup(func() { newFactoryBitbucketAPI = previous })
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucket.Client, error) {
		return &bitbucket.Client{HTTP: httpCtx, Token: "token", AuthType: bitbucket.AuthTypeWorkspaceAccessToken}, nil
	}
}

func TestPublishMergeConfidenceHostFailuresAndMalformedMarkers(t *testing.T) {
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
				err := PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com")
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
	require.NoError(t, PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com"))
	require.Len(t, httpCtx.Requests, 1)
}

// This host retains the written body, like the real API. Delaying the first
// write exposes lost updates if simultaneous deliveries do not take the PR lock.
type concurrentPublicationHost struct {
	mu             sync.Mutex
	body           string
	reads, updates int
	writing        chan struct{}
	release        chan struct{}
}

func (h *concurrentPublicationHost) Do(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodGet {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.reads++
		encoded, _ := json.Marshal(map[string]any{"state": "open", "head": map[string]any{"sha": "abc"}, "body": h.body})
		return publicationHTTPResponse(200, string(encoded)), nil
	}
	h.mu.Lock()
	h.updates++
	first := h.updates == 1
	h.mu.Unlock()
	if first {
		close(h.writing)
		<-h.release
	}
	var update map[string]string
	if err := json.NewDecoder(request.Body).Decode(&update); err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.body = update["body"]
	h.mu.Unlock()
	return publicationHTTPResponse(200, "{}"), nil
}

func TestPublishMergeConfidenceSerializesConcurrentDeliveries(t *testing.T) {
	f := newPublicationFixture(t)
	f.report(t, f.run, 5)
	f.finish(t, models.CanvasRunResultPassed)
	host := &concurrentPublicationHost{writing: make(chan struct{}), release: make(chan struct{})}
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, host), nil
	}
	finished := make(chan error, 2)
	publish := func() {
		finished <- PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com")
	}
	go publish()
	select {
	case <-host.writing:
	case <-time.After(5 * time.Second):
		t.Fatal("publication did not reach host")
	}
	go publish()
	// Give the second delivery a chance to wait for the durable claim.
	// Without serialization both deliveries read an empty description.
	time.Sleep(100 * time.Millisecond)
	close(host.release)
	for range 2 {
		select {
		case err := <-finished:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("publication did not complete")
		}
	}
	assert.Equal(t, 2, host.reads)
	assert.Equal(t, 1, host.updates, "the second delivery must see the published section and skip its write")
}

func TestPublishMergeConfidenceRecoversExpiredClaim(t *testing.T) {
	f := newPublicationFixture(t)
	f.finish(t, models.CanvasRunResultPassed)
	f.report(t, f.run, 5)
	old := &models.MergeConfidencePublicationClaim{PullRequestID: f.pr.ID, OwnerID: uuid.New(), LeaseUntil: time.Now().Add(-time.Minute)}
	acquired, err := old.Acquire(f.db)
	require.NoError(t, err)
	require.True(t, acquired)
	host := &publicationHTTPContext{
		HTTPContext: contexts.HTTPContext{Responses: []*http.Response{publicationHTTPResponse(200, `{"state":"open","head":{"sha":"abc"}}`), publicationHTTPResponse(200, `{}`)}},
		beforeRequest: func(*http.Request) {
			// An old worker can finish after its lease expires. Its cleanup must
			// leave the new owner's claim intact.
			require.NoError(t, old.Release(f.db))
		},
	}
	previous := newFactoryGitHubAPI
	t.Cleanup(func() { newFactoryGitHubAPI = previous })
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return publicationGitHubClient(t, host), nil
	}
	outcome, err := PublishMergeConfidenceWithOutcome(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "published", outcome.Status)
	require.Len(t, host.Requests, 2)
}

func TestPublishMergeConfidenceClassifiesAppTokenFailures(t *testing.T) {
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
			err := PublishMergeConfidence(t.Context(), f.db, IntakeDependencies{}, f.canvas.ID, f.run.ID, "https://example.com")
			if status == http.StatusUnauthorized {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPublishMergeConfidenceRetriesVCSSetupFailures(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			f := newPublicationFixture(t)
			f.report(t, f.run, 5)
			f.finish(t, models.CanvasRunResultPassed)
			integrationID := uuid.NewString()
			require.NoError(t, f.workspace.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
			require.NoError(t, f.db.Model(f.pr).Update("provider", provider).Error)

			// The integration does not exist, so the connection is missing.
			outcome, err := PublishMergeConfidenceWithOutcome(t.Context(), f.db, f.deps, f.canvas.ID, f.run.ID, "https://example.com")
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
			_, err = PublishMergeConfidenceWithOutcome(t.Context(), f.db, f.deps, f.canvas.ID, f.run.ID, "https://example.com")
			require.ErrorIs(t, err, errFactoryVCSSetupFailed)
		})
	}
}

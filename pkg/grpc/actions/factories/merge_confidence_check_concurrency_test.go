package factories

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestMergeConfidenceCheckReleasesRowLocksDuringGitHubDelivery(t *testing.T) {
	f, request := newMergeConfidenceCheckFixture(t)
	emitMergeConfidenceCheckAssessment(t, f, "passed", true)
	f.finish(t, models.CanvasRunResultPassed)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		publicationHTTPResponse(200, `{"check_runs":[]}`),
		publicationHTTPResponse(201, `{"id":123}`),
	}}
	paused := newPausedMergeConfidenceHTTP(t, host)
	installPublicationGitHub(t, paused)
	completed := make(chan error, 1)
	go func() { completed <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID) }()
	awaitMergeConfidenceDelivery(t, paused.started)

	for _, row := range []struct {
		name  string
		model any
		id    any
	}{
		{"request", &models.CanvasNodeRequest{}, request.ID},
		{"task", &models.FactoryWorkOrder{}, f.order.ID},
		{"pull request", &models.FactoryPullRequest{}, f.pr.ID},
	} {
		t.Run(row.name, func(t *testing.T) {
			err := f.db.Transaction(func(tx *gorm.DB) error {
				return tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).First(row.model, "id = ?", row.id).Error
			})
			require.NoError(t, err, "GitHub delivery must not hold the %s row lock", row.name)
		})
	}
	paused.resume()
	require.NoError(t, awaitMergeConfidenceResult(t, completed))
}

func TestMergeConfidenceCheckSkipsCompetingDeliveryForTheSamePullRequest(t *testing.T) {
	f, request := newMergeConfidenceCheckFixture(t)
	emitMergeConfidenceCheckAssessment(t, f, "passed", true)
	f.finish(t, models.CanvasRunResultPassed)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		publicationHTTPResponse(200, `{"check_runs":[]}`),
		publicationHTTPResponse(201, `{"id":123}`),
	}}
	paused := newPausedMergeConfidenceHTTP(t, host)
	installPublicationGitHub(t, paused)
	first := make(chan error, 1)
	go func() { first <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID) }()
	awaitMergeConfidenceDelivery(t, paused.started)

	competing := make(chan error, 1)
	go func() { competing <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID) }()
	require.NoError(t, awaitMergeConfidenceResult(t, competing), "an occupied publication lock must not block polling workers")
	stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)

	paused.resume()
	require.NoError(t, awaitMergeConfidenceResult(t, first))
	require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
	require.Len(t, host.Requests, 2, "simultaneous workers must create only one check")
	assert.Equal(t, http.MethodPost, host.Requests[1].Method)
}

func TestMergeConfidenceCheckRechecksNewerAssessmentBeforeCreatingCheck(t *testing.T) {
	f, request := newMergeConfidenceCheckFixture(t)
	emitMergeConfidenceCheckAssessment(t, f, "passed", true)
	f.finish(t, models.CanvasRunResultPassed)
	host := &contexts.HTTPContext{Responses: []*http.Response{
		publicationHTTPResponse(200, `{"check_runs":[]}`),
	}}
	paused := newPausedMergeConfidenceHTTP(t, host)
	paused.method = http.MethodGet
	installPublicationGitHub(t, paused)
	completed := make(chan error, 1)
	go func() { completed <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID) }()
	awaitMergeConfidenceDelivery(t, paused.started)

	newer := f
	newer.startRun(t)
	paused.resume()
	require.NoError(t, awaitMergeConfidenceResult(t, completed))
	require.Len(t, host.Requests, 1, "an old check must not be created after a newer assessment arrives")
	stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
}

func TestMergeConfidenceCheckReconcilesStateChangesDuringDelivery(t *testing.T) {
	for _, change := range []string{"newer assessment", "deleted automation", "finished assessment"} {
		t.Run(change, func(t *testing.T) {
			f, request := newMergeConfidenceCheckFixture(t)
			emitMergeConfidenceCheckAssessment(t, f, "passed", true)
			if change != "finished assessment" {
				f.finish(t, models.CanvasRunResultPassed)
			}
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"check_runs":[]}`),
				publicationHTTPResponse(201, `{"id":123}`),
			}}
			paused := newPausedMergeConfidenceHTTP(t, host)
			installPublicationGitHub(t, paused)
			completed := make(chan error, 1)
			go func() { completed <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID) }()
			awaitMergeConfidenceDelivery(t, paused.started)

			var newerRequest *models.CanvasNodeRequest
			var newer publicationFixture
			switch change {
			case "newer assessment":
				newer = f
				newer.startRun(t)
				emitMergeConfidenceCheckAssessment(t, newer, "passed", true)
				newer.finish(t, models.CanvasRunResultPassed)
				newerRequest = queueMergeConfidenceRun(t, newer)
				competing := make(chan error, 1)
				go func() {
					competing <- ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, newerRequest.ID)
				}()
				require.NoError(t, awaitMergeConfidenceResult(t, competing), "another run on the same PR must wait for a later poll")
				stored, err := models.FindCanvasNodeRequest(f.db, newerRequest.ID)
				require.NoError(t, err)
				assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)
			case "deleted automation":
				require.NoError(t, f.db.Model(f.canvas).Update("deleted_at", time.Now()).Error)
			case "finished assessment":
				f.finish(t, models.CanvasRunResultPassed)
			}
			paused.resume()
			require.NoError(t, awaitMergeConfidenceResult(t, completed))
			stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State, "delivery of an outdated result must leave reconciliation pending")

			if newerRequest != nil {
				host.Responses = []*http.Response{
					mergeConfidenceCheckListResponse(f.run.ID, "completed"),
					publicationHTTPResponse(201, `{"id":456}`),
				}
				require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, newerRequest.ID))
				require.Len(t, host.Requests, 4)
				assert.Equal(t, http.MethodPost, host.Requests[3].Method)
				assert.Equal(t, newer.run.ID.String(), mergeConfidenceCheckRequestBody(t, host.Requests[3])["external_id"])
			}
			host.Responses = []*http.Response{
				mergeConfidenceCheckListResponse(f.run.ID, "in_progress"),
				publicationHTTPResponse(200, `{"id":123}`),
			}
			require.NoError(t, request.Reschedule(f.db, time.Now().Add(-time.Second)))
			require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			last := host.Requests[len(host.Requests)-1]
			assert.Equal(t, http.MethodPatch, last.Method, "the old run must update its original check after a newer publication")
			assert.Equal(t, "/repos/acme/app/check-runs/123", last.URL.Path)
			conclusion := "cancelled"
			if change == "finished assessment" {
				conclusion = "success"
			}
			assert.Equal(t, conclusion, mergeConfidenceCheckRequestBody(t, last)["conclusion"])
			stored, err = models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
		})
	}
}

func TestMergeConfidenceCheckReleasesPublicationLockAfterFailureOrCancellation(t *testing.T) {
	for _, outcome := range []string{"provider failure", "cancellation during delivery", "cancellation after lookup", "cancellation during state read"} {
		t.Run(outcome, func(t *testing.T) {
			f, request := newMergeConfidenceCheckFixture(t)
			emitMergeConfidenceCheckAssessment(t, f, "passed", true)
			f.finish(t, models.CanvasRunResultPassed)
			host := &contexts.HTTPContext{Responses: []*http.Response{
				publicationHTTPResponse(200, `{"check_runs":[]}`),
				publicationHTTPResponse(502, `{"message":"GitHub unavailable"}`),
			}}
			paused := newPausedMergeConfidenceHTTP(t, host)
			installPublicationGitHub(t, paused)
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if outcome == "cancellation after lookup" || outcome == "cancellation during state read" {
				paused.method = http.MethodGet
			}
			if outcome == "cancellation after lookup" {
				paused.afterResponse = cancel
			}
			sqlDB, err := f.db.DB()
			require.NoError(t, err)
			// Reserve this physical connection before delivery so PostgreSQL cannot
			// satisfy a leaked session lock with a reentrant acquisition.
			observer, err := sqlDB.Conn(t.Context())
			require.NoError(t, err)
			t.Cleanup(func() { _ = observer.Close() })
			completed := make(chan error, 1)
			go func() { completed <- ProcessMergeConfidenceCheckRequest(ctx, IntakeDependencies{}, request.ID) }()
			awaitMergeConfidenceDelivery(t, paused.started)
			switch outcome {
			case "cancellation during delivery":
				cancel()
			case "cancellation during state read":
				blocker := f.db.Begin()
				require.NoError(t, blocker.Error)
				t.Cleanup(func() { _ = blocker.Rollback().Error })
				require.NoError(t, blocker.Clauses(clause.Locking{Strength: "UPDATE"}).First(&models.FactoryWorkOrder{}, "id = ?", f.order.ID).Error)
				var publisherPID int
				require.NoError(t, observer.QueryRowContext(t.Context(),
					"SELECT pid FROM pg_locks WHERE locktype = 'advisory' AND objid = hashtext($1)::oid AND granted",
					"merge-confidence:"+f.pr.ID.String()).Scan(&publisherPID))
				paused.resume()
				// Wait for the actual SQL read before canceling. The HTTP cancellation
				// guards cannot avoid a connection loss once the transaction starts.
				require.Eventually(t, func() bool {
					var waiting bool
					err := observer.QueryRowContext(t.Context(),
						"SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock')",
						publisherPID).Scan(&waiting)
					return err == nil && waiting
				}, 5*time.Second, 10*time.Millisecond, "publisher must reach the SQL row-lock wait")
				cancel()
				require.NoError(t, blocker.Rollback().Error)
			default:
				paused.resume()
			}
			publicationErr := awaitMergeConfidenceResult(t, completed)
			require.Error(t, publicationErr)
			lockKey := "merge-confidence:" + f.pr.ID.String()
			var observerErr error
			require.Eventually(t, func() bool {
				var acquired bool
				observerErr = observer.QueryRowContext(t.Context(), "SELECT pg_try_advisory_lock(hashtext($1))", lockKey).Scan(&acquired)
				if observerErr != nil || !acquired {
					return observerErr != nil
				}
				_, observerErr = observer.ExecContext(t.Context(), "SELECT pg_advisory_unlock(hashtext($1))", lockKey)
				return true
			}, 5*time.Second, 10*time.Millisecond, "failed or canceled delivery must release its session lock")
			require.NoError(t, observerErr)
			stored, err := models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)
			assert.True(t, stored.RunAt.After(time.Now()), "failed delivery must schedule a retry even after cancellation: %v", publicationErr)

			paused.resume()
			host.Responses = []*http.Response{
				publicationHTTPResponse(200, `{"check_runs":[]}`),
				publicationHTTPResponse(201, `{"id":123}`),
			}
			require.NoError(t, request.Reschedule(f.db, time.Now().Add(-time.Second)))
			require.NoError(t, ProcessMergeConfidenceCheckRequest(t.Context(), IntakeDependencies{}, request.ID))
			stored, err = models.FindCanvasNodeRequest(f.db, request.ID)
			require.NoError(t, err)
			assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State, "a later worker must acquire the released publication lock")
		})
	}
}

func queueMergeConfidenceRun(t *testing.T, f publicationFixture) *models.CanvasNodeRequest {
	t.Helper()
	require.NoError(t, f.run.QueueMergeConfidenceCheck(f.db, f.pr))
	var request models.CanvasNodeRequest
	require.NoError(t, f.db.Where("type = ? AND spec->'merge_confidence_check'->>'run_id' = ?", models.NodeRequestTypeMergeConfidenceCheck, f.run.ID.String()).First(&request).Error)
	return &request
}

type pausedMergeConfidenceHTTP struct {
	host          *contexts.HTTPContext
	method        string
	started       chan struct{}
	release       chan struct{}
	once          sync.Once
	afterResponse func()
}

func newPausedMergeConfidenceHTTP(t *testing.T, host *contexts.HTTPContext) *pausedMergeConfidenceHTTP {
	t.Helper()
	paused := &pausedMergeConfidenceHTTP{host: host, method: http.MethodPost, started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(paused.resume)
	return paused
}

func (h *pausedMergeConfidenceHTTP) Do(request *http.Request) (*http.Response, error) {
	if request.Method == h.method {
		h.once.Do(func() { close(h.started) })
		select {
		case <-h.release:
		case <-request.Context().Done():
			return nil, request.Context().Err()
		}
	}
	response, err := h.host.Do(request)
	if request.Method == h.method && h.afterResponse != nil {
		h.afterResponse()
	}
	return response, err
}

func (h *pausedMergeConfidenceHTTP) resume() {
	select {
	case <-h.release:
	default:
		close(h.release)
	}
}

func awaitMergeConfidenceDelivery(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("GitHub delivery did not start")
	}
}

func awaitMergeConfidenceResult(t *testing.T, completed <-chan error) error {
	t.Helper()
	select {
	case err := <-completed:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("GitHub publisher did not finish")
		return context.DeadlineExceeded
	}
}

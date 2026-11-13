package workers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func (h *reviewCheckHTTP) checkRequest(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodGet {
		runs := h.checkRuns
		if request.URL.Query().Get("filter") != "all" && len(runs) > 0 {
			runs = runs[len(runs)-1:]
		}
		page, _ := strconv.Atoi(request.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		more := false
		if h.checkRunPageSize > 0 {
			start := min((page-1)*h.checkRunPageSize, len(runs))
			end := min(start+h.checkRunPageSize, len(runs))
			more = end < len(runs)
			runs = runs[start:end]
		}
		body, _ := json.Marshal(map[string]any{"check_runs": runs})
		response := mocks.GitHubResponse(200, string(body))
		if more {
			next := *request.URL
			query := next.Query()
			query.Set("page", strconv.Itoa(page+1))
			next.RawQuery = query.Encode()
			response.Header.Set("Link", "<"+next.String()+`>; rel="next"`)
		}
		return response, nil
	}
	if h.failWrite {
		return mocks.GitHubResponse(503, `{"message":"unavailable"}`), nil
	}
	var run github.CheckRun
	require.NoError(h.t, json.NewDecoder(request.Body).Decode(&run))
	if request.Method == http.MethodPost {
		run.ID = github.Ptr(int64(len(h.checkRuns) + 1))
		h.checkRuns = append(h.checkRuns, &run)
	} else {
		id, err := strconv.ParseInt(request.URL.Path[strings.LastIndex(request.URL.Path, "/")+1:], 10, 64)
		require.NoError(h.t, err)
		run.ID = &id
		require.Positive(h.t, id)
		require.LessOrEqual(h.t, id, int64(len(h.checkRuns)))
		previous := h.checkRuns[id-1]
		run.HeadSHA, run.ExternalID = previous.HeadSHA, previous.ExternalID
		h.checkRuns[id-1] = &run
	}
	h.checkWrites = append(h.checkWrites, run)
	if h.loseCheckResponse {
		h.loseCheckResponse = false
		return nil, fmt.Errorf("connection lost after GitHub accepted the write")
	}
	body, _ := json.Marshal(run)
	return mocks.GitHubResponse(200, string(body)), nil
}

func (f *reviewCheckFixture) applyCheckRun(state, sha string) error {
	f.ctx.Configuration = map[string]any{
		"repository": "hello", "pullNumber": 42, "assessedSha": sha,
		"context": "team/review", "state": state, "reason": "Assessment result",
		"resultUrl": "https://superplane.example/run",
	}
	return (&pulls.SetPullRequestCheckRun{}).Execute(f.ctx)
}

func TestPullRequestCheckRunCompletesSameRun(t *testing.T) {
	for _, result := range []string{"success", "failure", "error"} {
		t.Run(result, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			require.NoError(t, f.processDue())
			require.Len(t, f.http.checkRuns, 1)
			require.Equal(t, "in_progress", f.http.checkRuns[0].GetStatus())
			require.Empty(t, f.http.checkRuns[0].GetConclusion())
			if result == "success" {
				f.http.failPR = true
				require.Error(t, f.processDue())
				require.Equal(t, "in_progress", f.http.checkRuns[0].GetStatus(), "a transient read failure must not finish the assessment")
				f.http.failPR = false
				require.NoError(t, f.processDue())
			}
			f.finishAction()
			require.NoError(t, f.applyCheckRun(result, "abc"))
			require.NoError(t, f.processDue())
			require.Len(t, f.http.checkRuns, 1)
			run := f.http.checkRuns[0]
			require.Equal(t, "completed", run.GetStatus())
			conclusion := result
			if result == "error" {
				conclusion = "failure"
			}
			require.Equal(t, conclusion, run.GetConclusion())
			require.Equal(t, "https://superplane.example/run", run.GetDetailsURL())
			require.Contains(t, run.GetOutput().GetSummary(), "Assessment result")
			require.Nil(t, f.record().WakeAt)
			require.Empty(t, f.http.writes)
			writes := len(f.http.checkWrites)
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			require.NoError(t, f.refresh())
			require.Len(t, f.http.checkWrites, writes, "delayed start cannot reopen a final check")
		})
	}
}

func TestPullRequestCheckRunRecoversInterruptedCreation(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	f.http.checkRunPageSize = 1
	f.http.checkRuns = []*github.CheckRun{{ID: github.Ptr(int64(1)), ExternalID: github.Ptr("another-attempt")}}
	f.http.loseCheckResponse = true
	require.Error(t, f.processDue())
	require.Len(t, f.http.checkRuns, 2)
	f.http.checkRuns = append(f.http.checkRuns, &github.CheckRun{ID: github.Ptr(int64(3)), ExternalID: github.Ptr("latest-unrelated-run")})
	require.NoError(t, f.processDue())
	require.Len(t, f.http.checkRuns, 3, "retry must find the accepted run across pages and older runs")
	require.NoError(t, f.applyCheckRun("success", "abc"))
	require.NoError(t, f.processDue())
	require.Len(t, f.http.checkRuns, 3)
	require.Equal(t, "success", f.http.checkRuns[1].GetConclusion())
}

func TestPullRequestCheckRunRetiresSupersededAttempts(t *testing.T) {
	for _, sha := range []string{"abc", "new-head"} {
		t.Run(sha, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			f.http.loseCheckResponse = true
			require.Error(t, f.processDue())
			old := f.root
			if sha == "abc" {
				retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", old.ID)
				require.NoError(t, err)
				root, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
				require.NoError(t, err)
				f.attempt(root)
			} else {
				f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
			}
			f.http.head = sha
			require.NoError(t, f.applyCheckRun("pending", sha))
			f.http.failWrite = true
			require.Error(t, f.processDue())
			f.http.failWrite = false
			require.NoError(t, f.processDue())
			require.NoError(t, f.processDue())
			require.Len(t, f.http.checkRuns, 2)
			require.Equal(t, "failure", f.http.checkRuns[0].GetConclusion())
			require.Equal(t, "abc", f.http.checkRuns[0].GetHeadSHA())
			require.Equal(t, "in_progress", f.http.checkRuns[1].GetStatus())
			require.Equal(t, sha, f.http.checkRuns[1].GetHeadSHA())
			require.NotEqual(t, f.http.checkRuns[0].GetExternalID(), f.http.checkRuns[1].GetExternalID())
			generation := f.record().Generation
			f.attempt(old)
			require.NoError(t, f.applyCheckRun("success", "abc"))
			require.Equal(t, generation, f.record().Generation)
		})
	}
}

func TestPullRequestCheckRunEndsWithoutFinalResult(t *testing.T) {
	for _, scenario := range []string{"cancelled", "finished", "removed", "closed", "pushed"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			require.NoError(t, f.processDue())
			switch scenario {
			case "cancelled":
				require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateCancelling).Error)
			case "finished":
				require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateFinished).Error)
			case "removed":
				require.NoError(t, f.db.Delete(f.canvas).Error)
			case "closed":
				f.http.closed = true
			case "pushed":
				f.http.head = "new-head"
			}
			f.http.failWrite = true
			require.Error(t, f.processDue())
			f.http.failWrite = false
			require.NoError(t, f.processDue())
			require.Len(t, f.http.checkRuns, 1)
			require.Equal(t, "abc", f.http.checkRuns[0].GetHeadSHA())
			require.Equal(t, "completed", f.http.checkRuns[0].GetStatus())
			require.Equal(t, "failure", f.http.checkRuns[0].GetConclusion())
			require.Nil(t, f.record().WakeAt)
		})
	}
}

func TestPullRequestCheckRunSavesReceiptWhenSupersededDuringCreation(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	entered, resume, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var release sync.Once
	f.http.onWrite = func() { close(entered); <-resume }
	done := make(chan error, 1)
	go func() { defer close(finished); done <- f.refresh() }()
	t.Cleanup(func() { release.Do(func() { close(resume) }); <-finished })
	<-entered
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	// Replacements retain a live delivery lease instead of starting a second write.
	require.NoError(t, f.refresh())
	require.Empty(t, f.http.checkRuns)
	release.Do(func() { close(resume) })
	require.NoError(t, <-done)
	f.http.onWrite = nil
	var payload struct {
		Retired []struct{ CheckRun struct{ ID int64 } }
	}
	require.NoError(t, json.Unmarshal(f.record().Payload, &payload))
	require.EqualValues(t, 1, payload.Retired[0].CheckRun.ID)
	require.NoError(t, f.processDue())
	require.NoError(t, f.processDue())
	require.Len(t, f.http.checkRuns, 2)
	require.Equal(t, "failure", f.http.checkRuns[0].GetConclusion())
	require.Equal(t, "in_progress", f.http.checkRuns[1].GetStatus())
}

func TestPullRequestCheckRunRecoversExpiredDeliveryLease(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	require.NoError(t, f.processDue())
	// Simulate a worker stopping after an accepted creation and before its acknowledgement.
	require.NoError(t, f.integration.store.Update(f.record().Key, func(record *core.OrderedPublication) error {
		var payload map[string]any
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return err
		}
		delete(payload["checkRun"].(map[string]any), "id")
		delete(payload, "delivered")
		payload["delivery"] = map[string]any{"id": uuid.NewString(), "expiresAt": time.Now().Add(-time.Second)}
		var err error
		record.Payload, err = json.Marshal(payload)
		return err
	}))
	require.NoError(t, f.refresh())
	require.Len(t, f.http.checkRuns, 1)
	require.NoError(t, f.applyCheckRun("success", "abc"))
	require.NoError(t, f.processDue())
	require.Len(t, f.http.checkRuns, 1)
	require.Equal(t, "success", f.http.checkRuns[0].GetConclusion())
}

func TestPullRequestCheckRunRetiresWritesThatFinishAfterRecovery(t *testing.T) {
	for _, scenario := range []string{"create", "create with lost response", "update"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			if scenario == "update" {
				require.NoError(t, f.processDue())
				f.http.failPR = true // A transient read failure produces another running update.
			}
			entered, resume, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			var release sync.Once
			f.http.onWrite = func() {
				if calls.Add(1) == 1 {
					close(entered)
					<-resume
				}
			}
			done := make(chan error, 1)
			go func() { defer close(finished); done <- f.refresh() }()
			t.Cleanup(func() { release.Do(func() { close(resume) }); <-finished })
			<-entered
			f.http.failPR = false
			f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
			require.NoError(t, f.applyCheckRun("pending", "abc"))
			require.NoError(t, f.integration.store.Update(f.record().Key, func(record *core.OrderedPublication) error {
				var payload map[string]any
				if err := json.Unmarshal(record.Payload, &payload); err != nil {
					return err
				}
				payload["delivery"].(map[string]any)["expiresAt"] = time.Now().Add(-time.Second)
				var err error
				record.Payload, err = json.Marshal(payload)
				return err
			}))
			require.NoError(t, f.refresh()) // Retire the old attempt while its writer is paused.
			require.NoError(t, f.refresh()) // Start the new attempt's check.
			if scenario == "create with lost response" {
				f.http.loseCheckResponse = true
			}
			release.Do(func() { close(resume) })
			err := <-done
			if scenario == "create" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			f.http.onWrite = nil
			require.NoError(t, f.processDue())
			require.NoError(t, f.refresh())
			var running int
			for _, run := range f.http.checkRuns {
				if run.GetStatus() == "in_progress" {
					running++
					continue
				}
				require.Equal(t, "failure", run.GetConclusion())
			}
			require.Equal(t, 1, running, "a late writer must not leave the superseded attempt running")
		})
	}
}

func TestPullRequestCheckRunPreservesUndeliveredFinalResult(t *testing.T) {
	for _, sha := range []string{"abc", "new-head"} {
		for _, result := range []string{"success", "failure"} {
			t.Run(sha+"/"+result, func(t *testing.T) {
				f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
				require.NoError(t, f.applyCheckRun("pending", "abc"))
				require.NoError(t, f.processDue())
				f.finishAction()
				require.NoError(t, f.applyCheckRun(result, "abc"))
				f.http.failWrite = true
				require.Error(t, f.processDue())
				f.http.failWrite = false
				f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
				f.http.head = sha
				require.NoError(t, f.applyCheckRun("pending", sha))
				require.NoError(t, f.processDue())
				require.NoError(t, f.processDue())
				require.Len(t, f.http.checkRuns, 2)
				require.Equal(t, "abc", f.http.checkRuns[0].GetHeadSHA())
				require.Equal(t, "completed", f.http.checkRuns[0].GetStatus())
				require.Equal(t, result, f.http.checkRuns[0].GetConclusion())
				require.Equal(t, "in_progress", f.http.checkRuns[1].GetStatus())
			})
		}
	}
}

func TestPullRequestCheckRunKeepsFinalResultWhenWriterFinishesAfterRecovery(t *testing.T) {
	f := newPullRequestCheckFixture(t, "github.setPullRequestCheckRun")
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	require.NoError(t, f.processDue())
	f.finishAction()
	require.NoError(t, f.applyCheckRun("success", "abc"))
	entered, resume, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	var release sync.Once
	f.http.onWrite = func() {
		if calls.Add(1) == 1 {
			close(entered)
			<-resume
		}
	}
	done := make(chan error, 1)
	go func() { defer close(finished); done <- f.refresh() }()
	t.Cleanup(func() { release.Do(func() { close(resume) }); <-finished })
	<-entered
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.NoError(t, f.applyCheckRun("pending", "abc"))
	require.NoError(t, f.integration.store.Update(f.record().Key, func(record *core.OrderedPublication) error {
		var payload map[string]any
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return err
		}
		payload["delivery"].(map[string]any)["expiresAt"] = time.Now().Add(-time.Second)
		var err error
		record.Payload, err = json.Marshal(payload)
		return err
	}))
	require.NoError(t, f.refresh()) // Deliver and remove the old result while its writer is paused.
	require.NoError(t, f.refresh()) // Start the new attempt's check.
	require.Equal(t, "success", f.http.checkRuns[0].GetConclusion())
	release.Do(func() { close(resume) })
	require.NoError(t, <-done)
	f.http.onWrite = nil
	require.NoError(t, f.processDue())
	require.NoError(t, f.refresh())
	require.Len(t, f.http.checkRuns, 2)
	require.Equal(t, "completed", f.http.checkRuns[0].GetStatus())
	require.Equal(t, "success", f.http.checkRuns[0].GetConclusion(), "a late receipt must not replace the delivered result")
	require.Equal(t, "in_progress", f.http.checkRuns[1].GetStatus())
}

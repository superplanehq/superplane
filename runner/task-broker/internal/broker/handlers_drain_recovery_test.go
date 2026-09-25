package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestDrainRunnersUnhealthyKeepsClaimedTaskBusyWithoutTerminating(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", "runner-1", 0)
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateBusy || got.Runners[0].ActiveTaskID != taskID {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 0 {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusClaimed || task.RunnerID != "runner-1" || task.RunnerTerminationRequestedAt != nil {
		t.Fatalf("task after drain: %#v", task)
	}
}

func TestUnhealthyDrainAllowsPassedCompletionWhileBusy(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	const runnerID = "runner-1"
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", runnerID, 0)
	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim(runnerID) {
		t.Fatal("expected claim to start")
	}
	drain.FinishClaim(runnerID, taskID)

	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: drain}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{runnerID},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)
	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateBusy || got.Runners[0].ActiveTaskID != taskID {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 0 {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusClaimed || task.RunnerTerminationRequestedAt != nil {
		t.Fatalf("task after drain: %#v", task)
	}

	accessToken := mintRunnerAccessToken(t, ctx, st, "fleet-a", runnerID)
	postComplete(t, ts, taskID, accessToken, api.CompleteTaskRequest{
		RunnerID: runnerID,
		ExitCode: 0,
	}, http.StatusNoContent)

	task, err = st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusSucceeded {
		t.Fatalf("status: got %s want succeeded", task.Status)
	}
	if task.ErrorMessage != "" {
		t.Fatalf("error message: %q", task.ErrorMessage)
	}

	got = postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{runnerID},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)
	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("runners: %#v", got.Runners)
	}
}

func TestDrainRunnersUnhealthyConfirmationFailsLostRunnerTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", "runner-1", 1)
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	_ = postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)
	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:              "fleet-a",
		RunnerIDs:            []string{"runner-1"},
		Reason:               api.DrainReasonUnhealthy,
		TerminationConfirmed: true,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 1 ||
		got.RecoveredTasks[0].TaskID != taskID ||
		got.RecoveredTasks[0].State != api.RunnerTaskRecoveryStateFailed {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusFailed || task.ErrorMessage != "runner lost before completion" {
		t.Fatalf("task after recovery: %#v", task)
	}
}

func TestDrainRunnersUnhealthyConfirmationCancelsStopPendingTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", "runner-1", 0)
	if _, _, err := st.RequestCancelTask(ctx, taskID); err != nil {
		t.Fatal(err)
	}
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	_ = postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)
	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:              "fleet-a",
		RunnerIDs:            []string{"runner-1"},
		Reason:               api.DrainReasonUnhealthy,
		TerminationConfirmed: true,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 1 ||
		got.RecoveredTasks[0].TaskID != taskID ||
		got.RecoveredTasks[0].State != api.RunnerTaskRecoveryStateCanceled {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusCanceled || task.Output != "canceled (runner lost while stop pending)" {
		t.Fatalf("task after recovery: %#v", task)
	}
}

func TestTerminationPendingAcceptsLatePassedCompletion(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", "runner-1", 0)
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-a", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	accessToken := mintRunnerAccessToken(t, ctx, st, "fleet-a", "runner-1")
	postComplete(t, ts, taskID, accessToken, api.CompleteTaskRequest{
		RunnerID: "runner-1",
		ExitCode: 0,
	}, http.StatusNoContent)

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusSucceeded || task.RunnerTerminationRequestedAt != nil {
		t.Fatalf("task after completion: %#v", task)
	}
}

func TestSoftDrainUnhealthyBusyRunnersKeepsPersistedClaim(t *testing.T) {
	srv := &Server{RunnerDrain: NewRunnerDrainHub()}
	statuses := []api.DrainRunnerStatus{{
		RunnerID:     "runner-1",
		State:        api.DrainRunnerStateBusy,
		ActiveTaskID: "task-1",
	}}

	got := srv.softDrainUnhealthyBusyRunners(statuses, map[string]string{"runner-1": "task-1"})
	if len(got) != 1 || got[0].State != api.DrainRunnerStateBusy || got[0].ActiveTaskID != "task-1" {
		t.Fatalf("runners: %#v", got)
	}
}

func TestSoftDrainUnhealthyBusyRunnersDrainsStaleHubOnly(t *testing.T) {
	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim("runner-1") {
		t.Fatal("expected claim to start")
	}
	drain.FinishClaim("runner-1", "task-stale")
	srv := &Server{RunnerDrain: drain}
	statuses := []api.DrainRunnerStatus{{
		RunnerID:     "runner-1",
		State:        api.DrainRunnerStateBusy,
		ActiveTaskID: "task-stale",
	}}

	got := srv.softDrainUnhealthyBusyRunners(statuses, map[string]string{})
	if len(got) != 1 || got[0].State != api.DrainRunnerStateDrained || got[0].ActiveTaskID != "" {
		t.Fatalf("runners: %#v", got)
	}
}

func TestDrainRunnersUnhealthyRefreshesClaimsAfterStickyDrain(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	const runnerID = "runner-1"
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", runnerID, 0)
	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim(runnerID) {
		t.Fatal("expected claim to start")
	}
	drain.FinishClaim(runnerID, taskID)

	spy := &claimedAfterDrainStore{Store: st, drain: drain, runnerID: runnerID}
	srv := &Server{Store: spy, TaskNotify: NewWaitHub(), RunnerDrain: drain}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{runnerID},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)

	if !spy.claimedWhileDraining {
		t.Fatal("ClaimedTaskIDsByRunners must run after sticky Drain")
	}
	if spy.claimedBeforeDraining {
		t.Fatal("ClaimedTaskIDsByRunners must not run before sticky Drain")
	}
	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateBusy || got.Runners[0].ActiveTaskID != taskID {
		t.Fatalf("runners: %#v", got.Runners)
	}
}

func TestDrainRunnersScaleDownDoesNotRecoverClaimedTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := createDrainRecoveryClaimedTask(t, ctx, st, "fleet-a", "runner-1", 0)
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonScaleDown,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateBusy || got.Runners[0].ActiveTaskID != taskID {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 0 {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusClaimed || task.RunnerID != "runner-1" {
		t.Fatalf("task after drain: %#v", task)
	}
}

func TestDrainRunnersUnhealthyDrainsStaleActiveTaskWithoutPersistedClaim(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim("runner-1") {
		t.Fatal("expected claim to start")
	}
	drain.FinishClaim("runner-1", "task-stale")
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: drain}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateDrained || got.Runners[0].ActiveTaskID != "" {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 0 {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}

	statuses := drain.Drain("fleet-a", []string{"runner-1"})
	if len(statuses) != 1 || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("runner drain status after recovery: %#v", statuses)
	}
}

func TestDrainRunnersUnhealthyKeepsInProgressClaimBusyWithoutPersistedClaim(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim("runner-1") {
		t.Fatal("expected claim to start")
	}
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: drain}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	got := postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReasonUnhealthy,
	}, http.StatusOK)

	if len(got.Runners) != 1 || got.Runners[0].State != api.DrainRunnerStateBusy {
		t.Fatalf("runners: %#v", got.Runners)
	}
	if len(got.RecoveredTasks) != 0 {
		t.Fatalf("recovered tasks: %#v", got.RecoveredTasks)
	}
}

func TestDrainRunnersRejectsUnknownReason(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	_ = postDrain(t, ts, api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
		Reason:    api.DrainReason("maintenance"),
	}, http.StatusBadRequest)
}

func createDrainRecoveryClaimedTask(t *testing.T, ctx context.Context, st taskCreator, fleetID, runnerID string, infraRetryCount int) string {
	t.Helper()
	taskID := uuid.NewString()
	if err := st.CreateTask(ctx, &models.Task{
		ID:              taskID,
		FleetID:         fleetID,
		Status:          models.StatusQueued,
		CreatedAt:       time.Now().UTC(),
		WebhookURL:      "https://example.com/hook",
		Command:         []string{"echo", "hi"},
		InfraRetryCount: infraRetryCount,
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := st.ClaimTask(ctx, runnerID, fleetID, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil {
		t.Fatal("expected task to be claimed")
	}
	return taskID
}

type taskCreator interface {
	CreateTask(context.Context, *models.Task) error
	ClaimTask(context.Context, string, string, time.Duration) (*models.Task, error)
}

type claimedAfterDrainStore struct {
	taskstore.Store
	drain                 *RunnerDrainHub
	runnerID              string
	claimedBeforeDraining bool
	claimedWhileDraining  bool
}

func (s *claimedAfterDrainStore) ClaimedTaskIDsByRunners(ctx context.Context, fleetID string, runnerIDs []string) (map[string]string, error) {
	if s.drain.IsDraining(s.runnerID) {
		s.claimedWhileDraining = true
	} else {
		s.claimedBeforeDraining = true
	}
	return s.Store.ClaimedTaskIDsByRunners(ctx, fleetID, runnerIDs)
}

func postDrain(t *testing.T, ts *httptest.Server, reqBody api.DrainRunnersRequest, wantStatus int) api.DrainRunnersResponse {
	t.Helper()
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/runners/drain", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("status: got %d want %d", resp.StatusCode, wantStatus)
	}
	if wantStatus != http.StatusOK {
		return api.DrainRunnersResponse{}
	}
	var got api.DrainRunnersResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func postComplete(t *testing.T, ts *httptest.Server, taskID, accessToken string, reqBody api.CompleteTaskRequest, wantStatus int) {
	t.Helper()
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/"+taskID+"/complete", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("status: got %d want %d", resp.StatusCode, wantStatus)
	}
}

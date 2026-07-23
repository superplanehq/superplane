package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/opaquetoken"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestRunnerRegistrationIsSingleUseAndIssuesScopedCredential(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC()
	registrationToken := "registration-secret"
	accessToken := "runner-access-secret"
	if err := st.CreateRunnerRegistration(ctx, &brokermodels.RunnerRegistration{
		TokenHash: opaquetoken.Hash(registrationToken),
		FleetID:   "fleet-a",
		ExpiresAt: now.Add(time.Minute),
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.ExchangeRunnerRegistration(
		ctx,
		opaquetoken.Hash(registrationToken),
		"i-012345",
		"wrong-fleet",
		opaquetoken.Hash(accessToken),
		now,
	); err != taskstore.ErrInvalidRunnerRegistration {
		t.Fatalf("wrong-fleet exchange error = %v, want invalid registration", err)
	}
	if err := st.ExchangeRunnerRegistration(
		ctx,
		opaquetoken.Hash(registrationToken),
		"i-012345",
		"fleet-a",
		opaquetoken.Hash(accessToken),
		now,
	); err != nil {
		t.Fatal(err)
	}
	credential, err := st.GetRunnerByAccessTokenHash(ctx, opaquetoken.Hash(accessToken))
	if err != nil {
		t.Fatal(err)
	}
	if credential == nil || credential.RunnerID != "i-012345" || credential.FleetID != "fleet-a" {
		t.Fatalf("credential = %#v", credential)
	}
	if err := st.ExchangeRunnerRegistration(
		ctx,
		opaquetoken.Hash(registrationToken),
		"i-other",
		"fleet-a",
		opaquetoken.Hash("other-access"),
		now,
	); err != taskstore.ErrInvalidRunnerRegistration {
		t.Fatalf("second exchange error = %v, want invalid registration", err)
	}
}

func TestPostgresStoreFleetsAndTasks(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:          "fleet-a",
		Provisioner: "aws",
		Arch:        "amd64",
		Size:        "t3.micro",
		CreatedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Provisioner != "aws" || got.Arch != "amd64" || got.Size != "t3.micro" {
		t.Fatalf("get fleet: %#v", got)
	}

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:          "fleet-a",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	got, err = st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provisioner != "local" || got.Size != "local" {
		t.Fatalf("upsert fleet: %#v", got)
	}

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-b",
		Arch:      "arm64",
		Size:      "t4g.micro",
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	list, err := st.ListFleets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list fleets: %d", len(list))
	}

	task := &models.Task{
		ID:         "task-1",
		FleetID:    "fleet-a",
		Command:    []string{"echo", "hi"},
		WebhookURL: "https://caller.example/hook",
		Status:     models.StatusQueued,
		CreatedAt:  now,
	}
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	gotTask, err := st.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if gotTask == nil || gotTask.FleetID != "fleet-a" {
		t.Fatalf("get task: %#v", gotTask)
	}

	claimed, err := st.ClaimTask(ctx, "runner-1", "fleet-a", 60*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != "task-1" {
		t.Fatalf("claim: %#v", claimed)
	}

	if err := st.DeleteFleet(ctx, "fleet-b"); err != nil {
		t.Fatal(err)
	}
	list, err = st.ListFleets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "fleet-a" {
		t.Fatalf("after delete fleet-b: %#v", list)
	}
}

func TestPostgresStoreCountTasksByFleet(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id, fleetID string, status models.TaskStatus) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    fleetID,
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("a-queued-1", "fleet-a", models.StatusQueued)
	create("a-queued-2", "fleet-a", models.StatusQueued)
	create("a-claimed-1", "fleet-a", models.StatusClaimed)
	create("a-done-1", "fleet-a", models.StatusSucceeded)
	create("a-failed-1", "fleet-a", models.StatusFailed)
	create("a-canceled-1", "fleet-a", models.StatusCanceled)

	create("b-queued-1", "fleet-b", models.StatusQueued)
	create("b-claimed-1", "fleet-b", models.StatusClaimed)
	create("b-claimed-2", "fleet-b", models.StatusClaimed)
	create("b-claimed-3", "fleet-b", models.StatusClaimed)

	qa, ca, err := st.CountTasksByFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if qa != 2 || ca != 1 {
		t.Fatalf("fleet-a counts: got queued=%d claimed=%d, want 2/1", qa, ca)
	}

	qb, cb, err := st.CountTasksByFleet(ctx, "fleet-b")
	if err != nil {
		t.Fatal(err)
	}
	if qb != 1 || cb != 3 {
		t.Fatalf("fleet-b counts: got queued=%d claimed=%d, want 1/3", qb, cb)
	}

	qc, cc, err := st.CountTasksByFleet(ctx, "fleet-missing")
	if err != nil {
		t.Fatal(err)
	}
	if qc != 0 || cc != 0 {
		t.Fatalf("fleet-missing counts: got queued=%d claimed=%d, want 0/0", qc, cc)
	}

	if _, _, err := st.CountTasksByFleet(ctx, ""); err == nil {
		t.Fatalf("expected error for empty fleet id")
	}
}

func TestPostgresStoreOldestQueuedTaskCreatedAt(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	oldest := now.Add(-2 * time.Minute)

	create := func(id, fleetID string, status models.TaskStatus, createdAt time.Time) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    fleetID,
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  createdAt,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("queued-oldest", "fleet-a", models.StatusQueued, oldest)
	create("queued-newer", "fleet-a", models.StatusQueued, now.Add(-30*time.Second))
	create("claimed-older", "fleet-a", models.StatusClaimed, now.Add(-10*time.Minute))
	create("other-fleet-queued", "fleet-b", models.StatusQueued, now.Add(-20*time.Minute))

	got, err := st.OldestQueuedTaskCreatedAt(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.Equal(oldest) {
		t.Fatalf("oldest queued task: got %v, want %v", got, oldest)
	}

	got, err = st.OldestQueuedTaskCreatedAt(ctx, "fleet-missing")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("missing fleet oldest queued task: got %v, want nil", got)
	}

	if _, err := st.OldestQueuedTaskCreatedAt(ctx, ""); err == nil {
		t.Fatalf("expected error for empty fleet id")
	}
}

func TestPostgresStoreClaimedRunnerIDsByFleet(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id, fleetID string, status models.TaskStatus, runnerID string) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    fleetID,
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  now,
			RunnerID:   runnerID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("claimed-1", "fleet-a", models.StatusClaimed, "i-aaa")
	create("claimed-2", "fleet-a", models.StatusClaimed, "i-bbb")
	create("claimed-same-runner", "fleet-a", models.StatusClaimed, "i-aaa")
	create("claimed-empty-runner", "fleet-a", models.StatusClaimed, "")
	create("queued", "fleet-a", models.StatusQueued, "i-queued")
	create("other-fleet", "fleet-b", models.StatusClaimed, "i-other")
	create("done", "fleet-a", models.StatusSucceeded, "i-done")

	got, err := st.ClaimedRunnerIDsByFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "i-aaa" || got[1] != "i-bbb" {
		t.Fatalf("claimed runner ids: got %#v want [i-aaa i-bbb]", got)
	}

	if _, err := st.ClaimedRunnerIDsByFleet(ctx, ""); err == nil {
		t.Fatalf("expected error for empty fleet id")
	}
}

func TestPostgresStoreClaimedTaskIDsByRunners(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id, fleetID string, status models.TaskStatus, runnerID string, createdAt time.Time) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    fleetID,
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  createdAt,
			RunnerID:   runnerID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("task-newer", "fleet-a", models.StatusClaimed, "i-aaa", now.Add(time.Second))
	create("task-older", "fleet-a", models.StatusClaimed, "i-aaa", now)
	create("task-bbb", "fleet-a", models.StatusClaimed, "i-bbb", now)
	create("queued", "fleet-a", models.StatusQueued, "i-aaa", now)
	create("other-fleet", "fleet-b", models.StatusClaimed, "i-aaa", now)
	create("done", "fleet-a", models.StatusSucceeded, "i-ccc", now)

	got, err := st.ClaimedTaskIDsByRunners(ctx, "fleet-a", []string{"i-aaa", "i-bbb", "i-missing", "i-aaa", ""})
	if err != nil {
		t.Fatal(err)
	}
	if got["i-aaa"] != "task-older" {
		t.Fatalf("i-aaa task: got %q want task-older", got["i-aaa"])
	}
	if got["i-bbb"] != "task-bbb" {
		t.Fatalf("i-bbb task: got %q want task-bbb", got["i-bbb"])
	}
	if _, ok := got["i-missing"]; ok {
		t.Fatalf("unexpected missing runner task: %#v", got)
	}

	empty, err := st.ClaimedTaskIDsByRunners(ctx, "fleet-a", []string{"", " "})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty runner ids: %#v", empty)
	}

	if _, err := st.ClaimedTaskIDsByRunners(ctx, "", []string{"i-aaa"}); err == nil {
		t.Fatalf("expected error for empty fleet id")
	}
}

func TestPostgresStoreListActiveTasks(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id string, status models.TaskStatus) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    "fleet-a",
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("queued-1", models.StatusQueued)
	create("claimed-1", models.StatusClaimed)
	create("done-1", models.StatusSucceeded)
	create("failed-1", models.StatusFailed)
	create("canceled-1", models.StatusCanceled)

	active, err := st.ListActiveTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("active tasks: got %d want 2: %#v", len(active), active)
	}
	if active[0].ID != "queued-1" || active[1].ID != "claimed-1" {
		t.Fatalf("order/ids: %#v", active)
	}
}

func TestUnclaimTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-unclaim", Provisioner: "test", Arch: "amd64", Size: "t3.micro",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	task := &models.Task{
		ID:         uuid.New().String(),
		FleetID:    "fleet-unclaim",
		Status:     models.StatusQueued,
		CreatedAt:  time.Now().UTC(),
		WebhookURL: "https://example.com/hook",
		Commands:   models.CommandList{{Command: "echo hi"}},
	}
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Claim the task.
	claimed, err := st.ClaimTask(ctx, "runner-1", "fleet-unclaim", 5*time.Minute)
	if err != nil || claimed == nil {
		t.Fatalf("ClaimTask: task=%v err=%v", claimed, err)
	}
	if claimed.Status != models.StatusClaimed {
		t.Fatalf("expected claimed, got %s", claimed.Status)
	}

	// Unclaim — task should go back to queued.
	unclaimed, err := st.UnclaimTask(ctx, claimed.ID, "runner-1")
	if err != nil {
		t.Fatal("UnclaimTask:", err)
	}
	if !unclaimed {
		t.Fatal("UnclaimTask: expected unclaimed=true")
	}
	got, err := st.GetTask(ctx, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusQueued {
		t.Fatalf("expected queued after unclaim, got %s", got.Status)
	}
	if got.RunnerID != "" || got.ClaimedAt != nil || got.LeaseUntil != nil {
		t.Fatalf("expected runner/claimed_at/lease_until cleared, got runner=%q claimedAt=%v lease=%v",
			got.RunnerID, got.ClaimedAt, got.LeaseUntil)
	}

	// Wrong runner — should be a no-op.
	if _, err2 := st.ClaimTask(ctx, "runner-1", "fleet-unclaim", 5*time.Minute); err2 != nil {
		t.Fatal(err2)
	}
	unclaimed, err = st.UnclaimTask(ctx, claimed.ID, "runner-WRONG")
	if err != nil {
		t.Fatal("UnclaimTask wrong runner:", err)
	}
	if unclaimed {
		t.Fatal("UnclaimTask wrong runner: expected unclaimed=false")
	}
	got2, _ := st.GetTask(ctx, claimed.ID)
	if got2.Status != models.StatusClaimed {
		t.Fatalf("wrong-runner unclaim should be no-op, got %s", got2.Status)
	}
}

func TestUnclaimTaskDoesNotRequeueTerminationPendingTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}

	unclaimed, err := st.UnclaimTask(ctx, taskID, "runner-1")
	if err != nil {
		t.Fatal(err)
	}
	if unclaimed {
		t.Fatal("termination-pending task should not be unclaimed")
	}
	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusClaimed || got.RunnerID != "runner-1" || got.RunnerTerminationRequestedAt == nil {
		t.Fatalf("task after unclaim: %#v", got)
	}
}

func TestCompleteTaskRequeuesRunnerInfraFailureOnce(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     "runner-1",
		ExitCode:     1,
		ErrorMessage: "context canceled",
		FailureKind:  api.FailureKindRunnerInfra,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeRequeued {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeRequeued)
	}
	if result.Task.Status != models.StatusQueued {
		t.Fatalf("status: got %s want queued", result.Task.Status)
	}
	if result.Task.InfraRetryCount != 1 {
		t.Fatalf("infra retry count: got %d want 1", result.Task.InfraRetryCount)
	}
	if result.Task.RunnerID != "" || result.Task.ClaimedAt != nil || result.Task.LeaseUntil != nil {
		t.Fatalf("expected claim fields cleared, got runner=%q claimed=%v lease=%v",
			result.Task.RunnerID, result.Task.ClaimedAt, result.Task.LeaseUntil)
	}
	if result.Task.ErrorMessage != "" || result.Task.ResultJSON != "" || result.Task.ExitCode != nil {
		t.Fatalf("expected terminal fields cleared, got exit=%v error=%q result=%q",
			result.Task.ExitCode, result.Task.ErrorMessage, result.Task.ResultJSON)
	}
	if len(result.Task.Environment) != 1 ||
		result.Task.Environment[0].Name != "BASE_URL" ||
		result.Task.Environment[0].Value != "http://example.test" {
		t.Fatalf("expected environment preserved on requeue, got %#v", result.Task.Environment)
	}

	retried, err := st.ClaimTask(ctx, "runner-2", "fleet-retry", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if retried == nil {
		t.Fatal("expected retried task to be claimed")
	}
	if retried.ID != taskID {
		t.Fatalf("retried task ID: got %s want %s", retried.ID, taskID)
	}
	if len(retried.Environment) != 1 ||
		retried.Environment[0].Name != "BASE_URL" ||
		retried.Environment[0].Value != "http://example.test" {
		t.Fatalf("expected environment preserved on retried claim, got %#v", retried.Environment)
	}
}

func TestCompleteTaskDoesNotRequeueRunnerInfraFailureTwice(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 1)
	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     "runner-1",
		ExitCode:     1,
		ErrorMessage: "context canceled",
		FailureKind:  api.FailureKindRunnerInfra,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}
	if result.Task.Status != models.StatusFailed {
		t.Fatalf("status: got %s want failed", result.Task.Status)
	}
	if result.Task.InfraRetryCount != 1 {
		t.Fatalf("infra retry count changed: got %d want 1", result.Task.InfraRetryCount)
	}
}

func TestCompleteTaskDoesNotRequeueNormalFailure(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     "runner-1",
		ExitCode:     1,
		ErrorMessage: "script failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}
	if result.Task.Status != models.StatusFailed {
		t.Fatalf("status: got %s want failed", result.Task.Status)
	}
	if result.Task.InfraRetryCount != 0 {
		t.Fatalf("infra retry count: got %d want 0", result.Task.InfraRetryCount)
	}
}

func TestCompleteTaskDoesNotRequeueCanceledInfraFailure(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     "runner-1",
		ExitCode:     130,
		ErrorMessage: "context canceled",
		FailureKind:  api.FailureKindRunnerInfra,
		Canceled:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}
	if result.Task.Status != models.StatusCanceled {
		t.Fatalf("status: got %s want canceled", result.Task.Status)
	}
	if result.Task.InfraRetryCount != 0 {
		t.Fatalf("infra retry count: got %d want 0", result.Task.InfraRetryCount)
	}
}

func TestCompleteTaskDoesNotRequeueCancelRequestedInfraFailure(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	_, outcome, err := st.RequestCancelTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != taskstore.CancelOutcomeCancelRequested {
		t.Fatalf("cancel outcome: got %s want %s", outcome, taskstore.CancelOutcomeCancelRequested)
	}

	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     "runner-1",
		ExitCode:     1,
		ErrorMessage: "context canceled",
		FailureKind:  api.FailureKindRunnerInfra,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}
	if result.Task.Status != models.StatusFailed {
		t.Fatalf("status: got %s want failed", result.Task.Status)
	}
	if result.Task.InfraRetryCount != 0 {
		t.Fatalf("infra retry count: got %d want 0", result.Task.InfraRetryCount)
	}
}

func TestCompleteTaskTreatsSameRunnerTerminalCompletionAsIdempotent(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:       taskID,
		RunnerID: "runner-1",
		ExitCode: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}

	duplicate, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:       taskID,
		RunnerID: "runner-1",
		ExitCode: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Outcome != taskstore.CompleteTaskOutcomeAlreadyTerminal {
		t.Fatalf("duplicate outcome: got %s want %s", duplicate.Outcome, taskstore.CompleteTaskOutcomeAlreadyTerminal)
	}
	if duplicate.Task.Status != models.StatusSucceeded {
		t.Fatalf("duplicate status: got %s want succeeded", duplicate.Task.Status)
	}
}

func TestMarkLostRunnerTasksTerminatingLeavesTaskClaimed(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	terminations, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(terminations) != 1 {
		t.Fatalf("terminations: %#v", terminations)
	}
	if terminations[0].ID != taskID || terminations[0].RunnerID != "runner-1" {
		t.Fatalf("termination: %#v", terminations[0])
	}

	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusClaimed || got.RunnerID != "runner-1" || got.RunnerTerminationRequestedAt == nil {
		t.Fatalf("task after mark: %#v", got)
	}
}

func TestTerminationPendingTaskIsNotRequeuedByLeaseReaper(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	taskID := uuid.NewString()
	if err := st.CreateTask(ctx, &models.Task{
		ID:                           taskID,
		FleetID:                      "fleet-retry",
		Status:                       models.StatusClaimed,
		CreatedAt:                    now,
		ClaimedAt:                    &now,
		LeaseUntil:                   ptrTime(now.Add(-time.Minute)),
		RunnerID:                     "runner-1",
		WebhookURL:                   "https://example.com/hook",
		Commands:                     models.CommandList{{Command: "echo hi"}},
		RunnerTerminationRequestedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}

	requeued, canceled, err := st.ReapExpiredLeases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(requeued) != 0 || len(canceled) != 0 {
		t.Fatalf("reaped termination pending task: requeued=%#v canceled=%#v", requeued, canceled)
	}
	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusClaimed || got.RunnerID != "runner-1" || got.RunnerTerminationRequestedAt == nil {
		t.Fatalf("task after reap: %#v", got)
	}
}

func TestTerminationPendingTaskAcceptsLateCompletion(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 1)
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}

	result, err := st.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:       taskID,
		RunnerID: "runner-1",
		ExitCode: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != taskstore.CompleteTaskOutcomeTerminal {
		t.Fatalf("outcome: got %s want %s", result.Outcome, taskstore.CompleteTaskOutcomeTerminal)
	}

	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusSucceeded {
		t.Fatalf("status: got %s want succeeded", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("exit code: %#v", got.ExitCode)
	}
	if got.RunnerTerminationRequestedAt != nil {
		t.Fatalf("termination marker should be cleared: %v", got.RunnerTerminationRequestedAt)
	}
}

func TestFinalizeTerminatedRunnerTasksFailsPendingTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 1)
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}
	recoveries, err := st.FinalizeTerminatedRunnerTasks(ctx, "fleet-retry", []string{"runner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveries) != 1 || recoveries[0].ID != taskID || recoveries[0].Status != models.StatusFailed {
		t.Fatalf("recoveries: %#v", recoveries)
	}
	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusFailed {
		t.Fatalf("status: got %s want failed", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != 1 {
		t.Fatalf("exit code: %#v", got.ExitCode)
	}
	if got.ErrorMessage != "runner lost before completion" {
		t.Fatalf("error message: %q", got.ErrorMessage)
	}
	if got.RunnerTerminationRequestedAt != nil {
		t.Fatalf("termination marker should be cleared: %v", got.RunnerTerminationRequestedAt)
	}
	if got.RunnerID != "" || got.ClaimedAt != nil || got.LeaseUntil != nil {
		t.Fatalf("expected claim fields cleared, got runner=%q claimed=%v lease=%v",
			got.RunnerID, got.ClaimedAt, got.LeaseUntil)
	}
	if len(got.Environment) != 0 {
		t.Fatalf("expected environment cleared, got %#v", got.Environment)
	}
}

func TestFinalizeTerminatedRunnerTasksCancelsStopPendingTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	_, outcome, err := st.RequestCancelTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != taskstore.CancelOutcomeCancelRequested {
		t.Fatalf("cancel outcome: got %s want %s", outcome, taskstore.CancelOutcomeCancelRequested)
	}
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}
	recoveries, err := st.FinalizeTerminatedRunnerTasks(ctx, "fleet-retry", []string{"runner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveries) != 1 {
		t.Fatalf("recoveries: %#v", recoveries)
	}
	if recoveries[0].ID != taskID || recoveries[0].Status != models.StatusCanceled {
		t.Fatalf("recovery: %#v", recoveries[0])
	}

	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusCanceled {
		t.Fatalf("status: got %s want canceled", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != 130 {
		t.Fatalf("exit code: %#v", got.ExitCode)
	}
	if got.Output != "canceled (runner lost while stop pending)" {
		t.Fatalf("output: %q", got.Output)
	}
	if got.CancelRequested {
		t.Fatal("cancel requested should be cleared")
	}
}

func TestFinalizeTerminatedRunnerTasksIsIdempotent(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()

	taskID := createClaimedTask(t, ctx, st, "runner-1", 0)
	if _, err := st.MarkLostRunnerTasksTerminating(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.FinalizeTerminatedRunnerTasks(ctx, "fleet-retry", []string{"runner-1"}); err != nil {
		t.Fatal(err)
	}
	recoveries, err := st.FinalizeTerminatedRunnerTasks(ctx, "fleet-retry", []string{"runner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveries) != 0 {
		t.Fatalf("second finalize should be empty: %#v", recoveries)
	}
	got, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StatusFailed || got.ErrorMessage != "runner lost before completion" {
		t.Fatalf("task after second finalize: %#v", got)
	}
}

func createClaimedTask(t *testing.T, ctx context.Context, st *taskstore.PostgresStore, runnerID string, infraRetryCount int) string {
	t.Helper()
	taskID := uuid.NewString()
	now := time.Now().UTC()
	if err := st.CreateTask(ctx, &models.Task{
		ID:              taskID,
		FleetID:         "fleet-retry",
		Status:          models.StatusQueued,
		CreatedAt:       now,
		WebhookURL:      "https://example.com/hook",
		Commands:        models.CommandList{{Command: "echo hi"}},
		Environment:     []models.EnvironmentVariable{{Name: "BASE_URL", Value: "http://example.test"}},
		InfraRetryCount: infraRetryCount,
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := st.ClaimTask(ctx, runnerID, "fleet-retry", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil {
		t.Fatal("expected task to be claimed")
	}
	return taskID
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

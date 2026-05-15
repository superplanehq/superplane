package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

func TestSQLite_CreateClaimComplete(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"echo", "hi"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != task.ID {
		t.Fatalf("claim: got %+v", got)
	}
	if got.Status != models.StatusClaimed || got.RunnerID != "runner-1" {
		t.Fatalf("claim state: %+v", got)
	}

	done, err := s.CompleteTask(ctx, task.ID, "runner-1", 0, "hello\n", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != models.StatusSucceeded {
		t.Fatalf("want succeeded got %s", done.Status)
	}

	empty, err := s.ClaimTask(ctx, "runner-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if empty != nil {
		t.Fatalf("expected no task, got %+v", empty)
	}
}

func TestSQLite_ResultJSONRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"echo", "hi"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimTask(ctx, "runner-1", 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	payload := `{"answer":42}`
	done, err := s.CompleteTask(ctx, task.ID, "runner-1", 0, "hello\n", payload, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if done.ResultJSON != payload {
		t.Fatalf("ResultJSON: got %q want %q", done.ResultJSON, payload)
	}
	reloaded, err := s.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ResultJSON != payload {
		t.Fatalf("reload ResultJSON: got %q want %q", reloaded.ResultJSON, payload)
	}
}

func TestSQLite_CreateCommandsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Commands:      []string{`echo 'hello world'`, `echo 'second'`},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected task")
	}
	if len(got.Commands) != 2 || got.Commands[0] != `echo 'hello world'` || got.Commands[1] != `echo 'second'` {
		t.Fatalf("commands: %#v", got.Commands)
	}
	if len(got.Command) != 0 {
		t.Fatalf("command argv should be empty, got %#v", got.Command)
	}
}

func TestSQLite_ReapExpiredLeases(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"sleep", "999"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimTask(ctx, "runner-1", time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	// Claim sets lease from max(runner, execution+buffer); force expiry for this test.
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET lease_until = ? WHERE id = ?`,
		time.Now().Unix()-1, task.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	n, canceled, err := s.ReapExpiredLeases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("reap count: %d", n)
	}
	if len(canceled) != 0 {
		t.Fatalf("unexpected canceled from reap: %v", canceled)
	}

	again, err := s.ClaimTask(ctx, "runner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if again == nil || again.ID != task.ID {
		t.Fatalf("expected task back in queue, got %+v", again)
	}
}

func TestSQLite_ClaimLeaseUsesMaxOfRunnerLeaseAndExecutionWindow(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	execSec := 3600
	task := &models.Task{
		ID:                      uuid.NewString(),
		Command:                 []string{"echo", "x"},
		WebhookURL:              "https://example.com/hook",
		Status:                  models.StatusQueued,
		CreatedAt:               time.Now().UTC(),
		ExecutionMode:           models.ExecutionHost,
		ExecutionTimeoutSeconds: &execSec,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.LeaseUntil == nil {
		t.Fatalf("claim: %+v", got)
	}
	now := time.Now().Unix()
	runnerEnd := now + int64((10 * time.Minute).Seconds())
	taskEnd := now + int64(execSec+api.LeaseBufferSeconds)
	wantLease := runnerEnd
	if taskEnd > wantLease {
		wantLease = taskEnd
	}
	gotUnix := got.LeaseUntil.Unix()
	if gotUnix < wantLease-2 || gotUnix > wantLease+5 {
		t.Fatalf("lease_until=%d want ~%d (runnerEnd=%d taskEnd=%d)", gotUnix, wantLease, runnerEnd, taskEnd)
	}
}

func TestSQLite_ClaimLeaseRunnerWinsWhenLongerThanExecution(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	execSec := 30
	task := &models.Task{
		ID:                      uuid.NewString(),
		Command:                 []string{"echo", "x"},
		WebhookURL:              "https://example.com/hook",
		Status:                  models.StatusQueued,
		CreatedAt:               time.Now().UTC(),
		ExecutionMode:           models.ExecutionHost,
		ExecutionTimeoutSeconds: &execSec,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.LeaseUntil == nil {
		t.Fatalf("claim: %+v", got)
	}
	now := time.Now().Unix()
	runnerEnd := now + int64((10 * time.Minute).Seconds())
	taskEnd := now + int64(execSec+api.LeaseBufferSeconds)
	wantLease := runnerEnd
	if taskEnd > wantLease {
		wantLease = taskEnd
	}
	gotUnix := got.LeaseUntil.Unix()
	if gotUnix < wantLease-2 || gotUnix > wantLease+5 {
		t.Fatalf("lease_until=%d want ~%d", gotUnix, wantLease)
	}
}

func TestSQLite_ClaimLeaseUnsetExecutionUsesDefaultAndBuffer(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"echo", "x"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.LeaseUntil == nil {
		t.Fatalf("claim: %+v", got)
	}
	if got.ExecutionTimeoutSeconds != nil {
		t.Fatalf("expected nil execution timeout in model, got %v", *got.ExecutionTimeoutSeconds)
	}
	now := time.Now().Unix()
	runnerEnd := now + int64((10 * time.Minute).Seconds())
	taskEnd := now + int64(api.DefaultExecutionTimeoutSeconds+api.LeaseBufferSeconds)
	wantLease := runnerEnd
	if taskEnd > wantLease {
		wantLease = taskEnd
	}
	gotUnix := got.LeaseUntil.Unix()
	if gotUnix < wantLease-2 || gotUnix > wantLease+5 {
		t.Fatalf("lease_until=%d want ~%d (runnerEnd=%d taskEnd=%d)", gotUnix, wantLease, runnerEnd, taskEnd)
	}
}

func TestSQLite_ExecutionTimeoutZeroInDBIgnoredForLeaseAndModel(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"echo", "x"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET execution_timeout_seconds = 0 WHERE id = ?`, task.ID); err != nil {
		t.Fatal(err)
	}

	got, err := s.ClaimTask(ctx, "runner-1", 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.LeaseUntil == nil {
		t.Fatalf("claim: %+v", got)
	}
	if got.ExecutionTimeoutSeconds != nil {
		t.Fatalf("model should treat 0 as unset, got %v", *got.ExecutionTimeoutSeconds)
	}
	now := time.Now().Unix()
	runnerEnd := now + int64((2 * time.Minute).Seconds())
	taskEnd := now + int64(api.DefaultExecutionTimeoutSeconds+api.LeaseBufferSeconds)
	wantLease := runnerEnd
	if taskEnd > wantLease {
		wantLease = taskEnd
	}
	gotUnix := got.LeaseUntil.Unix()
	if gotUnix < wantLease-2 || gotUnix > wantLease+5 {
		t.Fatalf("lease_until=%d want ~%d (use default+buffer not 0+buffer)", gotUnix, wantLease)
	}
}

func TestSQLite_CancelQueued(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"echo", "x"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, outcome, err := s.RequestCancelTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != CancelOutcomeCanceledQueued {
		t.Fatalf("outcome: %s", outcome)
	}
	if got.Status != models.StatusCanceled {
		t.Fatalf("status: %s", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != exitCanceled {
		t.Fatalf("exit: %v", got.ExitCode)
	}
}

func TestSQLite_CancelClaimedThenCompleteCanceled(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"sleep", "9"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimTask(ctx, "runner-1", 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	got, outcome, err := s.RequestCancelTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != CancelOutcomeCancelRequested {
		t.Fatalf("outcome: %s", outcome)
	}
	if !got.CancelRequested {
		t.Fatal("expected cancel_requested")
	}

	done, err := s.CompleteTask(ctx, task.ID, "runner-1", 130, "stopped\n", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != models.StatusCanceled {
		t.Fatalf("want canceled got %s", done.Status)
	}
	if done.CancelRequested {
		t.Fatal("cancel_requested should be cleared")
	}
}

func TestSQLite_ReapExpiredLeaseWithCancelRequested(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	task := &models.Task{
		ID:            uuid.NewString(),
		Command:       []string{"sleep", "999"},
		WebhookURL:    "https://example.com/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimTask(ctx, "runner-1", time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	if _, outcome, err := s.RequestCancelTask(ctx, task.ID); err != nil || outcome != CancelOutcomeCancelRequested {
		t.Fatalf("cancel claimed: %v %s", err, outcome)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET lease_until = ? WHERE id = ?`,
		time.Now().Unix()-1, task.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	requeued, canceled, err := s.ReapExpiredLeases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if requeued != 0 {
		t.Fatalf("requeued: %d", requeued)
	}
	if len(canceled) != 1 || canceled[0].ID != task.ID {
		t.Fatalf("canceled: %+v", canceled)
	}
	if canceled[0].Status != models.StatusCanceled {
		t.Fatalf("status: %s", canceled[0].Status)
	}

	empty, err := s.ClaimTask(ctx, "runner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if empty != nil {
		t.Fatalf("task should not be requeued, got %+v", empty)
	}
}

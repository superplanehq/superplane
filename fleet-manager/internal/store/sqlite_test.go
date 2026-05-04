package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

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

	done, err := s.CompleteTask(ctx, task.ID, "runner-1", 0, "hello\n", "")
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
	time.Sleep(5 * time.Millisecond)

	n, err := s.ReapExpiredLeases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("reap count: %d", n)
	}

	again, err := s.ClaimTask(ctx, "runner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if again == nil || again.ID != task.ID {
		t.Fatalf("expected task back in queue, got %+v", again)
	}
}

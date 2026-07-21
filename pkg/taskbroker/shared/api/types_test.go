package api

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestValidateExecutionTimeoutSeconds(t *testing.T) {
	if got := ValidateExecutionTimeoutSeconds(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
	one := 1
	if got := ValidateExecutionTimeoutSeconds(&one); got != "" {
		t.Fatalf("1: %q", got)
	}
	max := MaxExecutionTimeoutSecondsRequest
	if got := ValidateExecutionTimeoutSeconds(&max); got != "" {
		t.Fatalf("max: %q", got)
	}
	zero := 0
	if got := ValidateExecutionTimeoutSeconds(&zero); got == "" {
		t.Fatal("expected error for 0")
	}
	tooHigh := MaxExecutionTimeoutSecondsRequest + 1
	if got := ValidateExecutionTimeoutSeconds(&tooHigh); got == "" {
		t.Fatal("expected error for above max")
	}
}

func TestValidateEnvironment(t *testing.T) {
	cases := []struct {
		name string
		env  []EnvironmentVariable
		ok   bool
	}{
		{
			name: "valid",
			env: []EnvironmentVariable{
				{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
				{Name: "_TOKEN_1", Value: "abc=123"},
			},
			ok: true,
		},
		{
			name: "empty value ok",
			env:  []EnvironmentVariable{{Name: "EMPTY", Value: ""}},
			ok:   true,
		},
		{
			name: "invalid empty name",
			env:  []EnvironmentVariable{{Name: "", Value: "x"}},
			ok:   false,
		},
		{
			name: "invalid starts with digit",
			env:  []EnvironmentVariable{{Name: "1BAD", Value: "x"}},
			ok:   false,
		},
		{
			name: "invalid hyphen",
			env:  []EnvironmentVariable{{Name: "BAD-NAME", Value: "x"}},
			ok:   false,
		},
		{
			name: "duplicate",
			env: []EnvironmentVariable{
				{Name: "DUP", Value: "a"},
				{Name: "DUP", Value: "b"},
			},
			ok: false,
		},
		{
			name: "nul value",
			env:  []EnvironmentVariable{{Name: "BAD", Value: "a\x00b"}},
			ok:   false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEnvironment(tt.env)
			if tt.ok && got != "" {
				t.Fatalf("expected ok, got %q", got)
			}
			if !tt.ok && got == "" {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTaskPayloadFromClonesEnvironment(t *testing.T) {
	task := &models.Task{
		ID:          "task-1",
		Environment: []models.EnvironmentVariable{{Name: "A", Value: "1"}},
	}
	payload := TaskPayloadFrom(task)
	if len(payload.Environment) != 1 || payload.Environment[0].Name != "A" || payload.Environment[0].Value != "1" {
		t.Fatalf("environment: %#v", payload.Environment)
	}
	task.Environment[0].Value = "2"
	if payload.Environment[0].Value != "1" {
		t.Fatalf("expected cloned environment, got %#v", payload.Environment)
	}
}

func TestTaskPayloadFromClonesFiles(t *testing.T) {
	task := &models.Task{
		ID:    "task-1",
		Files: []models.TaskFile{{Path: "a.txt", Content: "one"}},
	}
	payload := TaskPayloadFrom(task)
	if len(payload.Files) != 1 || payload.Files[0].Path != "a.txt" || payload.Files[0].Content != "one" {
		t.Fatalf("files: %#v", payload.Files)
	}
	task.Files[0].Content = "two"
	if payload.Files[0].Content != "one" {
		t.Fatalf("expected cloned files, got %#v", payload.Files)
	}
}

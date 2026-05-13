package broker

import (
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestValidateCreateTaskPayload_executionTimeout(t *testing.T) {
	one := 1
	max := api.MaxExecutionTimeoutSecondsRequest
	tooHigh := max + 1
	zero := 0
	neg := -3

	t.Run("nil timeout ok", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:    []string{"echo"},
			WebhookURL: "https://x/h",
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("valid timeout ok", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:                 []string{"echo"},
			WebhookURL:              "https://x/h",
			ExecutionTimeoutSeconds: &one,
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("max boundary ok", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:                 []string{"echo"},
			WebhookURL:              "https://x/h",
			ExecutionTimeoutSeconds: &max,
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("above max", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:                 []string{"echo"},
			WebhookURL:              "https://x/h",
			ExecutionTimeoutSeconds: &tooHigh,
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})
	t.Run("zero rejected", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:                 []string{"echo"},
			WebhookURL:              "https://x/h",
			ExecutionTimeoutSeconds: &zero,
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})
	t.Run("negative rejected", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			Command:                 []string{"echo"},
			WebhookURL:              "https://x/h",
			ExecutionTimeoutSeconds: &neg,
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})
}

func TestValidateCreateTaskPayload_dockerRequiresImage(t *testing.T) {
	thirty := 30
	req := &api.CreateTaskRequest{
		Command:                 []string{"echo"},
		WebhookURL:              "https://x/h",
		ExecutionMode:           "docker",
		ExecutionTimeoutSeconds: &thirty,
	}
	if got := validateCreateTaskPayload(req); got == "" {
		t.Fatal("expected docker_image error")
	}
}

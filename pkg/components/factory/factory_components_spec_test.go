package factory

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestUpdateWorkOrderStatus_ValidatesConfiguration(t *testing.T) {
	c := &UpdateWorkOrderStatus{}
	fields := c.Configuration()

	t.Run("rejects missing work order id", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "ready",
		})
		if err == nil {
			t.Fatal("expected error for missing workOrderId")
		}
	})

	t.Run("rejects unknown status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"status":      "bogus",
		})
		if err == nil {
			t.Fatal("expected error for invalid status option")
		}
	})

	t.Run("requires result when closing", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"status":      "closed",
		})
		if err == nil {
			t.Fatal("expected error for closing without a result")
		}
	})

	t.Run("accepts ready transition", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"status":      "ready",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts closed with failed result", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"status":      "closed",
			"result":      "failed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddWorkOrderComment_ValidatesConfiguration(t *testing.T) {
	c := &AddWorkOrderComment{}
	fields := c.Configuration()

	t.Run("rejects missing body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
		})
		if err == nil {
			t.Fatal("expected error for missing body")
		}
	})

	t.Run("accepts llm default author kind", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"body":        "hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid author kind", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"body":        "hello",
			"authorKind":  "elf",
		})
		if err == nil {
			t.Fatal("expected error for invalid author kind")
		}
	})

	t.Run("rejects `user` author kind (canvas has no acting human)", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId": "wo-1",
			"body":        "hello",
			"authorKind":  "user",
		})
		if err == nil {
			t.Fatal("expected error for `user` author kind on the canvas component")
		}
	})
}

func TestAddWorkOrderArtifact_ValidatesConfiguration(t *testing.T) {
	c := &AddWorkOrderArtifact{}
	fields := c.Configuration()

	t.Run("requires url for pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId":  "wo-1",
			"artifactType": "pr",
		})
		if err == nil {
			t.Fatal("expected error for pr without url")
		}
	})

	t.Run("requires body for markdown", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId":  "wo-1",
			"artifactType": "markdown",
		})
		if err == nil {
			t.Fatal("expected error for markdown without body")
		}
	})

	t.Run("accepts valid pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId":  "wo-1",
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"title":        "Draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts valid markdown", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"workOrderId":  "wo-1",
			"artifactType": "markdown",
			"body":         "notes",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

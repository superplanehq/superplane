package factory

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestUpdateWorkOrderStatus_ValidatesConfiguration(t *testing.T) {
	c := &UpdateWorkOrderStatus{}
	fields := c.Configuration()

	t.Run("rejects unknown status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "bogus",
		})
		if err == nil {
			t.Fatal("expected error for invalid status option")
		}
	})

	t.Run("requires result when closing", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "closed",
		})
		if err == nil {
			t.Fatal("expected error for closing without a result")
		}
	})

	t.Run("accepts draft status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts open status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "open",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts closed with failed result", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "closed",
			"result": "failed",
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
		err := configuration.ValidateConfiguration(fields, map[string]any{})
		if err == nil {
			t.Fatal("expected error for missing body")
		}
	})

	t.Run("accepts body-only configuration", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"body": "hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddWorkOrderArtifact_ValidatesConfiguration(t *testing.T) {
	c := &AddWorkOrderArtifact{}
	fields := c.Configuration()

	t.Run("requires url for pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "pr",
		})
		if err == nil {
			t.Fatal("expected error for pr without url")
		}
	})

	t.Run("accepts valid pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"title":        "Draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts markdown with data-only body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "markdown",
			"data": []any{
				map[string]any{"name": "body", "value": "notes"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts pr with free-form data entries", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"data": []any{
				map[string]any{"name": "number", "value": "482"},
				map[string]any{"name": "provider", "value": "github"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestArtifactDataToMap_FlattensEntries(t *testing.T) {
	entries := []ArtifactDataEntry{
		{Name: "number", Value: "482"},
		{Name: "provider", Value: "github"},
		{Name: "", Value: "ignored"},
	}
	out := artifactDataToMap(entries)
	if got := out["number"]; got != "482" {
		t.Fatalf("expected number=482, got %v", got)
	}
	if got := out["provider"]; got != "github" {
		t.Fatalf("expected provider=github, got %v", got)
	}
	if len(out) != 2 {
		t.Fatalf("expected only two entries (blank names skipped), got %d", len(out))
	}
	if artifactDataToMap(nil) != nil {
		t.Fatal("expected nil map when no entries were provided")
	}
}

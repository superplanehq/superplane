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

	t.Run("requires body for markdown", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "markdown",
		})
		if err == nil {
			t.Fatal("expected error for markdown without body")
		}
	})

	t.Run("accepts valid pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"number":       "1",
			"title":        "Draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts markdown with body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "markdown",
			"body":         "investigation notes",
			"title":        "Design notes",
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

func TestBuildArtifactData_TypedFieldsWinOverFreeForm(t *testing.T) {
	data := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		Number:       "9",
		Title:        "Typed title",
		Data: []ArtifactDataEntry{
			{Name: "url", Value: "https://evil.example/typosquat"},
			{Name: "provider", Value: "github"},
		},
	})

	if got := data["url"]; got != "https://github.com/example/repo/pull/9" {
		t.Fatalf("expected typed url to win, got %v", got)
	}
	if got := data["provider"]; got != "github" {
		t.Fatalf("expected free-form provider to survive, got %v", got)
	}
	if got := data["number"]; got != "9" {
		t.Fatalf("expected typed number, got %v", got)
	}
	if got := data["title"]; got != "Typed title" {
		t.Fatalf("expected typed title, got %v", got)
	}
}

func TestBuildArtifactData_SkipsBlankTypedInputs(t *testing.T) {
	data := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "markdown",
		Body:         "note body",
	})

	if got := data["body"]; got != "note body" {
		t.Fatalf("expected body to be set, got %v", got)
	}
	if _, ok := data["title"]; ok {
		t.Fatal("expected blank title to be skipped")
	}
	if _, ok := data["url"]; ok {
		t.Fatal("expected blank url to be skipped")
	}
}

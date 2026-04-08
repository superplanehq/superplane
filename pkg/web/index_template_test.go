package web

import (
	"strings"
	"testing"
)

func TestRenderIndexTemplate_AgentEnabled(t *testing.T) {
	const raw = `window.SUPERPLANE_AGENT_ENABLED = {{if .AgentEnabled}}true{{else}}false{{end}};`

	t.Run("true only for lowercase yes", func(t *testing.T) {
		t.Setenv("AGENT_ENABLED", "yes")
		t.Setenv("SENTRY_DSN", "")
		t.Setenv("SENTRY_ENVIRONMENT", "")
		out, err := RenderIndexTemplate([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), `SUPERPLANE_AGENT_ENABLED = true`) {
			t.Fatalf("expected injected boolean true, got %q", out)
		}
	})

	t.Run("false when AGENT_ENABLED is YES uppercase", func(t *testing.T) {
		t.Setenv("AGENT_ENABLED", "YES")
		t.Setenv("SENTRY_DSN", "")
		t.Setenv("SENTRY_ENVIRONMENT", "")
		out, err := RenderIndexTemplate([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), `SUPERPLANE_AGENT_ENABLED = false`) {
			t.Fatalf("expected injected boolean false, got %q", out)
		}
	})

	t.Run("false when AGENT_ENABLED is empty", func(t *testing.T) {
		t.Setenv("AGENT_ENABLED", "")
		t.Setenv("SENTRY_DSN", "")
		t.Setenv("SENTRY_ENVIRONMENT", "")
		out, err := RenderIndexTemplate([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), `SUPERPLANE_AGENT_ENABLED = false`) {
			t.Fatalf("expected injected boolean false, got %q", out)
		}
	})

	t.Run("false when AGENT_ENABLED is true", func(t *testing.T) {
		t.Setenv("AGENT_ENABLED", "true")
		t.Setenv("SENTRY_DSN", "")
		t.Setenv("SENTRY_ENVIRONMENT", "")
		out, err := RenderIndexTemplate([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), `SUPERPLANE_AGENT_ENABLED = false`) {
			t.Fatalf("expected injected boolean false, got %q", out)
		}
	})
}

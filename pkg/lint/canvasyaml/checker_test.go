package canvasyaml

import (
	"strings"
	"testing"
)

func TestLintConfigurationFieldNames(t *testing.T) {
	t.Run("accepts camelCase configuration keys", func(t *testing.T) {
		raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: wait-1
      name: Wait
      component: wait
      configuration:
        durationSeconds: 30
        matchList:
          - name: monitorKey
            value: default
  edges: []
`)

		issues, err := LintConfigurationFieldNames(raw)
		if err != nil {
			t.Fatalf("LintConfigurationFieldNames() error = %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %#v", issues)
		}
	})

	t.Run("flags snake_case configuration keys", func(t *testing.T) {
		raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: wait-1
      name: Wait
      component: wait
      configuration:
        duration_seconds: 30
        nested:
          my_field: value
  edges: []
`)

		issues, err := LintConfigurationFieldNames(raw)
		if err != nil {
			t.Fatalf("LintConfigurationFieldNames() error = %v", err)
		}
		if len(issues) != 2 {
			t.Fatalf("expected 2 issues, got %d: %#v", len(issues), issues)
		}

		fields := []string{issues[0].Field, issues[1].Field}
		if !containsAll(fields, "duration_seconds", "my_field") {
			t.Fatalf("unexpected fields: %#v", fields)
		}
	})

	t.Run("ignores non-configuration node fields", func(t *testing.T) {
		raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: wait-1
      name: wait_node
      component: wait
      is_collapsed: true
      configuration:
        durationSeconds: 30
  edges: []
`)

		issues, err := LintConfigurationFieldNames(raw)
		if err != nil {
			t.Fatalf("LintConfigurationFieldNames() error = %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no configuration issues, got %#v", issues)
		}
	})

	t.Run("ignores edges with snake_case", func(t *testing.T) {
		raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: a
      name: A
      component: start
    - id: b
      name: B
      component: wait
      configuration:
        durationSeconds: 1
  edges:
    - source_id: a
      target_id: b
`)

		issues, err := LintConfigurationFieldNames(raw)
		if err != nil {
			t.Fatalf("LintConfigurationFieldNames() error = %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %#v", issues)
		}
	})
}

func TestIsSnakeCaseFieldName(t *testing.T) {
	tests := map[string]bool{
		"my_field":         true,
		"duration_seconds": true,
		"myField":          false,
		"type":             false,
		"URL":              false,
		"SCREAMING_SNAKE":  false,
		"invalid-key":      false,
	}

	for name, want := range tests {
		if got := isSnakeCaseFieldName(name); got != want {
			t.Errorf("isSnakeCaseFieldName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestSuggestCamelCase(t *testing.T) {
	if got := suggestCamelCase("duration_seconds"); got != "durationSeconds" {
		t.Fatalf("suggestCamelCase(duration_seconds) = %q", got)
	}
}

func containsAll(values []string, expected ...string) bool {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}

	for _, want := range expected {
		if _, ok := set[want]; !ok {
			return false
		}
	}

	return true
}

func TestIssueString(t *testing.T) {
	issue := Issue{
		Line:     12,
		Path:     `spec.nodes[0] (id="wait-1").configuration.duration_seconds`,
		Field:    "duration_seconds",
		NodeID:   "wait-1",
		NodeName: "Wait",
	}

	text := issue.String()
	if !strings.Contains(text, "duration_seconds") {
		t.Fatalf("issue string missing field name: %q", text)
	}
	if !strings.Contains(text, "durationSeconds") {
		t.Fatalf("issue string missing suggestion: %q", text)
	}
}

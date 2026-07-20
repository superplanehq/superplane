package yaml

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestConsoleFromYML_ParsesValidConsole(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata:
  name: Ops console
spec:
  panels:
    - id: intro
      type: markdown
      content:
        body: "# Hello"
  layout:
    - i: intro
      x: 0
      y: 0
      w: 12
      h: 6
      minW: 2
      minH: 2
`

	resource, err := ConsoleFromYML([]byte(yaml))
	require.NoError(t, err)
	require.Equal(t, "v1", resource.APIVersion)
	require.Equal(t, KindConsole, resource.Kind)
	require.Len(t, resource.Spec.Panels, 1)
	require.Equal(t, "intro", resource.Spec.Panels[0].ID)
	require.Equal(t, "markdown", resource.Spec.Panels[0].Type)
	require.Equal(t, "# Hello", resource.Spec.Panels[0].Content["body"])
	require.Len(t, resource.Spec.Layout, 1)
	require.Equal(t, 12, resource.Spec.Layout[0].W)
	require.NotNil(t, resource.Spec.Layout[0].MinW)
	assert.Equal(t, 2, *resource.Spec.Layout[0].MinW)
}

func TestConsoleFromYML_ReadsUnquotedLayoutY(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata:
  name: Ops console
spec:
  panels:
    - id: intro
      type: markdown
      content:
        body: "# Hello"
  layout:
    - i: intro
      x: 0
      y: 12
      w: 12
      h: 6
`

	resource, err := ConsoleFromYML([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, resource.Spec.Layout, 1)
	assert.Equal(t, 12, resource.Spec.Layout[0].Y)
}

func TestConsoleFromYML_NormalizesLegacyYAML11LayoutYKey(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Console
metadata:
  name: Ops console
spec:
  panels:
    - id: intro
      type: markdown
      content:
        body: "# Hello"
  layout:
    - i: intro
      x: 0
      y: 12
      w: 12
      h: 6
`)

	jsonBytes, err := ghodssyaml.YAMLToJSON(raw)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &doc))
	normalizeConsoleDocument(doc)

	normalizedJSON, err := json.Marshal(doc)
	require.NoError(t, err)

	var resource Console
	decoder := json.NewDecoder(bytes.NewReader(normalizedJSON))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&resource))

	require.Len(t, resource.Spec.Layout, 1)
	assert.Equal(t, 12, resource.Spec.Layout[0].Y)
}

func TestConsoleFromYML_RejectsLegacyDashboardKind(t *testing.T) {
	yaml := `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kind")
}

func TestConsoleFromYML_RejectsEmptyInput(t *testing.T) {
	_, err := ConsoleFromYML([]byte(""))
	require.Error(t, err)
	_, err = ConsoleFromYML([]byte("   \n\n  "))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsUnknownFields(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata:
  name: ok
spec:
  panels: []
  layout: []
  extraField: nope
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsWrongKind(t *testing.T) {
	yaml := `apiVersion: v1
kind: Canvas
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsWrongAPIVersion(t *testing.T) {
	yaml := `apiVersion: v2
kind: Console
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsNonObjectRoot(t *testing.T) {
	_, err := ConsoleFromYML([]byte("- 1\n- 2\n"))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsUnsupportedPanelType(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p1
      type: timeline
      content: {}
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestConsoleFromYML_RejectsDuplicatePanelIDs(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: dup
      type: markdown
      content: {}
    - id: dup
      type: markdown
      content: {}
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestConsoleFromYML_RejectsLayoutWithUnknownPanel(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p1
      type: markdown
      content: {}
  layout:
    - i: other
      x: 0
      y: 0
      w: 1
      h: 1
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestConsoleFromYML_RejectsNonStringBody(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p1
      type: markdown
      content:
        body: 42
  layout: []
`
	_, err := ConsoleFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body")
}

func TestValidateMarkdownVariables_AcceptsValidShapes(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"body": "hello {{ recipe.title }}",
			"variables": []any{
				map[string]any{
					"name": "recipe",
					"source": map[string]any{
						"kind":      "memory",
						"namespace": "recipes",
						"orderBy":   "createdAt",
						"direction": "desc",
						"matches": []any{
							map[string]any{"field": "status", "value": "approved"},
						},
					},
				},
				map[string]any{
					"name":   "lastRun",
					"source": map[string]any{"kind": "run", "select": "latest"},
				},
				map[string]any{
					"name":   "lastFailure",
					"source": map[string]any{"kind": "run", "select": "latest_failed"},
				},
			},
		},
	}
	assert.NoError(t, validateMarkdownContent(panel))
}

func TestValidateMarkdownVariables_RejectsBadName(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"variables": []any{
				map[string]any{
					"name":   "1invalid",
					"source": map[string]any{"kind": "run", "select": "latest"},
				},
			},
		},
	}
	err := validateMarkdownContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid identifier")
}

func TestValidateMarkdownVariables_RejectsDuplicateName(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"variables": []any{
				map[string]any{"name": "dup", "source": map[string]any{"kind": "run", "select": "latest"}},
				map[string]any{"name": "dup", "source": map[string]any{"kind": "run", "select": "latest_passed"}},
			},
		},
	}
	err := validateMarkdownContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")
}

func TestValidateMarkdownVariables_RejectsUnknownKind(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"variables": []any{
				map[string]any{"name": "bad", "source": map[string]any{"kind": "executions"}},
			},
		},
	}
	err := validateMarkdownContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory")
}

func TestValidateMarkdownVariables_RejectsUnknownRunSelect(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"variables": []any{
				map[string]any{"name": "bad", "source": map[string]any{"kind": "run", "select": "first"}},
			},
		},
	}
	err := validateMarkdownContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "latest")
}

func TestValidateMarkdownVariables_RejectsEmptyNamespace(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"variables": []any{
				map[string]any{"name": "bad", "source": map[string]any{"kind": "memory", "namespace": ""}},
			},
		},
	}
	err := validateMarkdownContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")
}

func TestValidateMarkdownVariables_MemoryListMode(t *testing.T) {
	// `mode: list` is the new opt-in that resolves the variable to every
	// matching memory row so authors can use CEL list macros. We exercise
	// the validator across the documented happy/sad paths so YAML diffs and
	// the FE editor surface the same errors.
	build := func(source map[string]any) ConsolePanel {
		return ConsolePanel{
			ID:   "p1",
			Type: ConsolePanelTypeMarkdown,
			Content: map[string]any{
				"variables": []any{
					map[string]any{"name": "rows", "source": source},
				},
			},
		}
	}

	t.Run("accepts mode: list with no limit", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list",
		}))
		require.NoError(t, err)
	})

	t.Run("accepts mode: list with an integer limit", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": 25,
		}))
		require.NoError(t, err)
	})

	t.Run("accepts mode: list with a whole-number float64 limit (YAML decoder shape)", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": float64(10),
		}))
		require.NoError(t, err)
	})

	t.Run("rejects an unknown mode", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "many",
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mode")
	})

	t.Run("rejects a non-numeric limit", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": "5",
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")
	})

	t.Run("rejects a fractional limit", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": 1.5,
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")
	})

	t.Run("rejects a zero / negative limit", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": 0,
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")

		err = validateMarkdownContent(build(map[string]any{
			"kind": "memory", "namespace": "n", "mode": "list", "limit": -3,
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")
	})
}

// TestValidateMarkdownVariables_RunFilters exercises the shared status /
// trigger filter shape on `kind: run` variables. Every field is optional
// so the accepted vocabulary must line up with the FE constants in
// `runPresentation.ts` — mismatches would let the FE persist YAML that
// the backend then rejects on import.
func TestValidateMarkdownVariables_RunFilters(t *testing.T) {
	build := func(source map[string]any) ConsolePanel {
		return ConsolePanel{
			ID:   "p1",
			Type: ConsolePanelTypeMarkdown,
			Content: map[string]any{
				"variables": []any{
					map[string]any{"name": "run", "source": source},
				},
			},
		}
	}

	t.Run("accepts empty filter arrays (empty means all)", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "run", "select": "latest", "statuses": []any{}, "triggers": []any{},
		}))
		require.NoError(t, err)
	})

	t.Run("accepts every allowed status value", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind":     "run",
			"select":   "latest",
			"statuses": []any{"running", "passed", "failed", "cancelled"},
			"triggers": []any{"deploy", "release"},
		}))
		require.NoError(t, err)
	})

	t.Run("rejects an unknown status value", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "run", "select": "latest", "statuses": []any{"running", "flaky"},
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "statuses[1]")
	})

	t.Run("rejects a non-string trigger entry", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "run", "select": "latest", "triggers": []any{"deploy", 42},
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "triggers[1]")
	})

	t.Run("rejects an empty trigger entry", func(t *testing.T) {
		err := validateMarkdownContent(build(map[string]any{
			"kind": "run", "select": "latest", "triggers": []any{"deploy", ""},
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "triggers[1]")
	})
}

// TestValidateDataSource_RunsFilters exercises the runs datasource
// status / trigger filter fields on widget panels (table / chart /
// number / scorecard). Mirrors the markdown-variable filter tests since
// both surfaces share `validateRunStatusesField` / `validateRunTriggersField`.
func TestValidateDataSource_RunsFilters(t *testing.T) {
	table := func(ds map[string]any) ConsolePanel {
		return ConsolePanel{
			ID:   "t1",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": ds,
				"render":     map[string]any{"kind": "table", "columns": []any{map[string]any{"field": "status"}}},
			},
		}
	}

	t.Run("accepts populated filter arrays", func(t *testing.T) {
		err := validateTablePanelContent(table(map[string]any{
			"kind":     "runs",
			"limit":    50,
			"statuses": []any{"failed", "cancelled"},
			"triggers": []any{"deploy"},
		}))
		require.NoError(t, err)
	})

	t.Run("accepts nil filter fields (empty means all)", func(t *testing.T) {
		err := validateTablePanelContent(table(map[string]any{"kind": "runs"}))
		require.NoError(t, err)
	})

	t.Run("rejects an unknown status", func(t *testing.T) {
		err := validateTablePanelContent(table(map[string]any{
			"kind": "runs", "statuses": []any{"flaky"},
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dataSource.statuses[0]")
	})

	t.Run("rejects a non-string trigger", func(t *testing.T) {
		err := validateTablePanelContent(table(map[string]any{
			"kind": "runs", "triggers": []any{7},
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dataSource.triggers[0]")
	})
}

func TestValidateHTMLContent_AcceptsWellFormed(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeHTML,
		Content: map[string]any{
			"title": "Status",
			"body":  `<div class="p-2"><strong>{{ rec.status }}</strong></div>`,
			"variables": []any{
				map[string]any{
					"name":   "rec",
					"source": map[string]any{"kind": "memory", "namespace": "deploys"},
				},
			},
		},
	}
	assert.NoError(t, validateHTMLContent(panel))
}

func TestValidateHTMLContent_RejectsNonStringBody(t *testing.T) {
	panel := ConsolePanel{
		ID:      "p1",
		Type:    ConsolePanelTypeHTML,
		Content: map[string]any{"body": 42},
	}
	err := validateHTMLContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.body must be a string")
}

func TestValidateHTMLContent_RejectsNonStringTitle(t *testing.T) {
	panel := ConsolePanel{
		ID:      "p1",
		Type:    ConsolePanelTypeHTML,
		Content: map[string]any{"title": 42},
	}
	err := validateHTMLContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.title must be a string")
}

func TestValidateHTMLContent_PropagatesVariableValidation(t *testing.T) {
	panel := ConsolePanel{
		ID:   "p1",
		Type: ConsolePanelTypeHTML,
		Content: map[string]any{
			"variables": []any{
				map[string]any{
					"name":   "1bad",
					"source": map[string]any{"kind": "memory", "namespace": "n"},
				},
			},
		},
	}
	err := validateHTMLContent(panel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid identifier")
}

func TestConsoleFromYML_AcceptsHTMLPanel(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: html-1
      type: html
      content:
        title: Status
        body: '<div class="p-2"><strong>{{ rec.status }}</strong></div>'
        variables:
          - name: rec
            source:
              kind: memory
              namespace: deploys
  layout:
    - i: html-1
      x: 0
      y: 0
      w: 6
      h: 3
`

	resource, err := ConsoleFromYML([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, resource.Spec.Panels, 1)
	assert.Equal(t, ConsolePanelTypeHTML, resource.Spec.Panels[0].Type)
}

func TestDashboardFromYML_AcceptsMarkdownVariables(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: panel-1
      type: markdown
      content:
        body: |
          Today: {{ recipe.title }} from {{ lastRun.status }}.
        variables:
          - name: recipe
            source:
              kind: memory
              namespace: recipes
              orderBy: createdAt
              direction: desc
              matches:
                - field: status
                  value: approved
          - name: lastRun
            source:
              kind: run
              select: latest
  layout:
    - i: panel-1
      x: 0
      y: 0
      w: 6
      h: 4
`
	resource, err := ConsoleFromYML([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, resource.Spec.Panels, 1)
	content := resource.Spec.Panels[0].Content
	variables, ok := content["variables"].([]any)
	require.True(t, ok)
	require.Len(t, variables, 2)
}

func TestDashboardFromYML_RejectsTooManyPanels(t *testing.T) {
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  panels:\n")
	for i := 0; i < MaxConsolePanels+1; i++ {
		b.WriteString("    - id: p")
		b.WriteString(strings.Repeat("a", 1))
		b.WriteString(strings.Repeat("b", i+1))
		b.WriteString("\n      type: markdown\n      content: {}\n")
	}
	b.WriteString("  layout: []\n")

	_, err := ConsoleFromYML([]byte(b.String()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many panels")
}

func TestCanvasVersionToConsoleYML_RoundTripsEmptyDashboard(t *testing.T) {
	canvasID := uuid.New()
	canvasVersion := &models.CanvasVersion{
		WorkflowID:    canvasID,
		ConsolePanels: datatypes.NewJSONType([]models.ConsolePanel{}),
		ConsoleLayout: datatypes.NewJSONType([]models.ConsoleLayoutItem{}),
	}

	out, err := VersionToConsoleYML("Canvas Name", canvasVersion)
	require.NoError(t, err)
	assert.Contains(t, string(out), "apiVersion: v1")
	assert.Contains(t, string(out), "kind: Console")
	assert.NotContains(t, string(out), canvasID.String())
	assert.NotContains(t, string(out), "name: Canvas Name")

	parsed, err := ConsoleFromYML([]byte(out))
	require.NoError(t, err)
	require.Equal(t, KindConsole, parsed.Kind)
	assert.Empty(t, parsed.Spec.Panels)
	assert.Empty(t, parsed.Spec.Layout)
}

func TestCanvasVersionToConsoleYML_RoundTripsPanelsAndLayout(t *testing.T) {
	canvasID := uuid.New()
	minW, minH := 2, 1
	canvasVersion := &models.CanvasVersion{
		WorkflowID: canvasID,
		ConsolePanels: datatypes.NewJSONType([]models.ConsolePanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hello"}},
		}),
		ConsoleLayout: datatypes.NewJSONType([]models.ConsoleLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2, MinW: &minW, MinH: &minH},
		}),
	}

	out, err := VersionToConsoleYML("Canvas Name", canvasVersion)
	require.NoError(t, err)

	parsed, err := ConsoleFromYML([]byte(out))
	require.NoError(t, err)
	require.Len(t, parsed.Spec.Panels, 1)
	require.Equal(t, "p1", parsed.Spec.Panels[0].ID)
	require.Equal(t, "markdown", parsed.Spec.Panels[0].Type)
	require.Equal(t, "hello", parsed.Spec.Panels[0].Content["body"])
	require.Len(t, parsed.Spec.Layout, 1)
	require.Equal(t, 4, parsed.Spec.Layout[0].W)
	require.NotNil(t, parsed.Spec.Layout[0].MinW)
	assert.Equal(t, 2, *parsed.Spec.Layout[0].MinW)
}

func TestCanvasVersionToConsoleYML_IsDeterministic(t *testing.T) {
	canvasID := uuid.New()
	canvasVersion := &models.CanvasVersion{
		WorkflowID: canvasID,
		ConsolePanels: datatypes.NewJSONType([]models.ConsolePanel{
			{ID: "a", Type: "markdown", Content: map[string]any{"body": "hi"}},
			{ID: "b", Type: "markdown", Content: map[string]any{"body": "hey"}},
		}),
		ConsoleLayout: datatypes.NewJSONType([]models.ConsoleLayoutItem{
			{I: "a", X: 0, Y: 0, W: 1, H: 1},
			{I: "b", X: 1, Y: 0, W: 1, H: 1},
		}),
	}

	first, err := VersionToConsoleYML("Canvas Name", canvasVersion)
	require.NoError(t, err)
	second, err := VersionToConsoleYML("Canvas Name", canvasVersion)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

func TestValidateConsoleContent_RejectsInvalidLayout(t *testing.T) {
	panels := []ConsolePanel{{ID: "p", Type: "markdown", Content: map[string]any{}}}
	err := ValidateConsoleContent(panels, []ConsoleLayoutItem{
		{I: "p", X: -1, Y: 0, W: 1, H: 1},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-negative")

	err = ValidateConsoleContent(panels, []ConsoleLayoutItem{
		{I: "p", X: 0, Y: 0, W: 0, H: 1},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestValidateDashboardContent_AcceptsDraftTablePanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "table",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": ""},
				"render":     map[string]any{"kind": "table", "columns": []any{}},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsInvalidTypedPanelConfig(t *testing.T) {
	tests := []struct {
		name     string
		panel    ConsolePanel
		contains string
	}{
		{
			name: "memory source without namespace",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory"},
					"render":     map[string]any{"kind": "table", "columns": []any{}},
				},
			},
			contains: "dataSource.namespace must be a string",
		},
		{
			name: "table column without field",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":    "table",
						"columns": []any{map[string]any{"label": "Missing field"}},
					},
				},
			},
			contains: "render.columns[0].field",
		},
		{
			name: "table filter with unsupported op",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":    "table",
						"columns": []any{},
						"where":   []any{map[string]any{"field": "status", "op": "regex"}},
					},
				},
			},
			contains: "render.where[0].op",
		},
		{
			name: "trigger row action without node",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":       "table",
						"columns":    []any{},
						"rowActions": []any{map[string]any{"kind": "trigger", "label": "Run"}},
					},
				},
			},
			contains: "render.rowActions[0].node",
		},
		{
			name: "progress column without target",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind": "table",
						"columns": []any{
							map[string]any{"field": "current", "format": "progress"},
						},
					},
				},
			},
			contains: "render.columns[0].progressTarget must be a non-empty string for progress columns",
		},
		{
			name: "progress column with unknown label",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind": "table",
						"columns": []any{
							map[string]any{
								"field":          "current",
								"format":         "progress",
								"progressTarget": "total",
								"progressLabel":  "fraction",
							},
						},
					},
				},
			},
			contains: "render.columns[0].progressLabel must be one of",
		},
		{
			name: "table column with invalid trendBetter",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind": "table",
						"columns": []any{
							map[string]any{
								"field":        "durationMs",
								"format":       "trend",
								"trendBetter":  "sideways",
								"trendDisplay": "percent",
							},
						},
					},
				},
			},
			contains: "render.columns[0].trendBetter must be one of",
		},
		{
			name: "table column with invalid trendDisplay",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind": "table",
						"columns": []any{
							map[string]any{
								"field":        "durationMs",
								"format":       "trend",
								"trendBetter":  "down",
								"trendDisplay": "chart",
							},
						},
					},
				},
			},
			contains: "render.columns[0].trendDisplay must be one of",
		},
		{
			name: "table column with invalid showTrend",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind": "table",
						"columns": []any{
							map[string]any{
								"field":     "durationMs",
								"format":    "duration",
								"showTrend": "yes",
							},
						},
					},
				},
			},
			contains: "render.columns[0].showTrend must be a boolean",
		},
		{
			name: "row style with unknown tone",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":    "table",
						"columns": []any{},
						"rowStyles": []any{
							map[string]any{"field": "status", "op": "eq", "value": "error", "tone": "magenta"},
						},
					},
				},
			},
			contains: "render.rowStyles[0].tone must be one of",
		},
		{
			name: "row style with unsupported op",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":    "table",
						"columns": []any{},
						"rowStyles": []any{
							map[string]any{"field": "status", "op": "regex", "value": "err.*", "tone": "red"},
						},
					},
				},
			},
			contains: "render.rowStyles[0].op is not supported",
		},
		{
			name: "row style with empty field",
			panel: ConsolePanel{
				ID:   "table",
				Type: ConsolePanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
					"render": map[string]any{
						"kind":    "table",
						"columns": []any{},
						"rowStyles": []any{
							map[string]any{"field": "", "op": "eq", "value": "error", "tone": "red"},
						},
					},
				},
			},
			contains: "render.rowStyles[0].field must be a non-empty string",
		},
		{
			name: "chart with unsupported type",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":   "chart",
						"type":   "pie",
						"xField": "status",
						"series": []any{map[string]any{"label": "Count"}},
					},
				},
			},
			contains: "render.type",
		},
		{
			name: "data source limit with wrong type",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions", "limit": "many"},
					"render": map[string]any{
						"kind":   "chart",
						"type":   "bar",
						"xField": "status",
						"series": []any{map[string]any{"label": "Count"}},
					},
				},
			},
			contains: "dataSource.limit must be a number",
		},
		{
			name: "number render prefix must be string",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "runs"},
					"render":     map[string]any{"kind": "number", "aggregation": "count", "prefix": 42},
				},
			},
			contains: "render.prefix must be a string",
		},
		{
			name: "composite number panel rejects render.aggregation",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{
						"kind":    "memory",
						"combine": "sum",
						"sources": []any{
							map[string]any{"namespace": "a", "aggregation": "sum", "field": "cost"},
						},
					},
					"render": map[string]any{"kind": "number", "aggregation": "sum", "field": "cost"},
				},
			},
			contains: "render.aggregation must not be set",
		},
		{
			name: "composite number panel rejects render.field",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{
						"kind":    "memory",
						"combine": "sum",
						"sources": []any{
							map[string]any{"namespace": "a", "aggregation": "sum", "field": "cost"},
						},
					},
					"render": map[string]any{"kind": "number", "field": "cost"},
				},
			},
			contains: "render.field must not be set",
		},
		{
			name: "composite number panel rejects unknown combine",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{
						"kind":    "memory",
						"combine": "median",
						"sources": []any{
							map[string]any{"namespace": "a", "aggregation": "sum", "field": "cost"},
						},
					},
					"render": map[string]any{"kind": "number"},
				},
			},
			contains: "dataSource.combine must be one of",
		},
		{
			name: "composite number panel requires field for non-count source",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{
						"kind":    "memory",
						"combine": "sum",
						"sources": []any{
							map[string]any{"namespace": "a", "aggregation": "sum"},
						},
					},
					"render": map[string]any{"kind": "number"},
				},
			},
			contains: "dataSource.sources[0].field is required",
		},
		{
			name: "composite number panel rejects empty sources",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{
						"kind":    "memory",
						"combine": "sum",
						"sources": []any{},
					},
					"render": map[string]any{"kind": "number"},
				},
			},
			contains: "dataSource.sources must be a non-empty array",
		},
		{
			name: "multi-number panel rejects empty metrics",
			panel: ConsolePanel{
				ID:      "n",
				Type:    ConsolePanelTypeNumber,
				Content: map[string]any{"metrics": []any{}},
			},
			contains: "metrics must be a non-empty array",
		},
		{
			name: "multi-number metric rejects unknown aggregation",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"metrics": []any{
						map[string]any{
							"dataSource": map[string]any{"kind": "runs"},
							"render":     map[string]any{"kind": "number", "aggregation": "median"},
						},
					},
				},
			},
			contains: "metrics[0].render.aggregation must be one of",
		},
		{
			name: "multi-number metric requires field for non-count aggregation",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"metrics": []any{
						map[string]any{
							"dataSource": map[string]any{"kind": "memory", "namespace": "costs"},
							"render":     map[string]any{"kind": "number", "aggregation": "sum"},
						},
					},
				},
			},
			contains: "metrics[0].render.field is required",
		},
		{
			name: "multi-number metric rejects composite data source",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"metrics": []any{
						map[string]any{
							"dataSource": map[string]any{
								"kind":    "memory",
								"combine": "sum",
								"sources": []any{map[string]any{"namespace": "a", "aggregation": "count"}},
							},
							"render": map[string]any{"kind": "number", "aggregation": "count"},
						},
					},
				},
			},
			contains: "metrics[0].dataSource must be a single-source",
		},
		{
			name: "multi-number metric rejects non-number render kind",
			panel: ConsolePanel{
				ID:   "n",
				Type: ConsolePanelTypeNumber,
				Content: map[string]any{
					"metrics": []any{
						map[string]any{
							"dataSource": map[string]any{"kind": "runs"},
							"render":     map[string]any{"kind": "table"},
						},
					},
				},
			},
			contains: `render.kind must be "number"`,
		},
		{
			name: "chart series prefix must be a string",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":   "chart",
						"type":   "bar",
						"xField": "service",
						"series": []any{map[string]any{"field": "cost", "prefix": 42}},
					},
				},
			},
			contains: "render.series[0].prefix must be a string",
		},
		{
			name: "chart legend mode must be auto/show/hide",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":   "chart",
						"type":   "bar",
						"xField": "service",
						"series": []any{map[string]any{"field": "cost"}},
						"legend": "bogus",
					},
				},
			},
			contains: "render.legend must be one of auto/show/hide",
		},
		{
			name: "chart seriesField must be a string",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory", "namespace": "costs"},
					"render": map[string]any{
						"kind":        "chart",
						"type":        "stacked-bar",
						"xField":      "date",
						"seriesField": 42,
						"series":      []any{map[string]any{"field": "cost_usd"}},
					},
				},
			},
			contains: "render.seriesField must be a string",
		},
		{
			name: "chart xFormat must be a string",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":    "chart",
						"type":    "bar",
						"xField":  "createdAt",
						"xFormat": 7,
						"series":  []any{map[string]any{"field": "cost"}},
					},
				},
			},
			contains: "render.xFormat must be a string",
		},
		{
			name: "chart yLabel must be a string",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":   "chart",
						"type":   "bar",
						"xField": "service",
						"yLabel": false,
						"series": []any{map[string]any{"field": "cost"}},
					},
				},
			},
			contains: "render.yLabel must be a string",
		},
		{
			name: "chart yFormat must be a string",
			panel: ConsolePanel{
				ID:   "chart",
				Type: ConsolePanelTypeChart,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "executions"},
					"render": map[string]any{
						"kind":    "chart",
						"type":    "bar",
						"xField":  "service",
						"yFormat": 1.5,
						"series":  []any{map[string]any{"field": "cost"}},
					},
				},
			},
			contains: "render.yFormat must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConsoleContent([]ConsolePanel{tt.panel}, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestValidateConsoleContent_AcceptsChartSeriesFormatAndLegend(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "chart",
			Type: ConsolePanelTypeChart,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "executions"},
				"render": map[string]any{
					"kind":   "chart",
					"type":   "bar",
					"xField": "service",
					"series": []any{
						map[string]any{"field": "cost", "label": "Cost", "format": "number", "prefix": "$", "suffix": " /mo"},
					},
					"legend": "show",
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsChartAxisFormatting(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "chart",
			Type: ConsolePanelTypeChart,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "executions"},
				"render": map[string]any{
					"kind":    "chart",
					"type":    "bar",
					"xField":  "createdAt",
					"xFormat": "date",
					"yLabel":  "USD",
					"yFormat": "number",
					"series":  []any{map[string]any{"field": "cost", "label": "Cost", "prefix": "$"}},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsTableTrendColumns(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "table",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
				"render": map[string]any{
					"kind": "table",
					"columns": []any{
						map[string]any{
							"field":        "durationMs",
							"format":       "trend",
							"trendBetter":  "down",
							"trendDisplay": "percent",
						},
						map[string]any{
							"field":        "score",
							"format":       "trend",
							"trendBetter":  "up",
							"trendDisplay": "value",
						},
						map[string]any{
							"field":        "coverage",
							"format":       "trend",
							"trendDisplay": "none",
						},
						map[string]any{
							"field":        "durationMs",
							"format":       "duration",
							"showTrend":    true,
							"trendBetter":  "down",
							"trendDisplay": "percent",
						},
						map[string]any{
							"field":     "passRate",
							"format":    "percent",
							"showTrend": true,
						},
					},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsTableRowStyles(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "table",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
				"render": map[string]any{
					"kind":    "table",
					"columns": []any{map[string]any{"field": "status"}},
					"rowStyles": []any{
						map[string]any{"field": "status", "op": "eq", "value": "error", "tone": "red-soft"},
						map[string]any{"field": "status", "op": "eq", "value": "deploying", "tone": "orange-soft"},
						map[string]any{"field": "deployedAt", "op": "not_exists", "tone": "dimmed"},
					},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsProgressColumn(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "table",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "env"},
				"render": map[string]any{
					"kind": "table",
					"columns": []any{
						map[string]any{
							"field":          "completed",
							"format":         "progress",
							"progressTarget": "total",
							"progressLabel":  "number",
						},
						map[string]any{
							"field":          "score",
							"format":         "progress",
							"progressTarget": "100",
						},
					},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsWidgetSort(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "runs",
			Type: ConsolePanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "executions"},
				"render": map[string]any{
					"kind":    "table",
					"columns": []any{map[string]any{"field": "status"}},
					"sort":    map[string]any{"field": "createdAt", "order": "desc"},
				},
			},
		},
		{
			ID:   "perf",
			Type: ConsolePanelTypeChart,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "executions"},
				"render": map[string]any{
					"kind":   "chart",
					"type":   "bar",
					"xField": "service",
					"series": []any{map[string]any{"field": "cost"}},
					"sort":   map[string]any{"field": `{{ formatDate(createdAt, "yyyy-MM-dd") }}`},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsSortWithoutField(t *testing.T) {
	panel := ConsolePanel{
		ID:   "runs",
		Type: ConsolePanelTypeTable,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "executions"},
			"render": map[string]any{
				"kind":    "table",
				"columns": []any{map[string]any{"field": "status"}},
				"sort":    map[string]any{"order": "asc"},
			},
		},
	}

	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.field must be a non-empty string`)
}

func TestValidateConsoleContent_RejectsSortWithBlankField(t *testing.T) {
	panel := ConsolePanel{
		ID:   "runs",
		Type: ConsolePanelTypeTable,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "executions"},
			"render": map[string]any{
				"kind":    "table",
				"columns": []any{map[string]any{"field": "status"}},
				"sort":    map[string]any{"field": "   "},
			},
		},
	}

	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.field must be a non-empty string`)
}

func TestValidateConsoleContent_RejectsUnknownSortOrder(t *testing.T) {
	panel := ConsolePanel{
		ID:   "perf",
		Type: ConsolePanelTypeChart,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "executions"},
			"render": map[string]any{
				"kind":   "chart",
				"type":   "bar",
				"xField": "service",
				"series": []any{map[string]any{"field": "cost"}},
				"sort":   map[string]any{"field": "createdAt", "order": "random"},
			},
		},
	}

	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.order must be one of asc/desc`)
}

func TestValidateConsoleContent_AcceptsNodesPanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "key-nodes",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"title": "Key Nodes",
				"nodes": []any{
					map[string]any{
						"node":        "deploy-prod",
						"description": "Promotes the latest build",
						"showRun":     true,
					},
					map[string]any{
						"node":  "rollback",
						"label": "Rollback",
					},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsDraftNodesPanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:      "key-nodes",
			Type:    ConsolePanelTypeNodes,
			Content: map[string]any{"nodes": []any{}},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsNodesPanelMissingNodeRef(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "key-nodes",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"description": "missing"},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].node must be a non-empty string")
}

func TestValidateConsoleContent_RejectsNodesPanelWithNonArrayNodes(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:      "key-nodes",
			Type:    ConsolePanelTypeNodes,
			Content: map[string]any{"nodes": map[string]any{"oops": true}},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes must be an array")
}

func TestValidateConsoleContent_RejectsNodesPanelWithBadShowRun(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "key-nodes",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"node": "deploy-prod", "showRun": "yes"},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].showRun must be a boolean")
}

func TestValidateConsoleContent_RejectsNodesPanelWithBadPromptConfirmation(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "key-nodes",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"node": "deploy-prod", "promptConfirmation": "yes"},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].promptConfirmation must be a boolean")
}

func TestValidateConsoleContent_AcceptsNodesPanelWithFormMode(t *testing.T) {
	for _, mode := range []string{ConsoleNodesPanelFormModeModal, ConsoleNodesPanelFormModeInline} {
		t.Run(mode, func(t *testing.T) {
			panels := []ConsolePanel{
				{
					ID:   "prompt",
					Type: ConsolePanelTypeNodes,
					Content: map[string]any{
						"nodes": []any{
							map[string]any{"node": "start", "showRun": true, "formMode": mode},
						},
					},
				},
			}

			err := ValidateConsoleContent(panels, nil)
			require.NoError(t, err)
		})
	}
}

func TestValidateConsoleContent_RejectsNodesPanelWithUnknownFormMode(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "prompt",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"node": "start", "showRun": true, "formMode": "drawer"},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `content.nodes[0].formMode must be "modal" or "inline"`)
}

func TestValidateConsoleContent_RejectsNodesPanelWithNonStringFormMode(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "prompt",
			Type: ConsolePanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"node": "start", "showRun": true, "formMode": true},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].formMode must be a string")
}

func TestValidateConsoleContent_RejectsNodePanelWithBadPromptConfirmation(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "deploy",
			Type: ConsolePanelTypeNode,
			Content: map[string]any{
				"node":               "deploy-prod",
				"promptConfirmation": "yes",
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.promptConfirmation must be a boolean")
}

func TestValidateConsoleContent_AcceptsNodePanelWithLabel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "deploy",
			Type: ConsolePanelTypeNode,
			Content: map[string]any{
				"node":  "deploy-prod",
				"label": "Ship to prod",
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsNodePanelWithBadLabel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "deploy",
			Type: ConsolePanelTypeNode,
			Content: map[string]any{
				"node":  "deploy-prod",
				"label": 42,
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.label must be a string")
}

func TestValidateConsoleContent_AcceptsCompositeNumberPanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "score",
			Type: ConsolePanelTypeNumber,
			Content: map[string]any{
				"dataSource": map[string]any{
					"kind":    "memory",
					"combine": "sum",
					"sources": []any{
						map[string]any{"namespace": "a", "aggregation": "sum", "field": "cost"},
						map[string]any{"namespace": "b", "aggregation": "count"},
					},
				},
				"render": map[string]any{"kind": "number", "prefix": "R$"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsMultiNumberPanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "kpis",
			Type: ConsolePanelTypeNumber,
			Content: map[string]any{
				"title": "Pipeline KPIs",
				"metrics": []any{
					map[string]any{
						"dataSource": map[string]any{"kind": "runs"},
						"render":     map[string]any{"kind": "number", "aggregation": "count", "label": "Total runs"},
					},
					map[string]any{
						"dataSource": map[string]any{"kind": "memory", "namespace": "costs"},
						"render": map[string]any{
							"kind":        "number",
							"aggregation": "sum",
							"field":       "cost",
							"label":       "Total cost",
							"format":      "number",
							"prefix":      "R$",
						},
					},
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsScorecardPanel(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"title":      "Open UX papercuts",
				"dataSource": map[string]any{"kind": "memory", "namespace": "ux_papercuts"},
				"render": map[string]any{
					"kind":           "scorecard",
					"aggregation":    "last",
					"field":          "openCount",
					"format":         "number",
					"label":          "Open UX papercuts",
					"better":         "down",
					"target":         "80",
					"showProgress":   true,
					"sparklineField": "openCount",
					"showChange":     "both",
					"changeCaption":  "vs start of range",
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_AcceptsCountScorecardWithoutField(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "runs",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "runs", "limit": 100},
				"render": map[string]any{
					"kind":        "scorecard",
					"aggregation": "count",
				},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsScorecardWithoutContent(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestValidateConsoleContent_RejectsScorecardWithWrongRenderKind(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "number", "aggregation": "count"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.kind must be "scorecard"`)
}

func TestValidateConsoleContent_RejectsScorecardWithUnknownAggregation(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "median"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.aggregation must be one of")
}

func TestValidateConsoleContent_RejectsScorecardWithoutFieldForNonCount(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "last"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.field is required")
}

func TestValidateConsoleContent_RejectsScorecardWithInvalidBetter(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "count", "better": "sideways"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.better must be one of")
}

func TestValidateConsoleContent_RejectsScorecardWithInvalidShowChange(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "count", "showChange": "chart"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.showChange must be one of")
}

func TestValidateConsoleContent_RejectsScorecardWithNonBooleanShowProgress(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "count", "showProgress": "yes"},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.showProgress must be a boolean")
}

func TestValidateConsoleContent_RejectsScorecardWithNonStringTarget(t *testing.T) {
	panels := []ConsolePanel{
		{
			ID:   "papercuts",
			Type: ConsolePanelTypeScorecard,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": "x"},
				"render":     map[string]any{"kind": "scorecard", "aggregation": "count", "target": 42},
			},
		},
	}

	err := ValidateConsoleContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.target must be a string")
}

// boardPanel returns a ConsolePanel of type "board" with the given render
// map merged over a minimal valid base. Tests customize just the fields
// they exercise so the intent stays legible.
func boardPanel(render map[string]any) ConsolePanel {
	base := map[string]any{
		"kind":    "board",
		"groupBy": "status",
		"lanes":   []any{map[string]any{"value": "Todo"}, map[string]any{"value": "Done", "color": "green"}},
		"card":    map[string]any{"titleField": "title"},
	}
	for k, v := range render {
		base[k] = v
	}
	return ConsolePanel{
		ID:   "factory",
		Type: ConsolePanelTypeBoard,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "memory", "namespace": "factory_tasks"},
			"render":     base,
		},
	}
}

func TestValidateConsoleContent_AcceptsBoardPanel(t *testing.T) {
	panel := boardPanel(map[string]any{
		"otherLane": true,
		"card": map[string]any{
			"titleField": "title",
			"fields": []any{
				map[string]any{"field": "pr_url", "format": "link", "label": "PR"},
				map[string]any{"field": "updatedAt", "format": "relative"},
			},
		},
		"where":      []any{map[string]any{"field": "archived", "op": "not_exists"}},
		"sort":       map[string]any{"field": "updatedAt", "order": "desc"},
		"rowActions": []any{map[string]any{"kind": "trigger", "node": "start", "hook": "run"}},
	})

	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.NoError(t, err)
}

func TestValidateConsoleContent_RejectsBoardWithoutGroupBy(t *testing.T) {
	panel := boardPanel(map[string]any{"groupBy": ""})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.groupBy must be a non-empty string")
}

func TestValidateConsoleContent_RejectsBoardWithEmptyLanes(t *testing.T) {
	panel := boardPanel(map[string]any{"lanes": []any{}})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.lanes must be a non-empty array")
}

func TestValidateConsoleContent_RejectsBoardWithBlankLaneValue(t *testing.T) {
	panel := boardPanel(map[string]any{"lanes": []any{map[string]any{"value": " "}}})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.lanes[0].value must be a non-empty string")
}

func TestValidateConsoleContent_RejectsBoardWithUnknownLaneColor(t *testing.T) {
	panel := boardPanel(map[string]any{"lanes": []any{map[string]any{"value": "Todo", "color": "fuchsia"}}})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.lanes[0].color must be one of")
}

func TestValidateConsoleContent_RejectsBoardWithoutCard(t *testing.T) {
	panel := boardPanel(map[string]any{"card": nil})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.card must be an object")
}

func TestValidateConsoleContent_RejectsBoardWithBlankTitleField(t *testing.T) {
	panel := boardPanel(map[string]any{"card": map[string]any{"titleField": ""}})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.card.titleField must be a non-empty string")
}

func TestValidateConsoleContent_RejectsBoardWithBadCardField(t *testing.T) {
	panel := boardPanel(map[string]any{
		"card": map[string]any{
			"titleField": "title",
			"fields":     []any{map[string]any{"field": ""}},
		},
	})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.card.fields[0].field must be a non-empty string")
}

func TestValidateConsoleContent_RejectsBoardWithNonBooleanOtherLane(t *testing.T) {
	panel := boardPanel(map[string]any{"otherLane": "yes"})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.otherLane must be a boolean")
}

func TestValidateConsoleContent_RejectsBoardWithBadRowAction(t *testing.T) {
	panel := boardPanel(map[string]any{
		"rowActions": []any{map[string]any{"kind": "trigger", "hook": "run"}},
	})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render.rowActions[0].node must be set to a trigger node")
}

func TestValidateConsoleContent_RejectsBoardWithWrongRenderKind(t *testing.T) {
	panel := boardPanel(map[string]any{"kind": "table"})
	err := ValidateConsoleContent([]ConsolePanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.kind must be "board"`)
}

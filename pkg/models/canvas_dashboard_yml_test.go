package models

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestDashboardFromYML_ParsesValidConsole(t *testing.T) {
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

	resource, err := DashboardFromYML([]byte(yaml))
	require.NoError(t, err)
	require.Equal(t, "v1", resource.APIVersion)
	require.Equal(t, ConsoleKind, resource.Kind)
	require.Equal(t, "Ops console", resource.Metadata.Name)
	require.Len(t, resource.Spec.Panels, 1)
	require.Equal(t, "intro", resource.Spec.Panels[0].ID)
	require.Equal(t, "markdown", resource.Spec.Panels[0].Type)
	require.Equal(t, "# Hello", resource.Spec.Panels[0].Content["body"])
	require.Len(t, resource.Spec.Layout, 1)
	require.Equal(t, 12, resource.Spec.Layout[0].W)
	require.NotNil(t, resource.Spec.Layout[0].MinW)
	assert.Equal(t, 2, *resource.Spec.Layout[0].MinW)
}

func TestDashboardFromYML_RejectsLegacyDashboardKind(t *testing.T) {
	yaml := `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kind")
}

func TestDashboardFromYML_RejectsEmptyInput(t *testing.T) {
	_, err := DashboardFromYML([]byte(""))
	require.Error(t, err)
	_, err = DashboardFromYML([]byte("   \n\n  "))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsUnknownFields(t *testing.T) {
	yaml := `apiVersion: v1
kind: Console
metadata:
  name: ok
spec:
  panels: []
  layout: []
  extraField: nope
`
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsWrongKind(t *testing.T) {
	yaml := `apiVersion: v1
kind: Canvas
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsWrongAPIVersion(t *testing.T) {
	yaml := `apiVersion: v2
kind: Console
metadata: {}
spec:
  panels: []
  layout: []
`
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsNonObjectRoot(t *testing.T) {
	_, err := DashboardFromYML([]byte("- 1\n- 2\n"))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsUnsupportedPanelType(t *testing.T) {
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
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestDashboardFromYML_RejectsDuplicatePanelIDs(t *testing.T) {
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
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestDashboardFromYML_RejectsLayoutWithUnknownPanel(t *testing.T) {
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
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsNonStringBody(t *testing.T) {
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
	_, err := DashboardFromYML([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body")
}

func TestDashboardFromYML_RejectsTooManyPanels(t *testing.T) {
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  panels:\n")
	for i := 0; i < MaxDashboardPanels+1; i++ {
		b.WriteString("    - id: p")
		b.WriteString(strings.Repeat("a", 1))
		b.WriteString(strings.Repeat("b", i+1))
		b.WriteString("\n      type: markdown\n      content: {}\n")
	}
	b.WriteString("  layout: []\n")

	_, err := DashboardFromYML([]byte(b.String()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many panels")
}

func TestDashboardToYML_RoundTripsEmptyDashboard(t *testing.T) {
	canvasID := uuid.New()
	dashboard := &CanvasDashboard{
		CanvasID: canvasID,
		Panels:   datatypes.NewJSONType([]DashboardPanel{}),
		Layout:   datatypes.NewJSONType([]DashboardLayoutItem{}),
	}

	out, err := DashboardToYML(dashboard, "Canvas Name")
	require.NoError(t, err)
	assert.Contains(t, string(out), "apiVersion: v1")
	assert.Contains(t, string(out), "kind: Console")
	assert.Contains(t, string(out), canvasID.String())
	assert.Contains(t, string(out), "name: Canvas Name")

	parsed, err := DashboardFromYML(out)
	require.NoError(t, err)
	require.Equal(t, ConsoleKind, parsed.Kind)
	assert.Empty(t, parsed.Spec.Panels)
	assert.Empty(t, parsed.Spec.Layout)
}

func TestDashboardToYML_RoundTripsPanelsAndLayout(t *testing.T) {
	canvasID := uuid.New()
	minW, minH := 2, 1
	dashboard := &CanvasDashboard{
		CanvasID: canvasID,
		Panels: datatypes.NewJSONType([]DashboardPanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hello"}},
		}),
		Layout: datatypes.NewJSONType([]DashboardLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2, MinW: &minW, MinH: &minH},
		}),
	}

	out, err := DashboardToYML(dashboard, "")
	require.NoError(t, err)

	parsed, err := DashboardFromYML(out)
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

func TestDashboardToYML_IsDeterministic(t *testing.T) {
	canvasID := uuid.New()
	dashboard := &CanvasDashboard{
		CanvasID: canvasID,
		Panels: datatypes.NewJSONType([]DashboardPanel{
			{ID: "a", Type: "markdown", Content: map[string]any{"body": "hi"}},
			{ID: "b", Type: "markdown", Content: map[string]any{"body": "hey"}},
		}),
		Layout: datatypes.NewJSONType([]DashboardLayoutItem{
			{I: "a", X: 0, Y: 0, W: 1, H: 1},
			{I: "b", X: 1, Y: 0, W: 1, H: 1},
		}),
	}

	first, err := DashboardToYML(dashboard, "name")
	require.NoError(t, err)
	second, err := DashboardToYML(dashboard, "name")
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

func TestValidateDashboardContent_RejectsInvalidLayout(t *testing.T) {
	panels := []DashboardPanel{{ID: "p", Type: "markdown", Content: map[string]any{}}}
	err := ValidateDashboardContent(panels, []DashboardLayoutItem{
		{I: "p", X: -1, Y: 0, W: 1, H: 1},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-negative")

	err = ValidateDashboardContent(panels, []DashboardLayoutItem{
		{I: "p", X: 0, Y: 0, W: 0, H: 1},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestValidateDashboardContent_AcceptsDraftTablePanel(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "table",
			Type: DashboardPanelTypeTable,
			Content: map[string]any{
				"dataSource": map[string]any{"kind": "memory", "namespace": ""},
				"render":     map[string]any{"kind": "table", "columns": []any{}},
			},
		},
	}

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_RejectsInvalidTypedPanelConfig(t *testing.T) {
	tests := []struct {
		name     string
		panel    DashboardPanel
		contains string
	}{
		{
			name: "memory source without namespace",
			panel: DashboardPanel{
				ID:   "table",
				Type: DashboardPanelTypeTable,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "memory"},
					"render":     map[string]any{"kind": "table", "columns": []any{}},
				},
			},
			contains: "dataSource.namespace must be a string",
		},
		{
			name: "table column without field",
			panel: DashboardPanel{
				ID:   "table",
				Type: DashboardPanelTypeTable,
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
			panel: DashboardPanel{
				ID:   "table",
				Type: DashboardPanelTypeTable,
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
			panel: DashboardPanel{
				ID:   "table",
				Type: DashboardPanelTypeTable,
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
			name: "chart with unsupported type",
			panel: DashboardPanel{
				ID:   "chart",
				Type: DashboardPanelTypeChart,
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
			panel: DashboardPanel{
				ID:   "chart",
				Type: DashboardPanelTypeChart,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
				Content: map[string]any{
					"dataSource": map[string]any{"kind": "runs"},
					"render":     map[string]any{"kind": "number", "aggregation": "count", "prefix": 42},
				},
			},
			contains: "render.prefix must be a string",
		},
		{
			name: "composite number panel rejects render.aggregation",
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:      "n",
				Type:    DashboardPanelTypeNumber,
				Content: map[string]any{"metrics": []any{}},
			},
			contains: "metrics must be a non-empty array",
		},
		{
			name: "multi-number metric rejects unknown aggregation",
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "n",
				Type: DashboardPanelTypeNumber,
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
			panel: DashboardPanel{
				ID:   "chart",
				Type: DashboardPanelTypeChart,
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
			panel: DashboardPanel{
				ID:   "chart",
				Type: DashboardPanelTypeChart,
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
			panel: DashboardPanel{
				ID:   "chart",
				Type: DashboardPanelTypeChart,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDashboardContent([]DashboardPanel{tt.panel}, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestValidateDashboardContent_AcceptsChartSeriesFormatAndLegend(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "chart",
			Type: DashboardPanelTypeChart,
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

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_AcceptsWidgetSort(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "runs",
			Type: DashboardPanelTypeTable,
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
			Type: DashboardPanelTypeChart,
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

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_RejectsSortWithoutField(t *testing.T) {
	panel := DashboardPanel{
		ID:   "runs",
		Type: DashboardPanelTypeTable,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "executions"},
			"render": map[string]any{
				"kind":    "table",
				"columns": []any{map[string]any{"field": "status"}},
				"sort":    map[string]any{"order": "asc"},
			},
		},
	}

	err := ValidateDashboardContent([]DashboardPanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.field must be a non-empty string`)
}

func TestValidateDashboardContent_RejectsSortWithBlankField(t *testing.T) {
	panel := DashboardPanel{
		ID:   "runs",
		Type: DashboardPanelTypeTable,
		Content: map[string]any{
			"dataSource": map[string]any{"kind": "executions"},
			"render": map[string]any{
				"kind":    "table",
				"columns": []any{map[string]any{"field": "status"}},
				"sort":    map[string]any{"field": "   "},
			},
		},
	}

	err := ValidateDashboardContent([]DashboardPanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.field must be a non-empty string`)
}

func TestValidateDashboardContent_RejectsUnknownSortOrder(t *testing.T) {
	panel := DashboardPanel{
		ID:   "perf",
		Type: DashboardPanelTypeChart,
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

	err := ValidateDashboardContent([]DashboardPanel{panel}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `render.sort.order must be one of asc/desc`)
}

func TestValidateDashboardContent_AcceptsNodesPanel(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "key-nodes",
			Type: DashboardPanelTypeNodes,
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

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_AcceptsDraftNodesPanel(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:      "key-nodes",
			Type:    DashboardPanelTypeNodes,
			Content: map[string]any{"nodes": []any{}},
		},
	}

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_RejectsNodesPanelMissingNodeRef(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "key-nodes",
			Type: DashboardPanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"description": "missing"},
				},
			},
		},
	}

	err := ValidateDashboardContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].node must be a non-empty string")
}

func TestValidateDashboardContent_RejectsNodesPanelWithNonArrayNodes(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:      "key-nodes",
			Type:    DashboardPanelTypeNodes,
			Content: map[string]any{"nodes": map[string]any{"oops": true}},
		},
	}

	err := ValidateDashboardContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes must be an array")
}

func TestValidateDashboardContent_RejectsNodesPanelWithBadShowRun(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "key-nodes",
			Type: DashboardPanelTypeNodes,
			Content: map[string]any{
				"nodes": []any{
					map[string]any{"node": "deploy-prod", "showRun": "yes"},
				},
			},
		},
	}

	err := ValidateDashboardContent(panels, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.nodes[0].showRun must be a boolean")
}

func TestValidateDashboardContent_AcceptsCompositeNumberPanel(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "score",
			Type: DashboardPanelTypeNumber,
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

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

func TestValidateDashboardContent_AcceptsMultiNumberPanel(t *testing.T) {
	panels := []DashboardPanel{
		{
			ID:   "kpis",
			Type: DashboardPanelTypeNumber,
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

	err := ValidateDashboardContent(panels, nil)
	require.NoError(t, err)
}

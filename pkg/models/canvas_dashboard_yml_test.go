package models

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestDashboardFromYML_ParsesValidDashboard(t *testing.T) {
	yaml := `apiVersion: v1
kind: Dashboard
metadata:
  name: My dashboard
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
	require.Equal(t, "Dashboard", resource.Kind)
	require.Equal(t, "My dashboard", resource.Metadata.Name)
	require.Len(t, resource.Spec.Panels, 1)
	require.Equal(t, "intro", resource.Spec.Panels[0].ID)
	require.Equal(t, "markdown", resource.Spec.Panels[0].Type)
	require.Equal(t, "# Hello", resource.Spec.Panels[0].Content["body"])
	require.Len(t, resource.Spec.Layout, 1)
	require.Equal(t, 12, resource.Spec.Layout[0].W)
	require.NotNil(t, resource.Spec.Layout[0].MinW)
	assert.Equal(t, 2, *resource.Spec.Layout[0].MinW)
}

func TestDashboardFromYML_RejectsEmptyInput(t *testing.T) {
	_, err := DashboardFromYML([]byte(""))
	require.Error(t, err)
	_, err = DashboardFromYML([]byte("   \n\n  "))
	require.Error(t, err)
}

func TestDashboardFromYML_RejectsUnknownFields(t *testing.T) {
	yaml := `apiVersion: v1
kind: Dashboard
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
kind: Dashboard
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
kind: Dashboard
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
kind: Dashboard
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
kind: Dashboard
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
kind: Dashboard
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
	b.WriteString("apiVersion: v1\nkind: Dashboard\nmetadata: {}\nspec:\n  panels:\n")
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
	assert.Contains(t, string(out), "kind: Dashboard")
	assert.Contains(t, string(out), canvasID.String())
	assert.Contains(t, string(out), "name: Canvas Name")

	parsed, err := DashboardFromYML(out)
	require.NoError(t, err)
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

package installation

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestPersistInstalledConsoleNoopWhenNil(t *testing.T) {
	support.Setup(t)

	// A nil console must not require a valid canvas id — it short-circuits
	// before any DB write.
	err := persistInstalledConsole("not-a-uuid", nil)
	require.NoError(t, err)
}

func TestPersistInstalledConsoleRejectsInvalidCanvasID(t *testing.T) {
	support.Setup(t)

	console := &models.DashboardYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.DashboardYAMLSpec{
			Panels: []models.DashboardPanel{
				{ID: "p1", Type: models.DashboardPanelTypeMarkdown, Content: map[string]any{"body": "hi"}},
			},
		},
	}

	err := persistInstalledConsole("not-a-uuid", console)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid canvas id")
}

func TestPersistInstalledConsoleWritesPanelsAndLayout(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	minW := 2
	console := &models.DashboardYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.DashboardYAMLSpec{
			Panels: []models.DashboardPanel{
				{
					ID:      "notes",
					Type:    models.DashboardPanelTypeMarkdown,
					Content: map[string]any{"title": "Notes", "body": "hello from console.yaml"},
				},
			},
			Layout: []models.DashboardLayoutItem{
				{I: "notes", X: 0, Y: 0, W: 4, H: 2, MinW: &minW},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	stored, err := models.FindCanvasDashboard(canvas.ID)
	require.NoError(t, err)
	panels := stored.Panels.Data()
	layout := stored.Layout.Data()

	require.Len(t, panels, 1)
	assert.Equal(t, "notes", panels[0].ID)
	assert.Equal(t, models.DashboardPanelTypeMarkdown, panels[0].Type)
	assert.Equal(t, "hello from console.yaml", panels[0].Content["body"])

	require.Len(t, layout, 1)
	assert.Equal(t, "notes", layout[0].I)
	assert.Equal(t, 4, layout[0].W)
	require.NotNil(t, layout[0].MinW)
	assert.Equal(t, 2, *layout[0].MinW)
}

func TestPersistInstalledConsoleIsReplaceAll(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	// Seed the canvas with a different console first.
	_, err := models.UpsertCanvasDashboard(
		canvas.ID,
		[]models.DashboardPanel{
			{ID: "old", Type: models.DashboardPanelTypeMarkdown, Content: map[string]any{"body": "old"}},
		},
		[]models.DashboardLayoutItem{
			{I: "old", X: 0, Y: 0, W: 4, H: 2},
		},
	)
	require.NoError(t, err)

	console := &models.DashboardYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.DashboardYAMLSpec{
			Panels: []models.DashboardPanel{
				{ID: "new", Type: models.DashboardPanelTypeMarkdown, Content: map[string]any{"body": "new"}},
			},
			Layout: []models.DashboardLayoutItem{
				{I: "new", X: 0, Y: 0, W: 6, H: 3},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	stored, err := models.FindCanvasDashboard(canvas.ID)
	require.NoError(t, err)
	panels := stored.Panels.Data()
	require.Len(t, panels, 1)
	assert.Equal(t, "new", panels[0].ID)

	// Sanity-check that calling persistInstalledConsole(nil) on the same
	// canvas does not clobber an already-stored console.
	require.NoError(t, persistInstalledConsole(canvas.ID.String(), nil))

	stored, err = models.FindCanvasDashboard(canvas.ID)
	require.NoError(t, err)
	require.Len(t, stored.Panels.Data(), 1)
	assert.Equal(t, "new", stored.Panels.Data()[0].ID)
}

func TestPersistInstalledConsoleUsesUUIDAsPrimaryKey(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	console := &models.DashboardYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.DashboardYAMLSpec{
			Panels: []models.DashboardPanel{
				{ID: "p", Type: models.DashboardPanelTypeMarkdown, Content: map[string]any{}},
			},
			Layout: []models.DashboardLayoutItem{
				{I: "p", X: 0, Y: 0, W: 4, H: 2},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	// A different (random) canvas id should not see this console.
	unrelated, err := models.FindCanvasDashboard(uuid.New())
	require.NoError(t, err)
	assert.Empty(t, unrelated.Panels.Data())
	assert.Empty(t, unrelated.Layout.Data())
}

package installation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
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

	console := &models.ConsoleYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.ConsoleYAMLSpec{
			Panels: []models.ConsolePanel{
				{ID: "p1", Type: models.ConsolePanelTypeMarkdown, Content: map[string]any{"body": "hi"}},
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
	console := &models.ConsoleYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.ConsoleYAMLSpec{
			Panels: []models.ConsolePanel{
				{
					ID:      "notes",
					Type:    models.ConsolePanelTypeMarkdown,
					Content: map[string]any{"title": "Notes", "body": "hello from console.yaml"},
				},
			},
			Layout: []models.ConsoleLayoutItem{
				{I: "notes", X: 0, Y: 0, W: 4, H: 2, MinW: &minW},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	canvasVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	panels := canvasVersion.ConsolePanels.Data()
	layout := canvasVersion.ConsoleLayout.Data()

	require.Len(t, panels, 1)
	assert.Equal(t, "notes", panels[0].ID)
	assert.Equal(t, models.ConsolePanelTypeMarkdown, panels[0].Type)
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
	_, err := models.UpsertCanvasVersionConsole(
		canvas.ID,
		[]models.ConsolePanel{
			{ID: "old", Type: models.ConsolePanelTypeMarkdown, Content: map[string]any{"body": "old"}},
		},
		[]models.ConsoleLayoutItem{
			{I: "old", X: 0, Y: 0, W: 4, H: 2},
		},
	)
	require.NoError(t, err)

	console := &models.ConsoleYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.ConsoleYAMLSpec{
			Panels: []models.ConsolePanel{
				{ID: "new", Type: models.ConsolePanelTypeMarkdown, Content: map[string]any{"body": "new"}},
			},
			Layout: []models.ConsoleLayoutItem{
				{I: "new", X: 0, Y: 0, W: 6, H: 3},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	canvasVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	panels := canvasVersion.ConsolePanels.Data()
	require.Len(t, panels, 1)
	assert.Equal(t, "new", panels[0].ID)

	// Sanity-check that calling persistInstalledConsole(nil) on the same
	// canvas does not clobber an already-stored console.
	require.NoError(t, persistInstalledConsole(canvas.ID.String(), nil))

	canvasVersion, err = models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	require.Len(t, canvasVersion.ConsolePanels.Data(), 1)
	assert.Equal(t, "new", canvasVersion.ConsolePanels.Data()[0].ID)
}

func TestPersistInstalledConsoleUsesUUIDAsPrimaryKey(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	console := &models.ConsoleYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Spec: models.ConsoleYAMLSpec{
			Panels: []models.ConsolePanel{
				{ID: "p", Type: models.ConsolePanelTypeMarkdown, Content: map[string]any{}},
			},
			Layout: []models.ConsoleLayoutItem{
				{I: "p", X: 0, Y: 0, W: 4, H: 2},
			},
		},
	}

	require.NoError(t, persistInstalledConsole(canvas.ID.String(), console))

	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	unrelated, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), otherCanvas.ID)
	require.NoError(t, err)
	assert.Empty(t, unrelated.ConsolePanels.Data())
	assert.Empty(t, unrelated.ConsoleLayout.Data())
}

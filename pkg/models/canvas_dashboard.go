package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DashboardPanel struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content map[string]any `json:"content"`
}

type DashboardLayoutItem struct {
	I    string `json:"i"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	MinW *int   `json:"minW,omitempty"`
	MinH *int   `json:"minH,omitempty"`
}

// CanvasDashboard is the API-facing view of a version-scoped console dashboard.
type CanvasDashboard struct {
	CanvasID  uuid.UUID
	VersionID string
	Panels    datatypes.JSONType[[]DashboardPanel]
	Layout    datatypes.JSONType[[]DashboardLayoutItem]
	UpdatedAt time.Time
}

func emptyConsolePanels() datatypes.JSONType[[]DashboardPanel] {
	return datatypes.NewJSONType([]DashboardPanel{})
}

func emptyConsoleLayout() datatypes.JSONType[[]DashboardLayoutItem] {
	return datatypes.NewJSONType([]DashboardLayoutItem{})
}

func consolePanelsData(panels datatypes.JSONType[[]DashboardPanel]) []DashboardPanel {
	if data := panels.Data(); data != nil {
		return data
	}
	return []DashboardPanel{}
}

func consoleLayoutData(layout datatypes.JSONType[[]DashboardLayoutItem]) []DashboardLayoutItem {
	if data := layout.Data(); data != nil {
		return data
	}
	return []DashboardLayoutItem{}
}

func CanvasDashboardFromVersion(version *CanvasVersion) *CanvasDashboard {
	if version == nil {
		return nil
	}

	updatedAt := time.Time{}
	if version.UpdatedAt != nil {
		updatedAt = *version.UpdatedAt
	}

	return &CanvasDashboard{
		CanvasID:  version.WorkflowID,
		VersionID: version.ID,
		Panels:    version.ConsolePanels,
		Layout:    version.ConsoleLayout,
		UpdatedAt: updatedAt,
	}
}

func FindCanvasDashboardForVersionInTransaction(tx *gorm.DB, workflowID uuid.UUID, versionID string) (*CanvasDashboard, error) {
	version, err := FindCanvasVersionInTransaction(tx, workflowID, versionID)
	if err != nil {
		return nil, err
	}

	return CanvasDashboardFromVersion(version), nil
}

func FindCanvasDashboardForVersion(workflowID uuid.UUID, versionID string) (*CanvasDashboard, error) {
	return FindCanvasDashboardForVersionInTransaction(database.Conn(), workflowID, versionID)
}

func FindLiveCanvasDashboardInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*CanvasDashboard, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	return CanvasDashboardFromVersion(version), nil
}

func FindLiveCanvasDashboard(workflowID uuid.UUID) (*CanvasDashboard, error) {
	return FindLiveCanvasDashboardInTransaction(database.Conn(), workflowID)
}

// FindCanvasDashboard returns the live version dashboard for backward compatibility.
func FindCanvasDashboard(canvasID uuid.UUID) (*CanvasDashboard, error) {
	dashboard, err := FindLiveCanvasDashboard(canvasID)
	if err != nil {
		return nil, err
	}
	if dashboard == nil {
		return &CanvasDashboard{
			CanvasID: canvasID,
			Panels:   emptyConsolePanels(),
			Layout:   emptyConsoleLayout(),
		}, nil
	}
	return dashboard, nil
}

func copyVersionConsoleFields(source *CanvasVersion, target *CanvasVersion) {
	if source == nil || target == nil {
		return
	}

	target.ConsolePanels = datatypes.NewJSONType(consolePanelsData(source.ConsolePanels))
	target.ConsoleLayout = datatypes.NewJSONType(consoleLayoutData(source.ConsoleLayout))
}

func UpdateCanvasVersionDashboardInTransaction(
	tx *gorm.DB,
	version *CanvasVersion,
	panels []DashboardPanel,
	layout []DashboardLayoutItem,
) (*CanvasDashboard, error) {
	if panels == nil {
		panels = []DashboardPanel{}
	}
	if layout == nil {
		layout = []DashboardLayoutItem{}
	}

	now := time.Now()
	version.ConsolePanels = datatypes.NewJSONType(panels)
	version.ConsoleLayout = datatypes.NewJSONType(layout)
	version.UpdatedAt = &now

	if err := tx.Save(version).Error; err != nil {
		return nil, err
	}

	return CanvasDashboardFromVersion(version), nil
}

func UpsertCanvasDashboardInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	panels []DashboardPanel,
	layout []DashboardLayoutItem,
) (*CanvasDashboard, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return nil, err
	}

	return UpdateCanvasVersionDashboardInTransaction(tx, version, panels, layout)
}

func UpsertCanvasDashboard(canvasID uuid.UUID, panels []DashboardPanel, layout []DashboardLayoutItem) (*CanvasDashboard, error) {
	return UpsertCanvasDashboardInTransaction(database.Conn(), canvasID, panels, layout)
}

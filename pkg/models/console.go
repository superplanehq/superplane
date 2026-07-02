package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ConsolePanel struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content map[string]any `json:"content"`
}

type ConsoleLayoutItem struct {
	I    string `json:"i"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	MinW *int   `json:"minW,omitempty"`
	MinH *int   `json:"minH,omitempty"`
}

func consolePanelsData(panels datatypes.JSONType[[]ConsolePanel]) []ConsolePanel {
	if data := panels.Data(); data != nil {
		return data
	}
	return []ConsolePanel{}
}

func consoleLayoutData(layout datatypes.JSONType[[]ConsoleLayoutItem]) []ConsoleLayoutItem {
	if data := layout.Data(); data != nil {
		return data
	}
	return []ConsoleLayoutItem{}
}

func copyVersionConsoleFields(source *CanvasVersion, target *CanvasVersion) {
	if source == nil || target == nil {
		return
	}

	target.ConsolePanels = datatypes.NewJSONType(consolePanelsData(source.ConsolePanels))
	target.ConsoleLayout = datatypes.NewJSONType(consoleLayoutData(source.ConsoleLayout))
}

func UpdateCanvasVersionConsoleInTransaction(
	tx *gorm.DB,
	version *CanvasVersion,
	panels []ConsolePanel,
	layout []ConsoleLayoutItem,
) (*CanvasVersion, error) {
	if panels == nil {
		panels = []ConsolePanel{}
	}
	if layout == nil {
		layout = []ConsoleLayoutItem{}
	}

	now := time.Now()
	version.ConsolePanels = datatypes.NewJSONType(panels)
	version.ConsoleLayout = datatypes.NewJSONType(layout)
	version.UpdatedAt = &now

	if err := tx.Save(version).Error; err != nil {
		return nil, err
	}

	return version, nil
}

func UpsertCanvasVersionConsoleInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	panels []ConsolePanel,
	layout []ConsoleLayoutItem,
) (*CanvasVersion, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return nil, err
	}

	return UpdateCanvasVersionConsoleInTransaction(tx, version, panels, layout)
}

func UpsertCanvasVersionConsole(canvasID uuid.UUID, panels []ConsolePanel, layout []ConsoleLayoutItem) (*CanvasVersion, error) {
	return UpsertCanvasVersionConsoleInTransaction(database.Conn(), canvasID, panels, layout)
}

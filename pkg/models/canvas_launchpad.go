package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LaunchpadPanel is a single user-authored panel rendered on the canvas Launchpad.
// Content is intentionally polymorphic so future panel types (charts, run lists,
// iframes, etc.) only need to extend the type registry without a schema change.
type LaunchpadPanel struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content map[string]any `json:"content"`
}

// LaunchpadLayoutItem mirrors the shape that react-grid-layout produces in its
// onLayoutChange callback. The `i` field references LaunchpadPanel.ID.
type LaunchpadLayoutItem struct {
	I          string `json:"i"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	W          int    `json:"w"`
	H          int    `json:"h"`
	MinW       *int   `json:"minW,omitempty"`
	MinH       *int   `json:"minH,omitempty"`
	AutoHeight *bool  `json:"autoHeight,omitempty"`
}

type CanvasLaunchpad struct {
	CanvasID  uuid.UUID `gorm:"type:uuid;primary_key"`
	Panels    datatypes.JSONType[[]LaunchpadPanel]
	Layout    datatypes.JSONType[[]LaunchpadLayoutItem]
	UpdatedAt time.Time
}

func (CanvasLaunchpad) TableName() string {
	return "canvas_launchpads"
}

// FindCanvasLaunchpadInTransaction returns the launchpad row for the given
// canvas, or a zero-valued struct (with empty panels/layout) if none exists.
// Callers can rely on the response always being non-nil so the UI sees an
// empty grid for canvases that never had a launchpad row.
func FindCanvasLaunchpadInTransaction(tx *gorm.DB, canvasID uuid.UUID) (*CanvasLaunchpad, error) {
	var record CanvasLaunchpad
	err := tx.Where("canvas_id = ?", canvasID).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return &CanvasLaunchpad{
			CanvasID: canvasID,
			Panels:   datatypes.NewJSONType([]LaunchpadPanel{}),
			Layout:   datatypes.NewJSONType([]LaunchpadLayoutItem{}),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func FindCanvasLaunchpad(canvasID uuid.UUID) (*CanvasLaunchpad, error) {
	return FindCanvasLaunchpadInTransaction(database.Conn(), canvasID)
}

// UpsertCanvasLaunchpadInTransaction creates or replaces the launchpad row
// atomically, replacing both panels and layout in a single statement.
func UpsertCanvasLaunchpadInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	panels []LaunchpadPanel,
	layout []LaunchpadLayoutItem,
) (*CanvasLaunchpad, error) {
	if panels == nil {
		panels = []LaunchpadPanel{}
	}
	if layout == nil {
		layout = []LaunchpadLayoutItem{}
	}

	record := CanvasLaunchpad{
		CanvasID:  canvasID,
		Panels:    datatypes.NewJSONType(panels),
		Layout:    datatypes.NewJSONType(layout),
		UpdatedAt: time.Now(),
	}

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "canvas_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"panels",
			"layout",
			"updated_at",
		}),
	}).Create(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func UpsertCanvasLaunchpad(canvasID uuid.UUID, panels []LaunchpadPanel, layout []LaunchpadLayoutItem) (*CanvasLaunchpad, error) {
	return UpsertCanvasLaunchpadInTransaction(database.Conn(), canvasID, panels, layout)
}

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type CanvasDashboard struct {
	CanvasID  uuid.UUID `gorm:"type:uuid;primary_key"`
	Panels    datatypes.JSONType[[]DashboardPanel]
	Layout    datatypes.JSONType[[]DashboardLayoutItem]
	UpdatedAt time.Time
}

func (CanvasDashboard) TableName() string {
	return "canvas_dashboards"
}

func FindCanvasDashboardInTransaction(tx *gorm.DB, canvasID uuid.UUID) (*CanvasDashboard, error) {
	var record CanvasDashboard
	err := tx.Where("canvas_id = ?", canvasID).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return &CanvasDashboard{
			CanvasID: canvasID,
			Panels:   datatypes.NewJSONType([]DashboardPanel{}),
			Layout:   datatypes.NewJSONType([]DashboardLayoutItem{}),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func FindCanvasDashboard(canvasID uuid.UUID) (*CanvasDashboard, error) {
	return FindCanvasDashboardInTransaction(database.Conn(), canvasID)
}

func UpsertCanvasDashboardInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	panels []DashboardPanel,
	layout []DashboardLayoutItem,
) (*CanvasDashboard, error) {
	if panels == nil {
		panels = []DashboardPanel{}
	}
	if layout == nil {
		layout = []DashboardLayoutItem{}
	}

	record := CanvasDashboard{
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

func UpsertCanvasDashboard(canvasID uuid.UUID, panels []DashboardPanel, layout []DashboardLayoutItem) (*CanvasDashboard, error) {
	return UpsertCanvasDashboardInTransaction(database.Conn(), canvasID, panels, layout)
}

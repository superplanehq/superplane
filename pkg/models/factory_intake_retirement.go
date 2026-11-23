package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RetiredProductiveIntakeSource identifies stored intakes only. It is not an allowed source.
const RetiredProductiveIntakeSource = "productive-tasks"

var ErrFactoryIntakeRetired = errors.New("Productive intake is retired")

// FactoryIntakeRetirement retains managed canvas identity after its intake row is removed.
// Inventory records the project bindings and webhook IDs needed to verify remote cleanup.
type FactoryIntakeRetirement struct {
	CanvasID       uuid.UUID `gorm:"primaryKey"`
	IntakeID       uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Inventory      datatypes.JSONType[[]map[string]any]
	RetiredAt      time.Time
}

func (FactoryIntakeRetirement) TableName() string { return "factory_intake_retirements" }

// ListProductiveIntakesForRetirement includes paused and soft-deleted canvases.
func ListProductiveIntakesForRetirement(tx *gorm.DB, after uuid.UUID, limit int) ([]FactoryIntake, error) {
	var intakes []FactoryIntake
	err := tx.Where("source = ? AND id > ?", RetiredProductiveIntakeSource, after).
		Order("id ASC").Limit(limit).Find(&intakes).Error
	return intakes, err
}

func IsRetiredFactoryIntakeCanvas(tx *gorm.DB, canvasID uuid.UUID) (bool, error) {
	var count int64
	if err := tx.Model(&FactoryIntake{}).Where("canvas_id = ? AND source = ?", canvasID, RetiredProductiveIntakeSource).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	err := tx.Model(&FactoryIntakeRetirement{}).Where("canvas_id = ?", canvasID).Count(&count).Error
	return count > 0, err
}

func ValidateFactoryIntakePublication(tx *gorm.DB, canvasID uuid.UUID, nodes []Node) error {
	if len(nodes) == 0 {
		return nil
	}
	retired, err := IsRetiredFactoryIntakeCanvas(tx, canvasID)
	if err != nil {
		return err
	}
	if retired {
		return ErrFactoryIntakeRetired
	}
	return nil
}

func (i *FactoryIntake) RecordRetirement(tx *gorm.DB, nodes []CanvasNode) error {
	inventory := make([]map[string]any, 0)
	for _, node := range nodes {
		if node.Type != NodeTypeTrigger {
			continue
		}
		entry := map[string]any{
			"nodeId": node.NodeID, "integrationId": node.AppInstallationID,
			"webhookId": node.WebhookID, "configuration": node.Configuration.Data(),
		}
		if node.WebhookID != nil {
			var webhook Webhook
			err := tx.Unscoped().Where("id = ?", *node.WebhookID).First(&webhook).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil {
				entry["remoteIds"] = webhook.Metadata.Data()
			}
		}
		inventory = append(inventory, entry)
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&FactoryIntakeRetirement{
		CanvasID: i.CanvasID, IntakeID: i.ID, OrganizationID: i.OrganizationID, FactoryID: i.FactoryID,
		Inventory: datatypes.NewJSONType(inventory), RetiredAt: time.Now(),
	}).Error
}

// ListActiveAutomations omits retired managed canvases while direct history lookups retain access.
func (f *Factory) ListActiveAutomations(tx *gorm.DB) ([]Canvas, error) {
	return f.ListCanvases(tx.Where(`NOT EXISTS (SELECT 1 FROM factory_intake_retirements r WHERE r.canvas_id = workflows.id)`).
		Where(`NOT EXISTS (SELECT 1 FROM factory_intakes i WHERE i.canvas_id = workflows.id AND i.source = ?)`, RetiredProductiveIntakeSource))
}

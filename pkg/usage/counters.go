package usage

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type UsageCounters struct {
	Canvases     int64
	Users        int64
	Integrations int64
	EventsMonth  int64
}

func GetUsageCounters(orgID uuid.UUID) (*UsageCounters, error) {
	return GetUsageCountersInTransaction(database.Conn(), orgID)
}

func GetUsageCountersInTransaction(tx *gorm.DB, orgID uuid.UUID) (*UsageCounters, error) {
	counters := &UsageCounters{}

	err := tx.Model(&models.Canvas{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&counters.Canvases).Error
	if err != nil {
		return nil, err
	}

	err = tx.Model(&models.User{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&counters.Users).Error
	if err != nil {
		return nil, err
	}

	err = tx.Model(&models.Integration{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&counters.Integrations).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	err = tx.Table("workflow_events").
		Joins("JOIN workflows ON workflows.id = workflow_events.workflow_id").
		Where("workflows.organization_id = ?", orgID).
		Where("workflow_events.created_at >= ?", startOfMonth).
		Count(&counters.EventsMonth).Error
	if err != nil {
		return nil, err
	}

	return counters, nil
}

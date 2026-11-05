package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FactoryWorkOrderEvent struct {
	ID          uuid.UUID
	WorkOrderID uuid.UUID
	Type        string
	Data        datatypes.JSON
	CreatedAt   time.Time
}

type FactoryWorkOrderStart struct {
	User      *User
	Automatic bool
	StartedAt time.Time
}

// StartAttribution reads the first start event, independently of later ownership
// or lifecycle changes. An automated first start has no person to attribute.
func (o *FactoryWorkOrder) StartAttribution(tx *gorm.DB) (*FactoryWorkOrderStart, error) {
	var event FactoryWorkOrderEvent
	err := tx.Where("work_order_id = ? AND type = ?", o.ID, factory.EventTypeOrderStatusUpdated).
		Where("data->>'fromState' = ? AND data->>'toState' = ?", FactoryWorkOrderStateDraft, FactoryWorkOrderStateOpen).
		Order("created_at ASC").Order("id ASC").Take(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var status factory.WorkOrderStatusUpdated
	if err := json.Unmarshal(event.Data, &status); err != nil {
		return nil, err
	}
	start := &FactoryWorkOrderStart{Automatic: status.User == nil, StartedAt: event.CreatedAt}
	if status.User == nil {
		return start, nil
	}
	users, err := FindUnscopedUsersByIDs(tx, []uuid.UUID{status.User.ID})
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		start.User = &users[0]
	}
	return start, nil
}

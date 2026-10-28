package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type OrganizationPaymentMethodObservation struct {
	OrganizationID     uuid.UUID `gorm:"primaryKey"`
	Usable             bool
	ReadStartedAt      int64
	ObservedAt         time.Time
	UsableMethodHashes pq.StringArray `gorm:"type:text[]"`
	TransitionID       *uuid.UUID
}

type PaymentMethodObservation struct {
	OrganizationID uuid.UUID
	ReadStartedAt  time.Time
	ObservedAt     time.Time
	UsableHashes   []string
	PresentHashes  map[string]bool // Value indicates proven expiration.
}

type PaymentMethodLoss struct {
	OccurrenceID uuid.UUID
	ObservedAt   time.Time
	Reason       string
}

// ObserveOrganizationPaymentMethods owns a short transaction. The organization
// lock serializes initial inserts, transitions, and deletion across processes.
func ObserveOrganizationPaymentMethods(db *gorm.DB, observation PaymentMethodObservation) (*PaymentMethodLoss, error) {
	var loss *PaymentMethodLoss
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, observation.OrganizationID); err != nil {
			return err
		}
		var previous OrganizationPaymentMethodObservation
		err := tx.Where("organization_id = ?", observation.OrganizationID).First(&previous).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		initial := errors.Is(err, gorm.ErrRecordNotFound)
		if !initial && observation.ReadStartedAt.UnixNano() <= previous.ReadStartedAt {
			return nil
		}
		usable := len(observation.UsableHashes) > 0
		if !initial && previous.Usable && !usable {
			loss = &PaymentMethodLoss{OccurrenceID: uuid.New(), ObservedAt: observation.ObservedAt,
				Reason: paymentMethodLossReason(previous.UsableMethodHashes, observation.PresentHashes)}
			previous.TransitionID = &loss.OccurrenceID
		}
		previous.OrganizationID = observation.OrganizationID
		previous.Usable = usable
		previous.ReadStartedAt = observation.ReadStartedAt.UnixNano()
		previous.ObservedAt = observation.ObservedAt
		previous.UsableMethodHashes = pq.StringArray(observation.UsableHashes)
		if previous.UsableMethodHashes == nil {
			previous.UsableMethodHashes = pq.StringArray{}
		}
		if initial {
			return tx.Create(&previous).Error
		}
		return tx.Save(&previous).Error
	})
	if err != nil {
		return nil, err
	}
	return loss, nil
}

func paymentMethodLossReason(previous []string, present map[string]bool) string {
	if len(previous) == 0 {
		return "unknown"
	}
	absent, expired := 0, 0
	for _, hash := range previous {
		isExpired, exists := present[hash]
		if !exists {
			absent++
			continue
		}
		if isExpired {
			expired++
		}
	}
	if absent == len(previous) {
		return "removed"
	}
	if expired == len(previous) {
		return "expired"
	}
	return "unknown"
}

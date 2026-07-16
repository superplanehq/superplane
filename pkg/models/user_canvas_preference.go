package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserCanvasPreference struct {
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	UserID         uuid.UUID `gorm:"primaryKey"`
	CanvasID       uuid.UUID `gorm:"primaryKey"`
	StarredAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (p *UserCanvasPreference) TableName() string {
	return "user_canvas_preferences"
}

func FindUserCanvasPreferencesForCanvases(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasIDs []uuid.UUID,
) (map[uuid.UUID]UserCanvasPreference, error) {
	preferencesByCanvasID := map[uuid.UUID]UserCanvasPreference{}
	if len(canvasIDs) == 0 {
		return preferencesByCanvasID, nil
	}

	var preferences []UserCanvasPreference
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id IN ?", canvasIDs).
		Find(&preferences).
		Error
	if err != nil {
		return nil, err
	}

	for _, preference := range preferences {
		preferencesByCanvasID[preference.CanvasID] = preference
	}

	return preferencesByCanvasID, nil
}

func SetUserCanvasPreference(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	starred *bool,
) (*UserCanvasPreference, error) {
	if err := ensureCanvasExistsForPreference(tx, organizationID, canvasID); err != nil {
		return nil, err
	}

	if starred == nil {
		return findUserCanvasPreference(tx, organizationID, userID, canvasID)
	}

	preference, err := lockUserCanvasPreference(tx, organizationID, userID, canvasID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return createUserCanvasPreference(tx, organizationID, userID, canvasID, starred)
	}

	if err != nil {
		return nil, err
	}

	applyUserCanvasPreferenceUpdate(preference, starred, time.Now())
	if preference.StarredAt == nil {
		if err := tx.Delete(preference).Error; err != nil {
			return nil, err
		}
		return preference, nil
	}

	if err := tx.Save(preference).Error; err != nil {
		return nil, err
	}

	return preference, nil
}

func findUserCanvasPreference(tx *gorm.DB, organizationID, userID, canvasID uuid.UUID) (*UserCanvasPreference, error) {
	var preference UserCanvasPreference
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id = ?", canvasID).
		First(&preference).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &UserCanvasPreference{
			OrganizationID: organizationID,
			UserID:         userID,
			CanvasID:       canvasID,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return &preference, nil
}

func ensureCanvasExistsForPreference(tx *gorm.DB, organizationID, canvasID uuid.UUID) error {
	var canvas Canvas
	return tx.
		Select("id").
		Where("organization_id = ?", organizationID).
		Where("id = ?", canvasID).
		First(&canvas).
		Error
}

func lockUserCanvasPreference(tx *gorm.DB, organizationID, userID, canvasID uuid.UUID) (*UserCanvasPreference, error) {
	var preference UserCanvasPreference
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id = ?", canvasID).
		First(&preference).
		Error
	if err != nil {
		return nil, err
	}

	return &preference, nil
}

func createUserCanvasPreference(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	starred *bool,
) (*UserCanvasPreference, error) {
	now := time.Now()
	preference := &UserCanvasPreference{
		OrganizationID: organizationID,
		UserID:         userID,
		CanvasID:       canvasID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	applyUserCanvasPreferenceUpdate(preference, starred, now)
	if preference.StarredAt == nil {
		return preference, nil
	}

	if err := tx.Create(preference).Error; err != nil {
		return nil, err
	}

	return preference, nil
}

func applyUserCanvasPreferenceUpdate(
	preference *UserCanvasPreference,
	starred *bool,
	now time.Time,
) {
	preference.UpdatedAt = now
	if starred != nil {
		preference.StarredAt = timestampIfEnabled(*starred, now)
	}
}

func timestampIfEnabled(enabled bool, now time.Time) *time.Time {
	if !enabled {
		return nil
	}

	return &now
}

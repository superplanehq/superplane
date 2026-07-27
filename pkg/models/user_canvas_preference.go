package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserCanvasPreference struct {
	OrganizationID              uuid.UUID `gorm:"primaryKey"`
	UserID                      uuid.UUID `gorm:"primaryKey"`
	CanvasID                    uuid.UUID `gorm:"primaryKey"`
	StarredAt                   *time.Time
	DismissedAgentSuggestionIDs datatypes.JSONSlice[string]
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

// UserCanvasPreferenceChanges is a partial update for a user/canvas preference row.
// Nil fields are left unchanged.
type UserCanvasPreferenceChanges struct {
	Starred                  *bool
	DismissAgentSuggestionID *string
}

func (p *UserCanvasPreference) TableName() string {
	return "user_canvas_preferences"
}

func (p *UserCanvasPreference) isEmpty() bool {
	return p.StarredAt == nil && len(p.DismissedAgentSuggestionIDs) == 0
}

func (c UserCanvasPreferenceChanges) hasUpdates() bool {
	return c.Starred != nil || c.DismissAgentSuggestionID != nil
}

func FindUserCanvasPreference(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
) (*UserCanvasPreference, error) {
	return findUserCanvasPreference(tx, organizationID, userID, canvasID)
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
	changes UserCanvasPreferenceChanges,
) (*UserCanvasPreference, error) {
	if err := ensureCanvasExistsForPreference(tx, organizationID, canvasID); err != nil {
		return nil, err
	}

	if !changes.hasUpdates() {
		return findUserCanvasPreference(tx, organizationID, userID, canvasID)
	}

	preference, err := lockUserCanvasPreference(tx, organizationID, userID, canvasID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return createUserCanvasPreference(tx, organizationID, userID, canvasID, changes)
	}

	if err != nil {
		return nil, err
	}

	applyUserCanvasPreferenceUpdate(preference, changes, time.Now())
	if preference.isEmpty() {
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
	changes UserCanvasPreferenceChanges,
) (*UserCanvasPreference, error) {
	now := time.Now()
	preference := &UserCanvasPreference{
		OrganizationID:              organizationID,
		UserID:                      userID,
		CanvasID:                    canvasID,
		DismissedAgentSuggestionIDs: datatypes.JSONSlice[string]{},
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
	applyUserCanvasPreferenceUpdate(preference, changes, now)
	if preference.isEmpty() {
		return preference, nil
	}

	if err := tx.Create(preference).Error; err != nil {
		return nil, err
	}

	return preference, nil
}

func applyUserCanvasPreferenceUpdate(
	preference *UserCanvasPreference,
	changes UserCanvasPreferenceChanges,
	now time.Time,
) {
	preference.UpdatedAt = now
	if changes.Starred != nil {
		preference.StarredAt = timestampIfEnabled(*changes.Starred, now)
	}
	if changes.DismissAgentSuggestionID != nil {
		appendDismissedAgentSuggestionID(preference, *changes.DismissAgentSuggestionID)
	}
}

func appendDismissedAgentSuggestionID(preference *UserCanvasPreference, suggestionID string) {
	if suggestionID == "" {
		return
	}
	if slices.Contains(preference.DismissedAgentSuggestionIDs, suggestionID) {
		return
	}
	preference.DismissedAgentSuggestionIDs = append(preference.DismissedAgentSuggestionIDs, suggestionID)
}

func timestampIfEnabled(enabled bool, now time.Time) *time.Time {
	if !enabled {
		return nil
	}

	return &now
}

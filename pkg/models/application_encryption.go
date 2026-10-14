package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationEncryptionKey struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	WrappedKey   []byte
	MasterKeyRef string
	CreatedAt    time.Time
}

type ApplicationEncryptionState struct {
	ID          int `gorm:"primaryKey"`
	ActiveKeyID uuid.UUID
	Generation  int64
	UpdatedAt   time.Time
}

func (ApplicationEncryptionState) TableName() string {
	return "application_encryption_state"
}

func LoadApplicationEncryptionState(tx *gorm.DB) (*ApplicationEncryptionState, error) {
	var state ApplicationEncryptionState
	if err := tx.First(&state, 1).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func LockApplicationEncryptionState(tx *gorm.DB) (*ApplicationEncryptionState, error) {
	var state ApplicationEncryptionState
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&state, 1).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func ListApplicationEncryptionKeys(tx *gorm.DB) ([]ApplicationEncryptionKey, error) {
	var keys []ApplicationEncryptionKey
	if err := tx.Order("created_at, id").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func LoadApplicationEncryptionKey(tx *gorm.DB, id uuid.UUID) (*ApplicationEncryptionKey, error) {
	var key ApplicationEncryptionKey
	if err := tx.First(&key, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func CreateApplicationEncryptionKey(tx *gorm.DB, key *ApplicationEncryptionKey) error {
	return tx.Create(key).Error
}

func CreateApplicationEncryptionState(tx *gorm.DB, state *ApplicationEncryptionState) error {
	return tx.Create(state).Error
}

func ActivateApplicationEncryptionKey(tx *gorm.DB, state *ApplicationEncryptionState, id uuid.UUID) error {
	return tx.Model(state).Updates(map[string]any{
		"active_key_id": id,
		"generation":    state.Generation + 1,
		"updated_at":    time.Now(),
	}).Error
}

func RewrapApplicationEncryptionKey(tx *gorm.DB, key *ApplicationEncryptionKey, wrapped []byte, reference string) error {
	return tx.Model(key).Updates(map[string]any{
		"wrapped_key":    wrapped,
		"master_key_ref": reference,
	}).Error
}

func AdvanceApplicationEncryptionGeneration(tx *gorm.DB, state *ApplicationEncryptionState) error {
	return tx.Model(state).Updates(map[string]any{
		"generation": state.Generation + 1,
		"updated_at": time.Now(),
	}).Error
}

func HasUnfinishedApplicationEncryptionMigration(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Table("application_encryption_migrations").
		Where("state IN ?", []string{"pending", "running", "failed"}).
		Count(&count).Error
	return count > 0, err
}

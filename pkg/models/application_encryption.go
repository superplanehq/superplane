package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ApplicationEncryptionKeyActive    = "active"
	ApplicationEncryptionKeyAvailable = "available"
)

type ApplicationEncryptionKey struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	WrappedKey   []byte
	MasterKeyRef string
	State        string
	CreatedAt    time.Time
}

func (k *ApplicationEncryptionKey) CreateInitial(tx *gorm.DB) error {
	k.State = ApplicationEncryptionKeyActive
	return tx.Create(k).Error
}

func (k *ApplicationEncryptionKey) CreateAndActivate(tx *gorm.DB, current *ApplicationEncryptionKey) error {
	if err := tx.Create(k).Error; err != nil {
		return err
	}
	if err := tx.Model(current).Update("state", ApplicationEncryptionKeyAvailable).Error; err != nil {
		return err
	}
	return tx.Model(k).Update("state", ApplicationEncryptionKeyActive).Error
}

func (k *ApplicationEncryptionKey) Rewrap(tx *gorm.DB, wrapped []byte, reference string) error {
	return tx.Model(k).Updates(map[string]any{
		"wrapped_key":    wrapped,
		"master_key_ref": reference,
	}).Error
}

func LoadActiveApplicationEncryptionKey(tx *gorm.DB) (*ApplicationEncryptionKey, error) {
	var key ApplicationEncryptionKey
	if err := tx.Where("state = ?", ApplicationEncryptionKeyActive).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func LockActiveApplicationEncryptionKey(tx *gorm.DB) (*ApplicationEncryptionKey, error) {
	var key ApplicationEncryptionKey
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("state = ?", ApplicationEncryptionKeyActive).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
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

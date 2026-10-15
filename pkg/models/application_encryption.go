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
	State        string
	CreatedAt    time.Time
}

const (
	ApplicationEncryptionKeyActive    = "active"
	ApplicationEncryptionKeyAvailable = "available"
)

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

func ActivateApplicationEncryptionKey(tx *gorm.DB, current, next *ApplicationEncryptionKey) error {
	if err := tx.Model(current).Update("state", ApplicationEncryptionKeyAvailable).Error; err != nil {
		return err
	}
	return tx.Model(next).Update("state", ApplicationEncryptionKeyActive).Error
}

func RewrapApplicationEncryptionKey(tx *gorm.DB, key *ApplicationEncryptionKey, wrapped []byte, reference string) error {
	return tx.Model(key).Updates(map[string]any{
		"wrapped_key":    wrapped,
		"master_key_ref": reference,
	}).Error
}

func HasUnfinishedApplicationEncryptionMigration(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Table("application_encryption_migrations").
		Where("state IN ?", []string{"pending", "running", "failed"}).
		Count(&count).Error
	return count > 0, err
}

package models

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const createRequestKeySeparator = "\x1e"
const createRequestKeyPrefix = "create:"

func AppendCreateRequestKey(label, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return label
	}
	label = StripCreateRequestKey(label)
	token := createRequestKeyToken(key)
	if label == "" {
		return token
	}
	return label + token
}

func CreateRequestKeyFromLabel(label string) string {
	_, rest, ok := strings.Cut(label, createRequestKeySeparator+createRequestKeyPrefix)
	if !ok {
		return ""
	}
	rest, _, _ = strings.Cut(rest, createRequestKeySeparator)
	return strings.TrimSpace(rest)
}

func StripCreateRequestKey(label string) string {
	label, _, _ = strings.Cut(label, createRequestKeySeparator+createRequestKeyPrefix)
	return label
}

func createRequestKeyToken(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return createRequestKeySeparator + createRequestKeyPrefix + key
}

func LockWorkOrderCreateRequest(tx *gorm.DB, factoryID, createdBy uuid.UUID, key string) error {
	if tx == nil || key == "" {
		return nil
	}
	sum := sha256.New()
	sum.Write([]byte("work-order-create:"))
	sum.Write(factoryID[:])
	sum.Write(createdBy[:])
	sum.Write([]byte(key))
	digest := sum.Sum(nil)
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(binary.BigEndian.Uint64(digest[:8]))).Error
}

func (f *Factory) FindWorkOrderByCreateRequestKey(tx *gorm.DB, createdBy uuid.UUID, key string) (*FactoryWorkOrder, error) {
	token := createRequestKeyToken(key)
	if tx == nil || f == nil || token == "" || createdBy == uuid.Nil {
		return nil, nil
	}

	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND created_by_id = ?", f.OrganizationID, f.ID, createdBy).
		Where("position(? in coalesce(origin_label, '')) > 0", token).
		Order("created_at ASC").
		First(&order).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

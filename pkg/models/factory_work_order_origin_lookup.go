package models

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net/url"
	"strings"

	"gorm.io/gorm"
)

// NormalizeWorkOrderOriginURL returns the trimmed link when it is an http or
// https URL with a host, and "" otherwise.
func NormalizeWorkOrderOriginURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return trimmed
}

// LockWorkOrderOriginURL holds a transaction lock on one origin link of this
// factory, so two requests with the same link cannot both create a task.
func (f *Factory) LockWorkOrderOriginURL(tx *gorm.DB, originURL string) error {
	sum := sha256.New()
	sum.Write([]byte("work-order-origin-url:"))
	sum.Write(f.ID[:])
	sum.Write([]byte(originURL))
	key := int64(binary.BigEndian.Uint64(sum.Sum(nil)[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
}

// FindWorkOrderByOriginURL returns this factory's oldest work order whose
// origin_url is exactly originURL, in every state, or nil.
func (f *Factory) FindWorkOrderByOriginURL(tx *gorm.DB, originURL string) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND origin_url = ?", f.OrganizationID, f.ID, originURL).
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

// ListWorkOrderOriginURLsContaining returns origin URLs of this factory's
// work orders whose origin_url contains fragment. Matching covers every
// state. Callers confirm the match; this lookup is a coarse filter.
func (f *Factory) ListWorkOrderOriginURLsContaining(tx *gorm.DB, fragment string) ([]string, error) {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return nil, nil
	}

	var urls []string
	err := tx.
		Model(&FactoryWorkOrder{}).
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Where("origin_url ILIKE ?", "%"+fragment+"%").
		Pluck("origin_url", &urls).
		Error
	if err != nil {
		return nil, err
	}

	return urls, nil
}

// ListWorkOrdersByOriginURLFragment returns this factory's work orders whose
// origin_url contains fragment, in every state. Callers confirm the match and
// read the state; this lookup is a coarse filter.
func (f *Factory) ListWorkOrdersByOriginURLFragment(tx *gorm.DB, fragment string) ([]FactoryWorkOrder, error) {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return nil, nil
	}

	var orders []FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Where("origin_url ILIKE ?", "%"+fragment+"%").
		Order("created_at ASC").
		Find(&orders).
		Error
	if err != nil {
		return nil, err
	}

	return orders, nil
}

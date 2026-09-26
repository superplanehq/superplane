package models

import (
	"strings"

	"gorm.io/gorm"
)

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

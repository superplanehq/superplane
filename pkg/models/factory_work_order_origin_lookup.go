package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
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

// FindWorkOrderByPendingGitHubMarker returns the work order that is still
// waiting for this manual-task marker. The stored label may also carry the
// GitHub login that is allowed to claim the issue.
func (f *Factory) FindWorkOrderByPendingGitHubMarker(tx *gorm.DB, marker string) (*FactoryWorkOrder, error) {
	marker = strings.TrimSpace(marker)
	if tx == nil || f == nil || marker == "" {
		return nil, nil
	}

	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Where("origin_url IS NULL OR origin_url = ''").
		Where("origin_label = ? OR origin_label LIKE ?", marker, marker+"\x1f%").
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

// FindRecentIdenticalManualWorkOrder returns a manual task with the same
// title and description created by the same user at or after since. A client
// retry after a canceled create uses this so it does not open a second task.
func (f *Factory) FindRecentIdenticalManualWorkOrder(
	tx *gorm.DB,
	createdBy uuid.UUID,
	title, description string,
	since time.Time,
) (*FactoryWorkOrder, error) {
	title = strings.TrimSpace(title)
	if tx == nil || f == nil || createdBy == uuid.Nil || title == "" || since.IsZero() {
		return nil, nil
	}

	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Where("created_by_id = ?", createdBy).
		Where("source_run_id IS NULL").
		Where("title = ?", title).
		Where("description = ?", description).
		Where("created_at >= ?", since).
		Order("created_at DESC").
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

// FindWorkOrderByOriginLabel returns the work order whose origin label is
// exactly label, in any state. A pending manual-task marker uses this until
// the GitHub issue URL is known.
func (f *Factory) FindWorkOrderByOriginLabel(tx *gorm.DB, label string) (*FactoryWorkOrder, error) {
	label = strings.TrimSpace(label)
	if tx == nil || f == nil || label == "" {
		return nil, nil
	}

	order, err := f.findWorkOrder(tx, "origin_label = ?", label)
	if errors.Is(err, ErrFactoryWorkOrderNotFound) {
		return nil, nil
	}
	return order, err
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

package models

import (
	"fmt"
	"strconv"
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

// FindWorkOrderByGitHubIssue finds the work order created from a GitHub issue,
// matching by repository and issue number. Returns nil if no work order exists
// for the given issue. This follows the same pattern as Jira issue lookups.
func (f *Factory) FindWorkOrderByGitHubIssue(tx *gorm.DB, repository string, issueNumber int) (*FactoryWorkOrder, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" || issueNumber <= 0 {
		return nil, nil
	}

	// Build the expected URL pattern: https://github.com/{owner}/{repo}/issues/{number}
	urlPattern := fmt.Sprintf("github.com/%s/issues/%d", repository, issueNumber)

	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Where("origin_url ILIKE ?", "%"+urlPattern+"%").
		Order("created_at ASC").
		First(&order).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}

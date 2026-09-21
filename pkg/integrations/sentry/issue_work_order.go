package sentry

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// IssueHasWorkOrder reports whether this factory already has a work order
// for the Sentry issue. Matching covers every work-order state.
func IssueHasWorkOrder(tx *gorm.DB, factory *models.Factory, issueID string) (bool, error) {
	issueID = strings.TrimSpace(issueID)
	if factory == nil || issueID == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, IssueURLFragment(issueID))
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, ok := IssueIDFromURL(rawURL)
		if ok && found == issueID {
			return true, nil
		}
	}

	return false, nil
}

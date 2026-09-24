package sentry

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LockIssueWorkOrder serializes work-order creation for one Sentry issue in
// this factory. The lock is held until the caller commits tx.
func LockIssueWorkOrder(tx *gorm.DB, factory *models.Factory, issueID string) error {
	if tx == nil || factory == nil {
		return nil
	}
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", sentryIssueLockKey(factory.ID, issueID)).Error
}

func sentryIssueLockKey(factoryID uuid.UUID, issueID string) int64 {
	sum := sha256.New()
	sum.Write([]byte("sentry-issue-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(issueID))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

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

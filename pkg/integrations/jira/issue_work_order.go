package jira

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LockIssueWorkOrder serializes work-order creation for one Jira issue on
// one site in this factory. The lock is held until the caller commits tx.
func LockIssueWorkOrder(tx *gorm.DB, factory *models.Factory, ref IssueRef) error {
	if tx == nil || factory == nil || !ref.valid() {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", jiraIssueLockKey(factory.ID, ref)).Error
}

func jiraIssueLockKey(factoryID uuid.UUID, ref IssueRef) int64 {
	sum := sha256.New()
	sum.Write([]byte("jira-issue-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(ref.Host))
	sum.Write([]byte{0})
	sum.Write([]byte(ref.Key))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// IssueHasWorkOrder reports whether this factory already has a work order
// for the Jira issue on this site. Matching covers every work-order state.
func IssueHasWorkOrder(tx *gorm.DB, factory *models.Factory, ref IssueRef) (bool, error) {
	if factory == nil || !ref.valid() {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, IssueURLFragment(ref))
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, ok := IssueRefFromURL(rawURL)
		if ok && found == ref {
			return true, nil
		}
	}

	return false, nil
}

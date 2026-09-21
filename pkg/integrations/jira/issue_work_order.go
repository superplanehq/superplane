package jira

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LockIssueWorkOrder serializes work-order creation for one Jira issue in
// this factory. The lock is held until the caller commits tx.
func LockIssueWorkOrder(tx *gorm.DB, factory *models.Factory, issueKey string) error {
	if tx == nil || factory == nil {
		return nil
	}
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", jiraIssueLockKey(factory.ID, issueKey)).Error
}

func jiraIssueLockKey(factoryID uuid.UUID, issueKey string) int64 {
	sum := sha256.New()
	sum.Write([]byte("jira-issue-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(issueKey))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// IssueHasWorkOrder reports whether this factory already has a work order
// for the Jira issue. Matching covers every work-order state.
func IssueHasWorkOrder(tx *gorm.DB, factory *models.Factory, issueKey string) (bool, error) {
	issueKey = strings.TrimSpace(issueKey)
	if factory == nil || issueKey == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, IssueURLFragment(issueKey))
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, ok := IssueKeyFromURL(rawURL)
		if ok && found == issueKey {
			return true, nil
		}
	}

	return false, nil
}

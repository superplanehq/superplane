package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FactoryPullRequestRevision struct {
	ID            uuid.UUID
	PullRequestID uuid.UUID
	SHA           string
	ObservedAt    time.Time
}

type FactoryPullRequestObserveResult struct {
	Revision *FactoryPullRequestRevision
	Current  bool
}

func (FactoryPullRequestRevision) TableName() string {
	return "factory_pull_request_revisions"
}

func (p *FactoryPullRequest) lock(tx *gorm.DB) error {
	var locked FactoryPullRequest
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", p.ID).
		First(&locked).
		Error
	if err != nil {
		return err
	}
	*p = locked
	return nil
}

func (p *FactoryPullRequest) ObserveRevision(tx *gorm.DB, sha string) (*FactoryPullRequestObserveResult, error) {
	if err := p.lock(tx); err != nil {
		return nil, err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, fmt.Errorf("%w: revision sha is required", ErrFactoryPullRequestInvalid)
	}

	revision, created, err := findOrCreatePullRequestRevision(tx, p.ID, sha)
	if err != nil {
		return nil, err
	}

	result := &FactoryPullRequestObserveResult{Revision: revision}
	if p.CurrentRevisionID != nil && *p.CurrentRevisionID == revision.ID {
		result.Current = true
		return result, nil
	}
	if !created && p.CurrentRevisionID != nil {
		return result, nil
	}

	// A newer head updates the current pointer. Do not cancel runs
	// still bound to the former SHA. A check fixer often creates that head.
	p.CurrentRevisionID = &revision.ID
	if err := tx.Model(p).Update("current_revision_id", revision.ID).Error; err != nil {
		return nil, err
	}
	result.Current = true
	return result, nil
}

func FindPullRequestRevision(tx *gorm.DB, id uuid.UUID) (*FactoryPullRequestRevision, error) {
	var revision FactoryPullRequestRevision
	err := tx.Where("id = ?", id).First(&revision).Error
	if err != nil {
		return nil, err
	}
	return &revision, nil
}

func findOrCreatePullRequestRevision(tx *gorm.DB, pullRequestID uuid.UUID, sha string) (*FactoryPullRequestRevision, bool, error) {
	var existing FactoryPullRequestRevision
	err := tx.Where("pull_request_id = ? AND sha = ?", pullRequestID, sha).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	now := time.Now()
	revision := &FactoryPullRequestRevision{
		ID:            uuid.New(),
		PullRequestID: pullRequestID,
		SHA:           sha,
		ObservedAt:    now,
	}
	if err := tx.Create(revision).Error; err != nil {
		if isPullRequestRevisionSHAConflict(err) {
			if reloadErr := tx.Where("pull_request_id = ? AND sha = ?", pullRequestID, sha).First(&existing).Error; reloadErr != nil {
				return nil, false, reloadErr
			}
			return &existing, false, nil
		}
		return nil, false, err
	}
	return revision, true, nil
}

func isPullRequestRevisionSHAConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == factoryPullRequestRevisionSHAUniqueConstraint
}

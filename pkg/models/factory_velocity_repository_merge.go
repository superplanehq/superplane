package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Who wrote the merge. The velocity report draws one series per source.
const (
	// FactoryVelocityMergeSourcePeople is work a person wrote by hand.
	FactoryVelocityMergeSourcePeople = "people"

	// FactoryVelocityMergeSourceAgent is work the SuperPlane agent wrote. The
	// sync recognizes it by the agent co-author trailer on the merge commit,
	// which names the agent whichever instance opened the pull request.
	FactoryVelocityMergeSourceAgent = "agent"
)

const factoryVelocityRepositoryMergeBatchSize = 200

// FactoryVelocityRepositoryMerge is one merged pull request of a workspace's
// repository that SuperPlane did not open. The sync worker excludes the pull
// requests this instance opened before it stores a window, so this table and
// factory_pull_requests describe disjoint sets of pull requests.
//
// MergedAt keeps the exact merge instant, so the velocity chart buckets a merge
// into the correct day whatever timezone it is rendered in.
type FactoryVelocityRepositoryMerge struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	Repository      string
	Number          int64
	Source          string
	AuthorLogin     string
	AuthorName      string
	AuthorAvatarURL string
	MergedAt        time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewFactoryVelocityRepositoryMerge normalizes the repository so the unique
// index matches whatever case the provider reported.
func NewFactoryVelocityRepositoryMerge(
	organizationID, factoryID uuid.UUID,
	repository string,
	number int64,
	source string,
	mergedAt time.Time,
) FactoryVelocityRepositoryMerge {
	now := time.Now()
	return FactoryVelocityRepositoryMerge{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		Repository:     strings.ToLower(strings.TrimSpace(repository)),
		Number:         number,
		Source:         source,
		MergedAt:       mergedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (FactoryVelocityRepositoryMerge) TableName() string {
	return "factory_velocity_repository_merges"
}

// IsAgent reports whether the SuperPlane agent wrote the merge.
func (m FactoryVelocityRepositoryMerge) IsAgent() bool {
	return m.Source == FactoryVelocityMergeSourceAgent
}

// ReplaceFactoryVelocityRepositoryMerges swaps the stored merges of a window for
// a freshly computed set.
//
// The sync recomputes a whole window rather than adding to it, so a pull request
// that SuperPlane recorded after the merge was first seen, or a merge that
// changed source, stops counting the old way on the next tick.
func ReplaceFactoryVelocityRepositoryMerges(
	tx *gorm.DB,
	factoryID uuid.UUID,
	from, to time.Time,
	merges []FactoryVelocityRepositoryMerge,
) error {
	err := tx.
		Where("factory_id = ? AND merged_at >= ? AND merged_at < ?", factoryID, from, to).
		Delete(&FactoryVelocityRepositoryMerge{}).Error
	if err != nil {
		return err
	}

	if len(merges) == 0 {
		return nil
	}

	// A merge that arrives twice in one window, or one just outside the
	// deleted range, must not break the write.
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "factory_id"},
			{Name: "repository"},
			{Name: "number"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"source",
			"author_login",
			"author_name",
			"author_avatar_url",
			"merged_at",
			"updated_at",
		}),
	}).CreateInBatches(merges, factoryVelocityRepositoryMergeBatchSize).Error
}

// DeleteFactoryVelocityRepositoryMerges drops every stored merge of a workspace.
// The sync calls it when the workspace changed its repository, so merges of the
// repository it no longer reports on do not linger.
func DeleteFactoryVelocityRepositoryMerges(tx *gorm.DB, factoryID uuid.UUID) error {
	return tx.Where("factory_id = ?", factoryID).
		Delete(&FactoryVelocityRepositoryMerge{}).Error
}

// ListFactoryVelocityRepositoryMerges returns the merges of [from, to), oldest
// first.
func ListFactoryVelocityRepositoryMerges(
	tx *gorm.DB,
	factoryID uuid.UUID,
	from, to time.Time,
) ([]FactoryVelocityRepositoryMerge, error) {
	var merges []FactoryVelocityRepositoryMerge
	err := tx.
		Where("factory_id = ? AND merged_at >= ? AND merged_at < ?", factoryID, from, to).
		Order("merged_at ASC").
		Find(&merges).Error
	if err != nil {
		return nil, err
	}
	return merges, nil
}

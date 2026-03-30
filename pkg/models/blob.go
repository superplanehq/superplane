package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	BlobScopeOrganization = "organization"
	BlobScopeCanvas       = "canvas"
	BlobScopeNode         = "node"
	BlobScopeExecution    = "execution"
)

type Blob struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	ScopeType       string     `gorm:"type:varchar(32);not null;index"`
	CanvasID        *uuid.UUID `gorm:"type:uuid;index"`
	NodeID          *string    `gorm:"type:varchar(255);index"`
	ExecutionID     *uuid.UUID `gorm:"type:uuid;index"`
	Path            string     `gorm:"type:text;not null"`
	ObjectKey       string     `gorm:"type:text;not null;uniqueIndex"`
	SizeBytes       int64      `gorm:"type:bigint;not null"`
	ContentType     string     `gorm:"type:text"`
	CreatedByUserID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func (Blob) TableName() string {
	return "blobs"
}

func CreateBlobInTransaction(tx *gorm.DB, blob *Blob) error {
	return tx.Create(blob).Error
}

func CreateBlob(blob *Blob) error {
	return CreateBlobInTransaction(database.Conn(), blob)
}

func FindBlobInTransaction(tx *gorm.DB, id uuid.UUID) (*Blob, error) {
	var blob Blob
	if err := tx.Where("id = ?", id).First(&blob).Error; err != nil {
		return nil, err
	}

	return &blob, nil
}

func FindBlob(id uuid.UUID) (*Blob, error) {
	return FindBlobInTransaction(database.Conn(), id)
}

func ListBlobsByScopeInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	scopeType string,
	canvasID *uuid.UUID,
	nodeID *string,
	executionID *uuid.UUID,
	limit int,
	before *time.Time,
) ([]Blob, error) {
	query := tx.
		Where("organization_id = ?", organizationID).
		Where("scope_type = ?", scopeType)

	if canvasID != nil {
		query = query.Where("canvas_id = ?", *canvasID)
	} else {
		query = query.Where("canvas_id IS NULL")
	}

	if nodeID != nil {
		query = query.Where("node_id = ?", *nodeID)
	} else {
		query = query.Where("node_id IS NULL")
	}

	if executionID != nil {
		query = query.Where("execution_id = ?", *executionID)
	} else {
		query = query.Where("execution_id IS NULL")
	}

	if before != nil {
		query = query.Where("created_at < ?", *before)
	}

	// Callers may request one extra row for pagination lookahead.
	if limit <= 0 || limit > 101 {
		limit = 101
	}

	var blobs []Blob
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Find(&blobs).
		Error
	if err != nil {
		return nil, err
	}

	return blobs, nil
}

func ListBlobsByScope(
	organizationID uuid.UUID,
	scopeType string,
	canvasID *uuid.UUID,
	nodeID *string,
	executionID *uuid.UUID,
	limit int,
	before *time.Time,
) ([]Blob, error) {
	return ListBlobsByScopeInTransaction(database.Conn(), organizationID, scopeType, canvasID, nodeID, executionID, limit, before)
}

func DeleteBlobInTransaction(tx *gorm.DB, id uuid.UUID) error {
	return tx.Where("id = ?", id).Delete(&Blob{}).Error
}

func DeleteBlob(id uuid.UUID) error {
	return DeleteBlobInTransaction(database.Conn(), id)
}

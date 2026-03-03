package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasDraftNotFound = errors.New("canvas draft not found")
var ErrCanvasVersionConflict = errors.New("canvas version conflict")

type CanvasVersion struct {
	ID               uuid.UUID
	WorkflowID       uuid.UUID
	Revision         int
	OwnerID          *uuid.UUID
	BasedOnVersionID *uuid.UUID
	IsPublished      bool
	PublishedAt      *time.Time
	Nodes            datatypes.JSONSlice[Node]
	Edges            datatypes.JSONSlice[Edge]
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func (c *CanvasVersion) TableName() string {
	return "workflow_versions"
}

type CanvasUserDraft struct {
	WorkflowID uuid.UUID `gorm:"primaryKey"`
	UserID     uuid.UUID `gorm:"primaryKey"`
	VersionID  uuid.UUID
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (c *CanvasUserDraft) TableName() string {
	return "workflow_user_drafts"
}

func FindCanvasVersionInTransaction(tx *gorm.DB, workflowID, versionID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("id = ?", versionID).
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return &version, nil
}

func FindCanvasVersion(workflowID, versionID uuid.UUID) (*CanvasVersion, error) {
	return FindCanvasVersionInTransaction(database.Conn(), workflowID, versionID)
}

func ListCanvasVersionsInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasVersion, error) {
	var versions []CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Order("revision DESC").
		Find(&versions).
		Error
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func ListCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	return ListCanvasVersionsInTransaction(database.Conn(), workflowID)
}

func FindCanvasDraftInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID) (*CanvasUserDraft, error) {
	var draft CanvasUserDraft
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("user_id = ?", userID).
		First(&draft).
		Error

	if err != nil {
		return nil, err
	}

	return &draft, nil
}

func nextCanvasRevisionInTransaction(tx *gorm.DB, workflowID uuid.UUID) (int, error) {
	var currentMax int
	err := tx.
		Model(&CanvasVersion{}).
		Select("COALESCE(MAX(revision), 0)").
		Where("workflow_id = ?", workflowID).
		Scan(&currentMax).
		Error

	if err != nil {
		return 0, err
	}

	return currentMax + 1, nil
}

func lockCanvasForVersioningInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", workflowID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func CreatePublishedCanvasVersionInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	ownerID *uuid.UUID,
	basedOnVersionID *uuid.UUID,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	nextRevision, err := nextCanvasRevisionInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:               uuid.New(),
		WorkflowID:       workflowID,
		Revision:         nextRevision,
		OwnerID:          ownerID,
		BasedOnVersionID: basedOnVersionID,
		IsPublished:      true,
		PublishedAt:      &now,
		Nodes:            datatypes.NewJSONSlice(nodes),
		Edges:            datatypes.NewJSONSlice(edges),
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}

	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now

	if err := tx.Save(canvas).Error; err != nil {
		return nil, err
	}

	return &version, nil
}

func SaveCanvasDraftInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	userID uuid.UUID,
	basedOnVersionID *uuid.UUID,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	if basedOnVersionID == nil {
		basedOnVersionID = canvas.LiveVersionID
	}

	nextRevision, err := nextCanvasRevisionInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:               uuid.New(),
		WorkflowID:       workflowID,
		Revision:         nextRevision,
		OwnerID:          &userID,
		BasedOnVersionID: basedOnVersionID,
		IsPublished:      false,
		Nodes:            datatypes.NewJSONSlice(nodes),
		Edges:            datatypes.NewJSONSlice(edges),
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}

	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}

	draft := &CanvasUserDraft{
		WorkflowID: workflowID,
		UserID:     userID,
		VersionID:  version.ID,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "workflow_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"version_id": version.ID,
			"updated_at": now,
		}),
	}).Create(draft).Error; err != nil {
		return nil, err
	}

	return &version, nil
}

func PublishCanvasDraftInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	userID uuid.UUID,
) (*CanvasVersion, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	draft, err := FindCanvasDraftInTransaction(tx, workflowID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCanvasDraftNotFound
		}
		return nil, err
	}

	version, err := FindCanvasVersionInTransaction(tx, workflowID, draft.VersionID)
	if err != nil {
		return nil, err
	}

	if version.BasedOnVersionID != nil {
		if canvas.LiveVersionID == nil || *canvas.LiveVersionID != *version.BasedOnVersionID {
			return nil, ErrCanvasVersionConflict
		}
	}

	now := time.Now()
	version.IsPublished = true
	version.PublishedAt = &now
	version.UpdatedAt = &now

	if err := tx.Save(version).Error; err != nil {
		return nil, err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now

	if err := tx.Save(canvas).Error; err != nil {
		return nil, err
	}

	if err := tx.Delete(&CanvasUserDraft{}, "workflow_id = ? AND user_id = ?", workflowID, userID).Error; err != nil {
		return nil, err
	}

	return version, nil
}

func FindLiveCanvasVersionInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*CanvasVersion, error) {
	canvas, err := FindCanvasWithoutOrgScopeInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	return FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
}

func FindLiveCanvasVersionByCanvasInTransaction(tx *gorm.DB, canvas *Canvas) (*CanvasVersion, error) {
	if canvas.LiveVersionID == nil {
		return nil, gorm.ErrRecordNotFound
	}

	return FindCanvasVersionInTransaction(tx, canvas.ID, *canvas.LiveVersionID)
}

func FindLiveCanvasSpecInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]Node, []Edge, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, workflowID)
	if err != nil {
		return nil, nil, err
	}

	nodes := append([]Node(nil), version.Nodes...)
	edges := append([]Edge(nil), version.Edges...)
	return nodes, edges, nil
}

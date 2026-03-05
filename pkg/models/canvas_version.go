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

type CanvasVersion struct {
	ID          uuid.UUID
	WorkflowID  uuid.UUID
	OwnerID     *uuid.UUID
	IsPublished bool
	PublishedAt *time.Time
	Nodes       datatypes.JSONSlice[Node]
	Edges       datatypes.JSONSlice[Edge]
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
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
		Order("created_at DESC").
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

func ListPublishedCanvasVersionsInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	query := tx.
		Where("workflow_id = ?", workflowID).
		Where("is_published = ?", true).
		Order("published_at DESC, created_at DESC")

	if before != nil {
		query = query.Where("published_at < ?", *before)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var versions []CanvasVersion
	if err := query.Find(&versions).Error; err != nil {
		return nil, err
	}

	return versions, nil
}

func CountPublishedCanvasVersionsInTransaction(tx *gorm.DB, workflowID uuid.UUID) (int64, error) {
	var count int64
	err := tx.
		Model(&CanvasVersion{}).
		Where("workflow_id = ?", workflowID).
		Where("is_published = ?", true).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
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

func FindCanvasDraftByVersionInTransaction(tx *gorm.DB, workflowID, userID, versionID uuid.UUID) (*CanvasUserDraft, error) {
	var draft CanvasUserDraft
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("user_id = ?", userID).
		Where("version_id = ?", versionID).
		First(&draft).
		Error

	if err != nil {
		return nil, err
	}

	return &draft, nil
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
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		OwnerID:     ownerID,
		IsPublished: true,
		PublishedAt: &now,
		Nodes:       datatypes.NewJSONSlice(nodes),
		Edges:       datatypes.NewJSONSlice(edges),
		CreatedAt:   &now,
		UpdatedAt:   &now,
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
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	_, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		OwnerID:     &userID,
		IsPublished: false,
		Nodes:       datatypes.NewJSONSlice(nodes),
		Edges:       datatypes.NewJSONSlice(edges),
		CreatedAt:   &now,
		UpdatedAt:   &now,
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

func CreateOrResetCanvasDraftInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	userID uuid.UUID,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	if canvas.LiveVersionID == nil {
		return nil, gorm.ErrRecordNotFound
	}

	now := time.Now()
	draft, draftErr := FindCanvasDraftInTransaction(tx, workflowID, userID)
	if draftErr == nil {
		version, versionErr := FindCanvasVersionInTransaction(tx, workflowID, draft.VersionID)
		if versionErr != nil {
			return nil, versionErr
		}

		version.OwnerID = &userID
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.IsPublished = false
		version.PublishedAt = nil
		version.UpdatedAt = &now

		if err := tx.Save(version).Error; err != nil {
			return nil, err
		}

		draft.VersionID = version.ID
		draft.UpdatedAt = &now
		if err := tx.Save(draft).Error; err != nil {
			return nil, err
		}

		return version, nil
	}
	if !errors.Is(draftErr, gorm.ErrRecordNotFound) {
		return nil, draftErr
	}

	version := CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		OwnerID:     &userID,
		IsPublished: false,
		Nodes:       datatypes.NewJSONSlice(nodes),
		Edges:       datatypes.NewJSONSlice(edges),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}

	draft = &CanvasUserDraft{
		WorkflowID: workflowID,
		UserID:     userID,
		VersionID:  version.ID,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	if err := tx.Create(draft).Error; err != nil {
		return nil, err
	}

	return &version, nil
}

func CreateCanvasSnapshotVersionInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	ownerID uuid.UUID,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	if _, err := lockCanvasForVersioningInTransaction(tx, workflowID); err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		OwnerID:     &ownerID,
		IsPublished: false,
		Nodes:       datatypes.NewJSONSlice(nodes),
		Edges:       datatypes.NewJSONSlice(edges),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	if err := tx.Create(&version).Error; err != nil {
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

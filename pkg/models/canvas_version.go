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

const (
	CanvasVersionStateDraft     = "draft"
	CanvasVersionStatePublished = "published"
	CanvasVersionStateSnapshot  = "snapshot"
)

type CanvasVersion struct {
	ID                      uuid.UUID
	WorkflowID              uuid.UUID
	OwnerID                 *uuid.UUID
	State                   string
	Name                    string
	Description             string
	ChangeManagementEnabled bool
	ChangeRequestApprovers  datatypes.JSONSlice[CanvasChangeRequestApprover]
	PublishedAt             *time.Time
	Nodes                   datatypes.JSONSlice[Node]
	Edges                   datatypes.JSONSlice[Edge]
	CreatedAt               *time.Time
	UpdatedAt               *time.Time
}

func (c *CanvasVersion) TableName() string {
	return "workflow_versions"
}

func (c *CanvasVersion) EffectiveChangeRequestApprovers() []CanvasChangeRequestApprover {
	if c == nil || len(c.ChangeRequestApprovers) == 0 {
		return DefaultCanvasChangeRequestApprovers()
	}

	approvers := make([]CanvasChangeRequestApprover, len(c.ChangeRequestApprovers))
	copy(approvers, c.ChangeRequestApprovers)
	return approvers
}

func (c *CanvasVersion) BeforeCreate(_ *gorm.DB) error {
	c.ensureDefaultChangeRequestApprovers()
	return nil
}

func (c *CanvasVersion) BeforeSave(_ *gorm.DB) error {
	c.ensureDefaultChangeRequestApprovers()
	return nil
}

func (c *CanvasVersion) ensureDefaultChangeRequestApprovers() {
	if len(c.ChangeRequestApprovers) > 0 {
		return
	}

	c.ChangeRequestApprovers = datatypes.NewJSONSlice(DefaultCanvasChangeRequestApprovers())
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

func FindCanvasVersionForUpdateInTransaction(tx *gorm.DB, workflowID, versionID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("workflow_id = ?", workflowID).
		Where("id = ?", versionID).
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return &version, nil
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
		Where("state = ?", CanvasVersionStatePublished).
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
		Where("state = ?", CanvasVersionStatePublished).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func FindCanvasDraftInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("owner_id = ?", userID).
		Where("state = ?", CanvasVersionStateDraft).
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return &version, nil
}

func FindCanvasDraftByVersionInTransaction(tx *gorm.DB, workflowID, userID, versionID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("owner_id = ?", userID).
		Where("id = ?", versionID).
		Where("state = ?", CanvasVersionStateDraft).
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return &version, nil
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

func PromoteToLiveInTransaction(tx *gorm.DB, version *CanvasVersion, nodes []Node, edges []Edge) error {
	canvas, err := lockCanvasForVersioningInTransaction(tx, version.WorkflowID)
	if err != nil {
		return err
	}

	now := time.Now()
	version.State = CanvasVersionStatePublished
	version.PublishedAt = &now
	version.UpdatedAt = &now
	version.Nodes = datatypes.NewJSONSlice(nodes)
	version.Edges = datatypes.NewJSONSlice(edges)
	if err := tx.Save(version).Error; err != nil {
		return err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now
	return tx.Save(canvas).Error
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
	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	// Reuse existing draft if one already exists for this user+canvas.
	existing, findErr := FindCanvasDraftInTransaction(tx, workflowID, userID)
	if findErr == nil {
		existing.Nodes = datatypes.NewJSONSlice(nodes)
		existing.Edges = datatypes.NewJSONSlice(edges)
		existing.UpdatedAt = &now
		if err := tx.Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}

	version := CanvasVersion{
		ID:                      uuid.New(),
		WorkflowID:              workflowID,
		OwnerID:                 &userID,
		State:                   CanvasVersionStateDraft,
		Name:                    liveVersion.Name,
		Description:             liveVersion.Description,
		ChangeManagementEnabled: liveVersion.ChangeManagementEnabled,
		ChangeRequestApprovers:  datatypes.NewJSONSlice(liveVersion.EffectiveChangeRequestApprovers()),
		Nodes:                   datatypes.NewJSONSlice(nodes),
		Edges:                   datatypes.NewJSONSlice(edges),
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}

	return &version, nil
}

func CreateCanvasSnapshotVersionInTransaction(
	tx *gorm.DB,
	sourceVersion *CanvasVersion,
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
		ID:                      uuid.New(),
		WorkflowID:              workflowID,
		OwnerID:                 &ownerID,
		State:                   CanvasVersionStateSnapshot,
		Name:                    sourceVersion.Name,
		Description:             sourceVersion.Description,
		ChangeManagementEnabled: sourceVersion.ChangeManagementEnabled,
		ChangeRequestApprovers:  datatypes.NewJSONSlice(sourceVersion.EffectiveChangeRequestApprovers()),
		Nodes:                   datatypes.NewJSONSlice(nodes),
		Edges:                   datatypes.NewJSONSlice(edges),
		CreatedAt:               &now,
		UpdatedAt:               &now,
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

	version, err := FindCanvasDraftInTransaction(tx, workflowID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCanvasDraftNotFound
		}
		return nil, err
	}

	now := time.Now()
	version.State = CanvasVersionStatePublished
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

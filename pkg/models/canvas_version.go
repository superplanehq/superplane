package models

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CanvasVersion struct {
	ID            uuid.UUID
	WorkflowID    uuid.UUID
	OwnerID       *uuid.UUID
	CommitMessage string
	Nodes         datatypes.JSONSlice[Node]
	Edges         datatypes.JSONSlice[Edge]
	ConsolePanels datatypes.JSONType[[]ConsolePanel]
	ConsoleLayout datatypes.JSONType[[]ConsoleLayoutItem]
	CommitSHA     string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func (c *CanvasVersion) TableName() string {
	return "workflow_versions"
}

func FindCanvasVersion(workflowID, versionID uuid.UUID) (*CanvasVersion, error) {
	return FindCanvasVersionInTransaction(database.Conn(), workflowID, versionID)
}

func FindVersionByCommitSHA(workflowID uuid.UUID, commitSHA string) (*CanvasVersion, error) {
	return FindVersionByCommitSHAInTransaction(database.Conn(), workflowID, commitSHA)
}

func ListCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	return ListCanvasVersionsInTransaction(database.Conn(), workflowID)
}

func FindLatestCanvasVersion(workflowID uuid.UUID) (*CanvasVersion, error) {
	return FindLatestCanvasVersionInTransaction(database.Conn(), workflowID)
}

func FindLiveCanvasVersion(workflowID uuid.UUID) (*CanvasVersion, error) {
	return FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
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

func FindVersionByCommitSHAInTransaction(tx *gorm.DB, workflowID uuid.UUID, commitSHA string) (*CanvasVersion, error) {
	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("commit_sha = ?", commitSHA).
		First(&version).
		Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func FindCanvasVersionsByIDs(workflowID uuid.UUID, versionIDs []uuid.UUID) (map[uuid.UUID]*CanvasVersion, error) {
	return FindCanvasVersionsByIDsInTransaction(database.Conn(), workflowID, versionIDs)
}

func FindCanvasVersionsByIDsInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	versionIDs []uuid.UUID,
) (map[uuid.UUID]*CanvasVersion, error) {
	result := make(map[uuid.UUID]*CanvasVersion)
	if len(versionIDs) == 0 {
		return result, nil
	}

	var versions []CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("id IN ?", versionIDs).
		Find(&versions).
		Error
	if err != nil {
		return nil, err
	}

	for i := range versions {
		result[versions[i].ID] = &versions[i]
	}

	return result, nil
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

func ListCanvasVersionHistoryInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	query := tx.
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC, id DESC")

	if before != nil {
		query = query.Where("created_at < ?", *before)
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

func CountCanvasVersionsInTransaction(tx *gorm.DB, workflowID uuid.UUID) (int64, error) {
	var count int64
	err := tx.
		Model(&CanvasVersion{}).
		Where("workflow_id = ?", workflowID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func FindLatestCanvasVersionInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC, id DESC").
		First(&version).
		Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func CreateCommitVersionWithSpecInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	userID uuid.UUID,
	commitMessage string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return nil, err
	}

	if nodes == nil {
		nodes = slices.Clone(liveVersion.Nodes)
	}
	if edges == nil {
		edges = slices.Clone(liveVersion.Edges)
	}

	now := time.Now()
	version := CanvasVersion{
		ID:            uuid.New(),
		WorkflowID:    canvasID,
		OwnerID:       &userID,
		CommitMessage: strings.TrimSpace(commitMessage),
		Nodes:         datatypes.NewJSONSlice(nodes),
		Edges:         datatypes.NewJSONSlice(edges),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	copyVersionConsoleFields(liveVersion, &version)

	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}

	return &version, nil
}

func PromoteToLiveInTransaction(tx *gorm.DB, version *CanvasVersion, nodes []Node, edges []Edge) error {
	canvas, err := lockCanvasForVersioningInTransaction(tx, version.WorkflowID)
	if err != nil {
		return err
	}

	now := time.Now()
	version.Nodes = datatypes.NewJSONSlice(nodes)
	version.Edges = datatypes.NewJSONSlice(edges)
	version.UpdatedAt = &now
	if err := tx.Save(version).Error; err != nil {
		return err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now
	return MapCanvasNameUniqueConstraintError(tx.
		Model(&Canvas{}).
		Where("id = ?", canvas.ID).
		Updates(map[string]any{
			"live_version_id": version.ID,
			"updated_at":      now,
		}).
		Error)
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

func IsLiveCanvasVersion(tx *gorm.DB, canvas *Canvas, version *CanvasVersion) bool {
	if canvas == nil || canvas.LiveVersionID == nil || version == nil {
		return false
	}
	return *canvas.LiveVersionID == version.ID
}

type LiveCanvasSpec struct {
	Nodes []Node
	Edges []Edge
}

type liveCanvasSpecRow struct {
	WorkflowID uuid.UUID `gorm:"column:workflow_id"`
	Nodes      datatypes.JSONSlice[Node]
	Edges      datatypes.JSONSlice[Edge]
}

func FindLiveCanvasSpecsByCanvasIDs(tx *gorm.DB, canvasIDs []uuid.UUID) (map[uuid.UUID]LiveCanvasSpec, error) {
	specs := make(map[uuid.UUID]LiveCanvasSpec, len(canvasIDs))
	if len(canvasIDs) == 0 {
		return specs, nil
	}

	var rows []liveCanvasSpecRow
	err := tx.
		Table("workflows").
		Select("workflows.id AS workflow_id", "live_version.nodes", "live_version.edges").
		Joins("JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id").
		Where("workflows.id IN ?", canvasIDs).
		Where("workflows.live_version_id IS NOT NULL").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		specs[row.WorkflowID] = LiveCanvasSpec{
			Nodes: row.Nodes,
			Edges: row.Edges,
		}
	}

	return specs, nil
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

func SaveCanvasVersionInTransaction(tx *gorm.DB, version *CanvasVersion) error {
	if version == nil {
		return errors.New("version is required")
	}
	return tx.Save(version).Error
}

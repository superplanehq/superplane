package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"
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

	MaterializationStatusPending = "pending"
	MaterializationStatusReady   = "ready"
	MaterializationStatusError   = "error"
	MaterializationStatusDeleted = "deleted"

	CanvasGitBranchMain = "main"
)

type CanvasVersion struct {
	ID                    uuid.UUID
	WorkflowID            uuid.UUID
	OwnerID               *uuid.UUID
	State                 string
	PublishedAt           *time.Time
	Nodes                 datatypes.JSONSlice[Node]
	Edges                 datatypes.JSONSlice[Edge]
	ConsolePanels         datatypes.JSONType[[]ConsolePanel]
	ConsoleLayout         datatypes.JSONType[[]ConsoleLayoutItem]
	DisplayName           string
	CommitSHA             string
	GitBranch             string
	MaterializationStatus string
	MaterializationError  string
	CreatedAt             *time.Time
	UpdatedAt             *time.Time
}

func (c *CanvasVersion) TableName() string {
	return "workflow_versions"
}

func IsUserOwnedDraftVersion(version *CanvasVersion, userID uuid.UUID) bool {
	if version == nil {
		return false
	}
	if version.State != CanvasVersionStateDraft {
		return false
	}
	if version.OwnerID == nil || *version.OwnerID != userID {
		return false
	}
	return true
}

func IsRegisteredDraftVersion(version *CanvasVersion) bool {
	return version != nil &&
		version.State == CanvasVersionStateDraft &&
		version.GitBranch != "" &&
		version.GitBranch != CanvasGitBranchMain
}

func FindCanvasVersion(workflowID, versionID uuid.UUID) (*CanvasVersion, error) {
	return FindCanvasVersionInTransaction(database.Conn(), workflowID, versionID)
}

func FindVersionByCommitSHA(workflowID uuid.UUID, commitSHA string) (*CanvasVersion, error) {
	return FindVersionByCommitSHAInTransaction(database.Conn(), workflowID, commitSHA)
}

func FindDraftVersionByBranch(canvasID uuid.UUID, branchName string) (*CanvasVersion, error) {
	return FindDraftVersionByBranchInTransaction(database.Conn(), canvasID, branchName)
}

func ListAllDraftBranchVersionsForCanvas(canvasID uuid.UUID) ([]CanvasVersion, error) {
	return ListAllDraftBranchVersionsForCanvasInTransaction(database.Conn(), canvasID)
}

func DeleteDraftVersionByBranch(canvasID uuid.UUID, branchName string) error {
	return DeleteDraftVersionByBranchInTransaction(database.Conn(), canvasID, branchName)
}

func UpsertMaterializedVersion(version *CanvasVersion) error {
	return UpsertMaterializedVersionInTransaction(database.Conn(), version)
}

func ListVersionsForBranch(workflowID uuid.UUID, gitBranch string) ([]CanvasVersion, error) {
	return ListVersionsForBranchInTransaction(database.Conn(), workflowID, gitBranch)
}

func ListCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	return ListCanvasVersionsInTransaction(database.Conn(), workflowID)
}

func ListDraftCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	var versions []CanvasVersion
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("state = ?", CanvasVersionStateDraft).
		Order("created_at DESC").
		Find(&versions).
		Error
	return versions, err
}

func FindLatestPublishedCanvasVersion(workflowID uuid.UUID) (*CanvasVersion, error) {
	var version CanvasVersion
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("state = ?", CanvasVersionStatePublished).
		Order("published_at DESC").
		First(&version).
		Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func CreateDraftBranchFromLive(
	canvasID uuid.UUID,
	userID uuid.UUID,
	displayName string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	var draft *CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		created, createErr := CreateDraftBranchFromLiveInTransaction(tx, canvasID, userID, displayName, nodes, edges)
		draft = created
		return createErr
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
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

func FindDraftVersionByBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) (*CanvasVersion, error) {
	var version CanvasVersion
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("state = ?", CanvasVersionStateDraft).
		Where("git_branch = ?", branchName).
		First(&version).
		Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func ListAllDraftBranchVersionsForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasVersion, error) {
	var versions []CanvasVersion
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("state = ?", CanvasVersionStateDraft).
		Where("git_branch <> ?", "").
		Order("updated_at DESC, created_at DESC").
		Find(&versions).
		Error
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func DeleteDraftVersionByBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) error {
	return tx.
		Where("workflow_id = ?", canvasID).
		Where("state = ?", CanvasVersionStateDraft).
		Where("git_branch = ?", branchName).
		Delete(&CanvasVersion{}).
		Error
}

func UpsertMaterializedVersionInTransaction(tx *gorm.DB, version *CanvasVersion) error {
	if version == nil {
		return gorm.ErrInvalidData
	}

	now := time.Now()
	if version.CreatedAt == nil {
		version.CreatedAt = &now
	}
	version.UpdatedAt = &now

	var existing *CanvasVersion
	var err error

	if version.State == CanvasVersionStateDraft && IsRegisteredDraftVersion(version) {
		existing, err = FindDraftVersionByBranchInTransaction(tx, version.WorkflowID, version.GitBranch)
	} else if strings.TrimSpace(version.CommitSHA) != "" {
		existing, err = FindVersionByCommitSHAInTransaction(tx, version.WorkflowID, version.CommitSHA)
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if existing != nil {
		version.ID = existing.ID
		if version.GitBranch == "" {
			version.GitBranch = existing.GitBranch
		}
		if version.DisplayName == "" {
			version.DisplayName = existing.DisplayName
		}
		if version.OwnerID == nil {
			version.OwnerID = existing.OwnerID
		}
		return tx.Save(version).Error
	}

	if version.ID == uuid.Nil {
		version.ID = uuid.New()
	}

	return tx.Create(version).Error
}

func ListVersionsForBranchInTransaction(tx *gorm.DB, workflowID uuid.UUID, gitBranch string) ([]CanvasVersion, error) {
	var versions []CanvasVersion
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("git_branch = ?", gitBranch).
		Order("updated_at DESC, created_at DESC").
		Find(&versions).
		Error
	if err != nil {
		return nil, err
	}

	return versions, nil
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

func ListDraftBranchesForCanvasInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	ownerID uuid.UUID,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	query := tx.
		Where("workflow_id = ?", canvasID).
		Where("owner_id = ?", ownerID).
		Where("state = ?", CanvasVersionStateDraft).
		Where("git_branch <> ?", "").
		Order("updated_at DESC, created_at DESC, id DESC")

	if before != nil {
		query = query.Where("updated_at < ?", *before)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var versions []CanvasVersion
	err := query.Find(&versions).Error
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func CountDraftBranchesForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID, ownerID uuid.UUID) (int64, error) {
	var count int64
	err := tx.
		Model(&CanvasVersion{}).
		Where("workflow_id = ?", canvasID).
		Where("owner_id = ?", ownerID).
		Where("state = ?", CanvasVersionStateDraft).
		Where("git_branch <> ?", "").
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
		Order("updated_at DESC, created_at DESC, id DESC").
		First(&version).
		Error

	if err != nil {
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
	version.State = CanvasVersionStatePublished
	version.PublishedAt = &now
	version.UpdatedAt = &now
	version.GitBranch = CanvasGitBranchMain
	version.DisplayName = ""
	version.Nodes = datatypes.NewJSONSlice(nodes)
	version.Edges = datatypes.NewJSONSlice(edges)
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

// NextDraftDisplayNameInTransaction assigns a canvas-wide monotonic display
// label so deleted draft numbers are never reused on the same app.
func NextDraftDisplayNameInTransaction(tx *gorm.DB, canvasID uuid.UUID) (string, error) {
	canvas, err := lockCanvasForVersioningInTransaction(tx, canvasID)
	if err != nil {
		return "", err
	}

	number := canvas.NextDraftDisplayNumber
	if number < 1 {
		number = 1
	}

	if err := tx.
		Model(&Canvas{}).
		Where("id = ?", canvasID).
		Update("next_draft_display_number", number+1).
		Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("Draft #%d", number), nil
}

func CreateDraftBranchFromLiveInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	userID uuid.UUID,
	displayName string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	if _, err := lockCanvasForVersioningInTransaction(tx, canvasID); err != nil {
		return nil, err
	}

	now := time.Now()
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

	if displayName == "" {
		displayName, err = NextDraftDisplayNameInTransaction(tx, canvasID)
		if err != nil {
			return nil, err
		}
	}

	branchName := newDraftBranchName()
	version := CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  canvasID,
		OwnerID:     &userID,
		State:       CanvasVersionStateDraft,
		Nodes:       datatypes.NewJSONSlice(nodes),
		Edges:       datatypes.NewJSONSlice(edges),
		GitBranch:   branchName,
		DisplayName: displayName,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	copyVersionConsoleFields(liveVersion, &version)

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
		ID:         uuid.New(),
		WorkflowID: workflowID,
		OwnerID:    &ownerID,
		State:      CanvasVersionStateSnapshot,
		Nodes:      datatypes.NewJSONSlice(nodes),
		Edges:      datatypes.NewJSONSlice(edges),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	copyVersionConsoleFields(sourceVersion, &version)

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
	version.GitBranch = CanvasGitBranchMain
	version.DisplayName = ""

	if err := tx.Save(version).Error; err != nil {
		return nil, err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now

	if err := tx.
		Model(&Canvas{}).
		Where("id = ?", canvas.ID).
		Updates(map[string]any{
			"live_version_id": version.ID,
			"updated_at":      now,
		}).
		Error; err != nil {
		return nil, MapCanvasNameUniqueConstraintError(err)
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

const canvasDraftBranchNamePrefix = "drafts/"

func newDraftBranchName() string {
	return canvasDraftBranchNamePrefix + uuid.New().String()
}

func lockCanvasForVersioningInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		// This locks workflows directly, so select only columns that physically
		// exist on workflows; metadata fields are projected from live versions.
		Select(
			"id",
			"organization_id",
			"live_version_id",
			"folder_id",
			"name",
			"description",
			"next_draft_display_number",
			"created_by",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		Where("id = ?", workflowID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

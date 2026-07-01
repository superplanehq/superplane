package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasDraftNotFound = errors.New("canvas draft not found")

type CanvasVersion struct {
	ID            uuid.UUID
	WorkflowID    uuid.UUID
	BranchID      uuid.UUID `gorm:"type:uuid;not null"`
	OwnerID       *uuid.UUID
	Nodes         datatypes.JSONSlice[Node]
	Edges         datatypes.JSONSlice[Edge]
	ConsolePanels datatypes.JSONType[[]ConsolePanel]
	ConsoleLayout datatypes.JSONType[[]ConsoleLayoutItem]
	GitBranch     string
	CommitSHA     string
	CommitMessage string
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

func ListVersionsForBranch(workflowID uuid.UUID, gitBranch string) ([]CanvasVersion, error) {
	return ListVersionsForBranchInTransaction(database.Conn(), workflowID, gitBranch)
}

func ListCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	return ListCanvasVersionsInTransaction(database.Conn(), workflowID)
}

func FindLatestPublishedCanvasVersion(workflowID uuid.UUID) (*CanvasVersion, error) {
	return FindBranchHeadVersion(workflowID, CanvasGitBranchMain)
}

func FindBranchHeadVersion(workflowID uuid.UUID, branchName string) (*CanvasVersion, error) {
	var version *CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		branch, branchErr := FindWorkflowBranch(tx, workflowID, branchName)
		if branchErr != nil {
			return branchErr
		}
		if branch.HeadVersionID == nil {
			return gorm.ErrRecordNotFound
		}
		found, versionErr := FindCanvasVersionInTransaction(tx, workflowID, *branch.HeadVersionID)
		version = found
		return versionErr
	})
	if err != nil {
		return nil, err
	}
	return version, nil
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

func ListVersionsForBranchInTransaction(tx *gorm.DB, workflowID uuid.UUID, gitBranch string) ([]CanvasVersion, error) {
	branch, err := FindWorkflowBranch(tx, workflowID, gitBranch)
	if err != nil {
		return nil, err
	}

	var versions []CanvasVersion
	err = tx.
		Where("branch_id = ?", branch.ID).
		Order("created_at DESC, id DESC").
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
		Order("created_at DESC, id DESC").
		Find(&versions).
		Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func ListBranchCommitsInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	branchName string,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	branch, err := FindWorkflowBranch(tx, workflowID, branchName)
	if err != nil {
		return nil, err
	}

	query := tx.
		Where("branch_id = ?", branch.ID).
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

func CountBranchCommitsInTransaction(tx *gorm.DB, workflowID uuid.UUID, branchName string) (int64, error) {
	branch, err := FindWorkflowBranch(tx, workflowID, branchName)
	if err != nil {
		return 0, err
	}

	var count int64
	err = tx.
		Model(&CanvasVersion{}).
		Where("branch_id = ?", branch.ID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListPublishedCanvasVersionsInTransaction lists commits on main (legacy name for callers).
func ListPublishedCanvasVersionsInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	return ListBranchCommitsInTransaction(tx, workflowID, CanvasGitBranchMain, limit, before)
}

func CountPublishedCanvasVersionsInTransaction(tx *gorm.DB, workflowID uuid.UUID) (int64, error) {
	return CountBranchCommitsInTransaction(tx, workflowID, CanvasGitBranchMain)
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

func CreateInitialCommitInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	ownerID uuid.UUID,
	branchName string,
	commitMessage string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, *WorkflowBranch, error) {
	branch, err := CreateWorkflowBranch(tx, canvasID, branchName, nil)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:            uuid.New(),
		WorkflowID:    canvasID,
		BranchID:      branch.ID,
		OwnerID:       &ownerID,
		Nodes:         datatypes.NewJSONSlice(nodes),
		Edges:         datatypes.NewJSONSlice(edges),
		GitBranch:     branchName,
		CommitMessage: commitMessage,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := tx.Create(&version).Error; err != nil {
		return nil, nil, err
	}

	if err := UpdateWorkflowBranchHead(tx, branch.ID, version.ID); err != nil {
		return nil, nil, err
	}

	return &version, branch, nil
}

type CreateCommitInput struct {
	WorkflowID    uuid.UUID
	BranchName    string
	OwnerID       uuid.UUID
	CommitMessage string
	Nodes         []Node
	Edges         []Edge
	ConsolePanels []ConsolePanel
	ConsoleLayout []ConsoleLayoutItem
	CommitSHA     string
}

// CreateCommitOnBranch appends an immutable commit and advances the branch head.
func CreateCommitOnBranch(tx *gorm.DB, input CreateCommitInput) (*CanvasVersion, error) {
	if _, err := lockCanvasForVersioningInTransaction(tx, input.WorkflowID); err != nil {
		return nil, err
	}

	foundBranch, err := FindWorkflowBranch(tx, input.WorkflowID, input.BranchName)
	if err != nil {
		return nil, err
	}
	lockedBranch, err := lockWorkflowBranchForUpdate(tx, foundBranch.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	version := CanvasVersion{
		ID:            uuid.New(),
		WorkflowID:    input.WorkflowID,
		BranchID:      lockedBranch.ID,
		OwnerID:       &input.OwnerID,
		Nodes:         datatypes.NewJSONSlice(input.Nodes),
		Edges:         datatypes.NewJSONSlice(input.Edges),
		GitBranch:     input.BranchName,
		CommitMessage: strings.TrimSpace(input.CommitMessage),
		CommitSHA:     strings.TrimSpace(input.CommitSHA),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if len(input.ConsolePanels) > 0 {
		version.ConsolePanels = datatypes.NewJSONType(input.ConsolePanels)
	} else if lockedBranch.HeadVersionID != nil {
		head, headErr := FindCanvasVersionInTransaction(tx, input.WorkflowID, *lockedBranch.HeadVersionID)
		if headErr == nil {
			copyVersionConsoleFields(head, &version)
		}
	}
	if len(input.ConsoleLayout) > 0 {
		version.ConsoleLayout = datatypes.NewJSONType(input.ConsoleLayout)
	}

	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}
	if err := UpdateWorkflowBranchHead(tx, lockedBranch.ID, version.ID); err != nil {
		return nil, err
	}

	if input.BranchName == CanvasGitBranchMain {
		canvas, canvasErr := lockCanvasForVersioningInTransaction(tx, input.WorkflowID)
		if canvasErr != nil {
			return nil, canvasErr
		}
		canvas.LiveVersionID = &version.ID
		canvas.UpdatedAt = &now
		if err := tx.Model(&Canvas{}).
			Where("id = ?", canvas.ID).
			Updates(map[string]any{
				"live_version_id": version.ID,
				"updated_at":      now,
			}).Error; err != nil {
			return nil, err
		}
	}

	return &version, nil
}

func UpdateCanvasVersionCommitSHA(workflowID, versionID uuid.UUID, commitSHA string) error {
	return UpdateCanvasVersionCommitSHAInTransaction(database.Conn(), workflowID, versionID, commitSHA)
}

func UpdateCanvasVersionCommitSHAInTransaction(tx *gorm.DB, workflowID, versionID uuid.UUID, commitSHA string) error {
	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA == "" {
		return nil
	}

	return tx.Model(&CanvasVersion{}).
		Where("workflow_id = ? AND id = ?", workflowID, versionID).
		Update("commit_sha", commitSHA).Error
}

func CreateBranchFromHeadInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	sourceBranchName string,
	newBranchName string,
	ownerID uuid.UUID,
) (*WorkflowBranch, *CanvasVersion, error) {
	source, err := FindWorkflowBranch(tx, workflowID, sourceBranchName)
	if err != nil {
		return nil, nil, err
	}
	if source.HeadVersionID == nil {
		return nil, nil, gorm.ErrRecordNotFound
	}

	head, err := FindCanvasVersionInTransaction(tx, workflowID, *source.HeadVersionID)
	if err != nil {
		return nil, nil, err
	}

	branch, err := CreateWorkflowBranch(tx, workflowID, newBranchName, source.HeadVersionID)
	if err != nil {
		return nil, nil, err
	}
	return branch, head, nil
}

func PromoteToLiveInTransaction(tx *gorm.DB, version *CanvasVersion, nodes []Node, edges []Edge) error {
	now := time.Now()
	version.Nodes = datatypes.NewJSONSlice(nodes)
	version.Edges = datatypes.NewJSONSlice(edges)
	version.UpdatedAt = &now
	if err := tx.Save(version).Error; err != nil {
		return err
	}

	canvas, err := lockCanvasForVersioningInTransaction(tx, version.WorkflowID)
	if err != nil {
		return err
	}
	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = &now
	return tx.Model(&Canvas{}).
		Where("id = ?", canvas.ID).
		Updates(map[string]any{
			"live_version_id": version.ID,
			"updated_at":      now,
		}).
		Error
}

func CreateCanvasSnapshotVersionInTransaction(
	tx *gorm.DB,
	sourceVersion *CanvasVersion,
	workflowID uuid.UUID,
	ownerID uuid.UUID,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	now := time.Now()
	version := CanvasVersion{
		ID:            uuid.New(),
		WorkflowID:    workflowID,
		BranchID:      sourceVersion.BranchID,
		OwnerID:       &ownerID,
		Nodes:         datatypes.NewJSONSlice(nodes),
		Edges:         datatypes.NewJSONSlice(edges),
		GitBranch:     sourceVersion.GitBranch,
		CommitMessage: fmt.Sprintf("Snapshot of %s", sourceVersion.ID),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	copyVersionConsoleFields(sourceVersion, &version)
	if err := tx.Create(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func lockCanvasForVersioningInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select(
			"id",
			"organization_id",
			"live_version_id",
			"folder_id",
			"name",
			"description",
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

func sanitizeBranchName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	return strings.ToLower(name)
}

func NewBranchNameFromMessage(message string) string {
	base := sanitizeBranchName(message)
	if base == "" {
		return "branch-" + uuid.New().String()[:8]
	}
	if len(base) > 48 {
		base = base[:48]
	}
	return base + "-" + uuid.New().String()[:8]
}

// Legacy stubs — removed draft-per-user model; kept so older call sites compile during POC.
func ListDraftCanvasVersions(workflowID uuid.UUID) ([]CanvasVersion, error) {
	branches, err := ListWorkflowBranchesConn(workflowID)
	if err != nil {
		return nil, err
	}
	var versions []CanvasVersion
	for _, branch := range branches {
		if branch.Name == CanvasGitBranchMain || branch.HeadVersionID == nil {
			continue
		}
		version, versionErr := FindCanvasVersion(workflowID, *branch.HeadVersionID)
		if versionErr != nil {
			continue
		}
		versions = append(versions, *version)
	}
	return versions, nil
}

func IsRegisteredDraftVersion(version *CanvasVersion) bool {
	return version != nil && version.GitBranch != "" && version.GitBranch != CanvasGitBranchMain
}

func IsUserOwnedDraftVersion(version *CanvasVersion, userID uuid.UUID) bool {
	return version != nil && version.OwnerID != nil && *version.OwnerID == userID
}

func CreateDraftBranchFromLive(
	canvasID uuid.UUID,
	userID uuid.UUID,
	displayName string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	var result *CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		version, createErr := CreateDraftBranchFromLiveInTransaction(tx, canvasID, userID, displayName, nodes, edges)
		result = version
		return createErr
	})
	return result, err
}

func CreateDraftBranchFromLiveInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	userID uuid.UUID,
	displayName string,
	nodes []Node,
	edges []Edge,
) (*CanvasVersion, error) {
	branchName := displayName
	if branchName == "" {
		branchName = NewBranchNameFromMessage("draft")
	} else {
		branchName = sanitizeBranchName(branchName)
	}

	branch, head, err := CreateBranchFromHeadInTransaction(tx, canvasID, CanvasGitBranchMain, branchName, userID)
	if err != nil {
		return nil, err
	}
	_ = branch

	if nodes != nil {
		head.Nodes = datatypes.NewJSONSlice(nodes)
	}
	if edges != nil {
		head.Edges = datatypes.NewJSONSlice(edges)
	}
	return head, nil
}

func FindDraftVersionByBranch(canvasID uuid.UUID, branchName string) (*CanvasVersion, error) {
	return FindBranchHeadVersion(canvasID, branchName)
}

func FindDraftVersionByBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) (*CanvasVersion, error) {
	branch, err := FindWorkflowBranch(tx, canvasID, branchName)
	if err != nil {
		return nil, err
	}
	if branch.HeadVersionID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return FindCanvasVersionInTransaction(tx, canvasID, *branch.HeadVersionID)
}

func ListAllDraftBranchVersionsForCanvas(canvasID uuid.UUID) ([]CanvasVersion, error) {
	return ListDraftCanvasVersions(canvasID)
}

func ListAllDraftBranchVersionsForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasVersion, error) {
	branches, err := ListWorkflowBranches(tx, canvasID)
	if err != nil {
		return nil, err
	}
	var versions []CanvasVersion
	for _, branch := range branches {
		if branch.Name == CanvasGitBranchMain || branch.HeadVersionID == nil {
			continue
		}
		version, versionErr := FindCanvasVersionInTransaction(tx, canvasID, *branch.HeadVersionID)
		if versionErr != nil {
			continue
		}
		versions = append(versions, *version)
	}
	return versions, nil
}

func DeleteDraftVersionByBranch(canvasID uuid.UUID, branchName string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return DeleteDraftVersionByBranchInTransaction(tx, canvasID, branchName)
	})
}

func DeleteDraftVersionByBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) error {
	return DeleteWorkflowBranch(tx, canvasID, branchName)
}

func UpsertMaterializedVersion(version *CanvasVersion) error {
	return UpsertMaterializedVersionInTransaction(database.Conn(), version)
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
	if version.ID == uuid.Nil {
		version.ID = uuid.New()
		return tx.Create(version).Error
	}
	return tx.Save(version).Error
}

func ListDraftBranchesForCanvasInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	ownerID uuid.UUID,
	limit int,
	before *time.Time,
) ([]CanvasVersion, error) {
	_ = ownerID
	_ = limit
	_ = before
	return ListAllDraftBranchVersionsForCanvasInTransaction(tx, canvasID)
}

func CountDraftBranchesForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID, ownerID uuid.UUID) (int64, error) {
	_ = ownerID
	branches, err := ListWorkflowBranches(tx, canvasID)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, branch := range branches {
		if branch.Name != CanvasGitBranchMain {
			count++
		}
	}
	return count, nil
}

func FindCanvasDraftInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID) (*CanvasVersion, error) {
	versions, err := ListAllDraftBranchVersionsForCanvasInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		if versions[i].OwnerID != nil && *versions[i].OwnerID == userID {
			return &versions[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func PublishCanvasDraftInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	userID uuid.UUID,
) (*CanvasVersion, error) {
	_ = userID
	mainHead, err := FindBranchHeadVersionInTransaction(tx, workflowID, CanvasGitBranchMain)
	if err != nil {
		return nil, err
	}
	return mainHead, nil
}

func FindBranchHeadVersionInTransaction(tx *gorm.DB, workflowID uuid.UUID, branchName string) (*CanvasVersion, error) {
	branch, err := FindWorkflowBranch(tx, workflowID, branchName)
	if err != nil {
		return nil, err
	}
	if branch.HeadVersionID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return FindCanvasVersionInTransaction(tx, workflowID, *branch.HeadVersionID)
}

func NextDraftDisplayNameInTransaction(tx *gorm.DB, canvasID uuid.UUID) (string, error) {
	_ = tx
	_ = canvasID
	return NewBranchNameFromMessage("draft"), nil
}

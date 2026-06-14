package canvases

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// loadOwnedDraftVersion resolves the canvas and draft version for a staging
// write/commit/discard, enforcing that the caller owns the registered draft.
func loadOwnedDraftVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*models.Canvas, *models.CanvasVersion, uuid.UUID, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, uuid.Nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, uuid.Nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, nil, uuid.Nil, status.Errorf(codes.Internal, "failed to load canvas: %v", err)
	}

	version, err := models.FindCanvasVersion(canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, uuid.Nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, nil, uuid.Nil, status.Errorf(codes.Internal, "failed to load version: %v", err)
	}

	userUUID := uuid.MustParse(userID)
	if err := ensureVersionIsOwnedRegisteredDraft(userUUID, version); err != nil {
		return nil, nil, uuid.Nil, err
	}

	return canvas, version, userUUID, nil
}

// ensureVersionIsOwnedRegisteredDraft guards draft mutations: the caller must own
// the version, and the version must be an editable, registered draft branch (not a
// published or snapshot version).
func ensureVersionIsOwnedRegisteredDraft(userID uuid.UUID, version *models.CanvasVersion) error {
	if version.OwnerID == nil || *version.OwnerID != userID {
		return status.Error(codes.PermissionDenied, "version owner mismatch")
	}

	if version.State == models.CanvasVersionStatePublished {
		return status.Error(codes.FailedPrecondition, "published versions are immutable")
	}

	if version.State != models.CanvasVersionStateDraft {
		return status.Error(codes.FailedPrecondition, "version is not your editable draft")
	}

	if !models.IsRegisteredDraftVersion(version) {
		return status.Error(codes.FailedPrecondition, "version is not a registered draft branch")
	}

	return nil
}

// ensureStagedReadAllowed restricts effective staged reads to the draft owner.
// Staging rows can outlive a draft's edit session; without this check any org
// reader could pass ?stage=true and read someone else's uncommitted work.
func ensureStagedReadAllowed(ctx context.Context, version *models.CanvasVersion) error {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	return ensureVersionIsOwnedRegisteredDraft(uuid.MustParse(userID), version)
}

// buildStagingSummary reports the uncommitted spec edits held in workflow_staged_files
// for a draft version so the UI can drive its orange/blue indicators.
func buildStagingSummary(versionID uuid.UUID, rows []models.WorkflowStaging) *pb.StagingSummary {
	state := &pb.StagingSummary{}
	if len(rows) == 0 {
		return state
	}

	paths := make([]string, 0, len(rows))
	var baseHeadSHA string
	for _, row := range rows {
		paths = append(paths, row.Path)
		if baseHeadSHA == "" && strings.TrimSpace(row.BaseHeadSHA) != "" {
			baseHeadSHA = strings.TrimSpace(row.BaseHeadSHA)
		}
	}

	base := versionID.String()
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = &base
	if baseHeadSHA != "" {
		state.BaseHeadSha = &baseHeadSHA
	}
	return state
}

// effectiveSpecYAML returns the YAML the UI should edit for a draft path:
// staged content when present, the materialized version row otherwise, and an
// empty string when the path is staged as deleted.
func effectiveSpecYAML(
	canvas *models.Canvas,
	version *models.CanvasVersion,
	organizationID string,
	rows []models.WorkflowStaging,
	path string,
) (string, error) {
	for _, row := range rows {
		if row.Path != path {
			continue
		}
		if row.Deleted {
			return "", nil
		}
		return row.Content, nil
	}

	switch path {
	case CanvasYAMLRepositoryPath:
		return canvasYAMLFromVersion(canvas, version, organizationID)
	case ConsoleYAMLRepositoryPath:
		return consoleYAMLFromVersion(version)
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported repository spec file %q", path)
	}
}

// StageRepositorySpecFileOperations stores repository file edits in
// workflow_staged_files verbatim, leaving workflow_versions untouched until commit.
// Both spec files (canvas.yaml/console.yaml, committed into the version row) and
// arbitrary repository files (committed to git) are accepted; the path kind is
// resolved at commit time.
func StageRepositorySpecFileOperations(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.StagingSummary, error) {
	canvas, version, userUUID, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	baseHeadSHA := strings.TrimSpace(version.CommitSHA)
	organizationUUID := canvas.OrganizationID

	for _, operation := range operations {
		if operation == nil {
			continue
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		if normalized == "" {
			return nil, status.Error(codes.InvalidArgument, "file path is required")
		}
		if normalized == gitprovider.ReservedSuperPlanePath ||
			strings.HasPrefix(normalized, gitprovider.ReservedSuperPlanePath+"/") {
			return nil, status.Errorf(codes.InvalidArgument, "path %q is reserved for SuperPlane", operation.GetPath())
		}

		if operation.GetDelete() {
			if err := models.MarkWorkflowStagingPathDeleted(version.ID, organizationUUID, normalized, baseHeadSHA, &userUUID); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to stage deletion of %q: %v", normalized, err)
			}
			continue
		}

		if _, err := models.UpsertWorkflowStagingPath(
			version.ID,
			organizationUUID,
			normalized,
			string(operation.GetContent()),
			baseHeadSHA,
			&userUUID,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to stage %q: %v", normalized, err)
		}
	}

	rows, err := models.ListWorkflowStaging(version.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load staging: %v", err)
	}

	return buildStagingSummary(version.ID, rows), nil
}

// stagingSummaryForVersion returns the StagingSummary for a version, used by reads
// to drive draft indicators without a dedicated list endpoint.
func stagingSummaryForVersion(versionID uuid.UUID) (*pb.StagingSummary, []models.WorkflowStaging, error) {
	rows, err := models.ListWorkflowStaging(versionID)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load staging: %v", err)
	}
	return buildStagingSummary(versionID, rows), rows, nil
}

// ReadStagedRepositoryFile returns the staged content for an arbitrary (non-spec)
// repository file on a draft version. found=false means there is no staging row
// for the path, so the caller should fall back to the committed git content.
// deleted=true means the file is staged for deletion.
func ReadStagedRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
) (content string, found bool, deleted bool, err error) {
	if strings.TrimSpace(versionID) == "" {
		return "", false, false, nil
	}

	_, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return "", false, false, err
	}

	if err := ensureStagedReadAllowed(ctx, version); err != nil {
		return "", false, false, err
	}

	_, rows, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return "", false, false, err
	}

	normalized := normalizeRepositoryFilePath(path)
	for _, row := range rows {
		if row.Path != normalized {
			continue
		}
		if row.Deleted {
			return "", true, true, nil
		}
		return row.Content, true, false, nil
	}

	return "", false, false, nil
}

// ReadCommittedRepositoryFile reads the committed content of an arbitrary
// (non-spec) repository file. When versionID refers to a draft version the file
// is read from that draft's branch — where Files-tab edits are committed —
// rather than the repository's default branch, so a just-committed edit is
// reflected back to the editor. With an empty versionID the default branch
// (live) content is returned.
func ReadCommittedRepositoryFile(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
) (string, error) {
	ref := ""
	if strings.TrimSpace(versionID) != "" {
		_, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
		if err != nil {
			return "", err
		}
		if version.BranchName != nil && strings.TrimSpace(*version.BranchName) != "" {
			ref = strings.TrimSpace(*version.BranchName)
		}
	}

	return readGitFile(ctx, gitProvider, repoID, path, ref)
}

// HasUnpublishedRepositoryFileChanges reports whether a draft version's branch has
// committed changes to arbitrary (non-spec) repository files relative to the live
// (main) branch. It powers the publish-readiness and unpublished-change indicators
// for files such as README.md, which the spec-based graph/console version diffs do
// not cover.
//
// Spec files (canvas.yaml/console.yaml) are intentionally excluded: their
// publishable differences are already surfaced by the graph/console diffs, and
// comparing raw YAML bytes here would report false positives for semantically equal
// but reformatted specs. A version without a draft branch (e.g. the live version)
// has nothing unpublished and returns false.
func HasUnpublishedRepositoryFileChanges(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	organizationID string,
	canvasID string,
	versionID string,
) (bool, error) {
	if strings.TrimSpace(versionID) == "" {
		return false, nil
	}

	_, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return false, err
	}

	if version.BranchName == nil {
		return false, nil
	}
	draftBranch := strings.TrimSpace(*version.BranchName)
	if draftBranch == "" || draftBranch == models.CanvasGitBranchMain {
		return false, nil
	}

	// A draft branch that has not advanced past main (identical head) cannot hold
	// any committed file changes, so skip the file walk entirely.
	draftHead, err := gitProvider.Head(ctx, repoID, draftBranch)
	if err != nil {
		return false, err
	}
	liveHead, err := gitProvider.Head(ctx, repoID, models.CanvasGitBranchMain)
	if err != nil {
		return false, err
	}
	if draftHead == liveHead {
		return false, nil
	}

	draftPaths, err := nonSpecRepositoryFilePaths(ctx, gitProvider, repoID, draftBranch)
	if err != nil {
		return false, err
	}
	livePaths, err := nonSpecRepositoryFilePaths(ctx, gitProvider, repoID, models.CanvasGitBranchMain)
	if err != nil {
		return false, err
	}

	// Added or removed non-spec files are unpublished changes outright.
	if !stringSetsEqual(draftPaths, livePaths) {
		return true, nil
	}

	// Same file set: an unpublished change exists only if some file's content
	// differs between the draft branch and live.
	for path := range draftPaths {
		draftContent, err := readGitFile(ctx, gitProvider, repoID, path, draftBranch)
		if err != nil {
			return false, err
		}
		liveContent, err := readGitFile(ctx, gitProvider, repoID, path, models.CanvasGitBranchMain)
		if err != nil {
			return false, err
		}
		if draftContent != liveContent {
			return true, nil
		}
	}

	return false, nil
}

// nonSpecRepositoryFilePaths returns the set of repository file paths on a ref,
// excluding the spec files (canvas.yaml/console.yaml) and SuperPlane-reserved paths.
func nonSpecRepositoryFilePaths(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	ref string,
) (map[string]struct{}, error) {
	files, err := gitProvider.ListFiles(ctx, repoID, ref)
	if err != nil {
		return nil, err
	}

	result := make(map[string]struct{}, len(files))
	for _, file := range files {
		normalized := normalizeRepositoryFilePath(file)
		if normalized == "" || IsRepositorySpecFilePath(normalized) {
			continue
		}
		if normalized == gitprovider.ReservedSuperPlanePath ||
			strings.HasPrefix(normalized, gitprovider.ReservedSuperPlanePath+"/") {
			continue
		}
		result[normalized] = struct{}{}
	}

	return result, nil
}

func stringSetsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

func readGitFile(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	path string,
	ref string,
) (string, error) {
	reader, err := gitProvider.GetFile(ctx, repoID, path, ref)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

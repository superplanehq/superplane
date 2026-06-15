package canvases

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/changerequests"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PublishCanvasChangeRequest merges an approved change request into the canvas's
// live state. The merge result (live ⊕ change request) is serialized and
// committed to the git main branch, keeping git as the source of truth, then the
// live projection is materialized from that commit.
func PublishCanvasChangeRequest(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	canvasID string,
	changeRequestID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)
	actorUserUUID := uuid.MustParse(userID)

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	changeManagementEnabled, modeErr := isChangeManagementEnabledForCanvas(canvas)
	if modeErr != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load change management setting: %v", modeErr)
	}
	if !changeManagementEnabled {
		return nil, nil, status.Error(codes.FailedPrecondition, "change management is disabled for this canvas")
	}

	var version *models.CanvasVersion
	var request *models.CanvasChangeRequest
	var mergedNodes []models.Node
	var mergedEdges []models.Edge
	var liveCommitSHA string

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasForUpdate, canvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if canvasErr != nil {
			return canvasErr
		}

		request, err = models.FindCanvasChangeRequestInTransaction(tx, canvasUUID, changeRequestUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "change request not found")
			}
			return err
		}

		if request.Status == models.CanvasChangeRequestStatusPublished {
			return status.Error(codes.FailedPrecondition, "change request was already merged")
		}
		if request.Status == models.CanvasChangeRequestStatusRejected {
			return status.Error(codes.FailedPrecondition, "change request is rejected")
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		nameErr := ensureCanvasNameAvailableInTransaction(tx, organizationUUID, canvasUUID, version.Name)
		if errors.Is(nameErr, models.ErrCanvasNameAlreadyExists) {
			return status.Error(codes.AlreadyExists, "Canvas with the same name already exists")
		}
		if nameErr != nil {
			return nameErr
		}

		if err := changerequests.RefreshCanvasChangeRequestDiffInTransaction(tx, canvasForUpdate, version, request); err != nil {
			return err
		}
		if len(request.ConflictingNodeIDs) > 0 {
			return status.Error(codes.FailedPrecondition, "change request has conflicts")
		}
		if !isOpenCanvasChangeRequestStatus(request.Status) {
			return status.Error(codes.FailedPrecondition, "change request cannot be published in its current status")
		}
		approvals, approvalsErr := models.ListCanvasChangeRequestApprovalsInTransaction(tx, canvasUUID, request.ID)
		if approvalsErr != nil {
			return approvalsErr
		}
		if publishCheckErr := ensureCanvasChangeRequestReadyToPublish(canvasForUpdate, approvals); publishCheckErr != nil {
			return publishCheckErr
		}

		baseNodes, baseEdges, liveNodes, liveEdges, resolveErr := changerequests.ResolveCanvasChangeRequestBaseAndLiveInTransaction(
			tx,
			canvasForUpdate,
			request,
		)
		if resolveErr != nil {
			return resolveErr
		}

		mergedNodes, mergedEdges = mergeCanvasVersionIntoLive(
			baseNodes,
			baseEdges,
			liveNodes,
			liveEdges,
			version.Nodes,
			version.Edges,
			request.ChangedNodeIDs,
		)

		liveVersion, liveErr := models.FindLiveCanvasVersionInTransaction(tx, canvasUUID)
		if liveErr != nil {
			return liveErr
		}
		liveCommitSHA = liveVersion.CommitSHA

		canvas = canvasForUpdate
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, actions.ToStatus(err)
	}

	mergeSHA, err := commitChangeRequestMergeToMain(
		ctx,
		gitProv,
		organizationID,
		organizationUUID,
		canvasUUID,
		actorUserUUID,
		version,
		mergedNodes,
		mergedEdges,
		liveCommitSHA,
	)
	if err != nil {
		return nil, nil, err
	}

	var liveVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		// Register a pending published version holding the merged result. It is
		// NOT promoted to live here: the worker's live materialization diffs the
		// current (old) live against the git snapshot to reconcile workflow_nodes,
		// then reuses this row (matched by commit SHA) and promotes it.
		now := time.Now()
		liveVersion = &models.CanvasVersion{
			WorkflowID:              canvasUUID,
			State:                   models.CanvasVersionStatePublished,
			Name:                    version.Name,
			Description:             version.Description,
			ChangeManagementEnabled: version.ChangeManagementEnabled,
			ChangeRequestApprovers:  version.ChangeRequestApprovers,
			Nodes:                   datatypes.NewJSONSlice(mergedNodes),
			Edges:                   datatypes.NewJSONSlice(mergedEdges),
			ConsolePanels:           version.ConsolePanels,
			ConsoleLayout:           version.ConsoleLayout,
			CommitSHA:               mergeSHA,
			GitBranch:               models.CanvasGitBranchMain,
			MaterializationStatus:   models.MaterializationStatusPending,
			PublishedAt:             &now,
			CreatedAt:               &now,
			UpdatedAt:               &now,
		}
		if upsertErr := models.UpsertMaterializedVersionInTransaction(tx, liveVersion); upsertErr != nil {
			return upsertErr
		}

		request, err = models.FindCanvasChangeRequestInTransaction(tx, canvasUUID, changeRequestUUID)
		if err != nil {
			return err
		}

		request.Status = models.CanvasChangeRequestStatusPublished
		request.PublishedAt = &now
		request.UpdatedAt = &now
		return tx.Save(request).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, status.Errorf(codes.Internal, "failed to publish change request: %v", err)
	}

	// Worker-authoritative materialization: main now points at the merge commit,
	// so the worker (inline in tests) materializes the live projection and promotes
	// the published version registered above.
	if err := materialize.RequestBranchMaterialization(ctx, canvasUUID, models.CanvasGitBranchMain, mergeSHA, &actorUserUUID); err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to request live materialization: %v", err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if liveVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), liveVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
		}
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return request, version, nil
}

// commitChangeRequestMergeToMain serializes the merged change request result and
// commits it to the canvas's git main branch, returning the resulting commit SHA.
func commitChangeRequestMergeToMain(
	ctx context.Context,
	gitProv gitprovider.Provider,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	actorUserUUID uuid.UUID,
	version *models.CanvasVersion,
	mergedNodes []models.Node,
	mergedEdges []models.Edge,
	liveCommitSHA string,
) (string, error) {
	if gitProv == nil {
		return "", status.Error(codes.FailedPrecondition, "git provider is not configured")
	}

	repository, err := models.FindRepository(organizationUUID, canvasUUID)
	if err != nil {
		return "", status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, actorUserUUID.String())
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	canvas := materialize.CanvasYAMLFromVersion(version)
	canvas.Spec.Nodes = mergedNodes
	canvas.Spec.Edges = mergedEdges
	canvasYAML, err := materialize.BuildCanvasYAMLFromCanvas(canvas)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to build canvas yaml: %v", err)
	}

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(version)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to build console yaml: %v", err)
	}

	operations := []gitprovider.FileOperation{
		{
			Path:      CanvasYAMLRepositoryPath,
			Content:   bytes.NewReader(canvasYAML),
			SizeBytes: int64(len(canvasYAML)),
		},
		{
			Path:      ConsoleYAMLRepositoryPath,
			Content:   bytes.NewReader(consoleYAML),
			SizeBytes: int64(len(consoleYAML)),
		},
	}

	mergeSHA, err := gitProv.Commit(ctx, repository.RepoID, gitprovider.CommitOptions{
		Branch:          models.CanvasGitBranchMain,
		BaseBranch:      models.CanvasGitBranchMain,
		ExpectedHeadSHA: liveCommitSHA,
		Message:         "Publish change request",
		Operations:      operations,
		Author: gitprovider.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to commit change request merge: %v", err)
	}

	return mergeSHA, nil
}

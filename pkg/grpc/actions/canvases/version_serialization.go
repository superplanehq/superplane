package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func canvasMetadataFromCanvas(canvas *models.Canvas) (name, description string) {
	if canvas == nil {
		return "", ""
	}

	return canvas.Name, canvas.Description
}

func SerializeCanvasVersion(version *models.CanvasVersion, organizationID string, ownersByID map[string]*models.User) (*pb.CanvasVersion, error) {
	var author *pb.UserRef
	if version.OwnerID != nil {
		author = canvasVersionOwnerRef(organizationID, version.OwnerID.String(), ownersByID)
	}

	metadata := &pb.CanvasVersion_Metadata{
		Id:            version.ID.String(),
		CanvasId:      version.WorkflowID.String(),
		Author:        author,
		CommitMessage: version.CommitMessage,
	}

	if version.CreatedAt != nil {
		metadata.CreatedAt = timestamppb.New(*version.CreatedAt)
	}
	if version.UpdatedAt != nil {
		metadata.UpdatedAt = timestamppb.New(*version.UpdatedAt)
	}

	spec, err := SerializeCanvasSpecFromVersion(version)
	if err != nil {
		return nil, err
	}

	return &pb.CanvasVersion{
		Metadata: metadata,
		Spec:     spec,
	}, nil
}

func canvasVersionOwnerRef(organizationID, ownerID string, ownersByID map[string]*models.User) *pb.UserRef {
	ownerName := ""
	if ownersByID != nil {
		if user := ownersByID[ownerID]; user != nil {
			ownerName = user.Name
		}
	} else if user, err := models.FindMaybeDeletedUserByID(organizationID, ownerID); err == nil && user != nil {
		ownerName = user.Name
	}

	return &pb.UserRef{Id: ownerID, Name: ownerName}
}

func ownersByIDForCanvasVersions(ctx context.Context, orgID string, versions []models.CanvasVersion) (map[string]*models.User, error) {
	db := database.DB(ctx)
	idSet := make(map[string]struct{})
	for i := range versions {
		if versions[i].OwnerID != nil {
			idSet[versions[i].OwnerID.String()] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return map[string]*models.User{}, nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID, ids)
	if err != nil {
		return nil, err
	}

	ownersByID := make(map[string]*models.User, len(users))
	for i := range users {
		ownersByID[users[i].ID.String()] = &users[i]
	}

	return ownersByID, nil
}

func serializeCanvasVersions(ctx context.Context, versions []models.CanvasVersion, organizationID string) ([]*pb.CanvasVersion, error) {
	var err error
	ctx, done := telemetry.Span(ctx, "canvases.serialize_versions")
	defer done(&err)

	ownersByID, ownersErr := ownersByIDForCanvasVersions(ctx, organizationID, versions)
	if ownersErr != nil {
		ownersByID = nil
	}

	protoVersions := make([]*pb.CanvasVersion, 0, len(versions))
	for i := range versions {
		v, err := SerializeCanvasVersion(&versions[i], organizationID, ownersByID)
		if err != nil {
			return nil, err
		}

		protoVersions = append(protoVersions, v)
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("canvases.version_count", len(versions)))
	}

	return protoVersions, nil
}

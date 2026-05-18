package apps

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeApp(app *models.App) *pb.App {
	metadata := &pb.App_Metadata{
		Id:             app.ID.String(),
		OrganizationId: app.OrganizationID.String(),
		DisplayName:    app.DisplayName,
		Slug:           app.Slug,
		Description:    app.Description,
	}

	if app.CanvasID != nil {
		metadata.CanvasId = app.CanvasID.String()
	}

	if app.CreatedBy != nil {
		metadata.CreatedById = app.CreatedBy.String()
	}

	if app.CreatedAt != nil {
		metadata.CreatedAt = timestamppb.New(*app.CreatedAt)
	}

	if app.UpdatedAt != nil {
		metadata.UpdatedAt = timestamppb.New(*app.UpdatedAt)
	}

	syncState := &pb.App_SyncState{
		Status:               app.SyncStatus,
		LiveCommitSha:        app.LiveCommitSha,
		DefaultBranch:        app.DefaultBranch,
		CodeStorageRemoteUrl: app.CodeStorageRemoteURL,
		CodeStorageRepoId:    app.CodeStorageRepoID,
	}

	if app.SyncError != nil {
		syncState.Error = *app.SyncError
	}

	if app.EditSessionBranch != nil {
		syncState.EditSessionBranch = *app.EditSessionBranch
	}

	return &pb.App{
		Metadata:  metadata,
		SyncState: syncState,
	}
}

func serializeApps(apps []models.App) []*pb.App {
	result := make([]*pb.App, 0, len(apps))
	for i := range apps {
		result = append(result, serializeApp(&apps[i]))
	}
	return result
}

func serializeAppDoc(doc *models.AppDoc) *pb.AppDoc {
	pbDoc := &pb.AppDoc{
		Id:      doc.ID.String(),
		AppId:   doc.AppID.String(),
		Path:    doc.Path,
		Content: doc.Content,
		Sha:     doc.Sha,
	}

	if doc.UpdatedAt != nil {
		pbDoc.UpdatedAt = timestamppb.New(*doc.UpdatedAt)
	}

	return pbDoc
}

func serializeAppDocs(docs []models.AppDoc) []*pb.AppDoc {
	result := make([]*pb.AppDoc, 0, len(docs))
	for i := range docs {
		result = append(result, serializeAppDoc(&docs[i]))
	}
	return result
}

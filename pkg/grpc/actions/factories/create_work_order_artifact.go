package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateWorkOrderArtifact(
	ctx context.Context,
	organizationID string,
	req *pb.CreateWorkOrderArtifactRequest,
) (*pb.CreateWorkOrderArtifactResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	artifactType, ok := artifactTypeFromProto(req.GetType())
	if !ok {
		return nil, factoryErrorToStatus(invalidArgument("artifact type is required"), "failed to create work order artifact")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order artifact")
	}

	data := map[string]any{}
	if req.GetData() != nil {
		data = req.GetData().AsMap()
	}

	db := database.DB(ctx)
	factoryModel, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}
	factoryID := factoryModel.ID

	order, err := findWorkOrder(db, factoryModel, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}
	orderID := order.ID

	var artifact *models.FactoryWorkOrderArtifact
	err = db.Transaction(func(tx *gorm.DB) error {
		if artifactType == models.FactoryWorkOrderArtifactTypeFile {
			// The task lock stops two requests from attaching the same file.
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(order, "id = ?", order.ID).Error; err != nil {
				return err
			}
			fileData, err := uploadedFileArtifactData(tx, order, userID, data)
			if err != nil {
				return err
			}
			data = fileData
		}
		created, err := order.CreateArtifact(tx, models.FactoryWorkOrderArtifactParams{
			Type:      artifactType,
			Data:      data,
			CreatedBy: &userID,
		})
		artifact = created
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	if err := db.Preload("CreatedBy", models.PreloadRemovedMember).First(artifact, "id = ?", artifact.ID).Error; err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderArtifactAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	serialized, err := serializeArtifact(artifact)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	return &pb.CreateWorkOrderArtifactResponse{Artifact: serialized}, nil
}

// uploadedFileArtifactData turns data.fileId into the stored file fields.
// The file must be an artifact file that this user uploaded to this task.
func uploadedFileArtifactData(
	db *gorm.DB,
	order *models.FactoryWorkOrder,
	userID uuid.UUID,
	input map[string]any,
) (map[string]any, error) {
	rawID, _ := input["fileId"].(string)
	fileID, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, invalidArgument("file artifacts require data.fileId")
	}
	file, err := models.FindFile(db, fileID)
	if err != nil {
		return nil, invalidArgument("file not found")
	}
	if !isUploadedArtifactFile(file, order, userID) {
		return nil, invalidArgument("file is not an uploaded artifact file of this task")
	}
	attached, err := order.HasFileArtifact(db, file.ID)
	if err != nil {
		return nil, err
	}
	if attached {
		return nil, invalidArgument("file is already attached to this task")
	}
	publicURL, err := storedfiles.ArtifactPublicURL(file)
	if err != nil {
		return nil, err
	}
	title, _ := input["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		title = file.Filename
	}
	return map[string]any{
		"fileId":      file.ID.String(),
		"filename":    file.Filename,
		"contentType": file.ContentType,
		"sizeBytes":   file.SizeBytes,
		"title":       title,
		"url":         publicURL,
	}, nil
}

func isUploadedArtifactFile(file *models.File, order *models.FactoryWorkOrder, userID uuid.UUID) bool {
	return file.State == models.FileStateReady &&
		file.Purpose == models.FilePurposeArtifact &&
		file.Scope == blob.ScopeTask &&
		file.WorkOrderID != nil && *file.WorkOrderID == order.ID &&
		file.OrganizationID != nil && *file.OrganizationID == order.OrganizationID &&
		file.CreatedByID != nil && *file.CreatedByID == userID
}

package files

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/files"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func CreateFactoryFile(ctx context.Context, organizationID string, req *pb.CreateFactoryFileRequest) (*pb.CreateFactoryFileResponse, error) {
	createdByID, err := parseCreatedBy(ctx)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create workspace file")
	}

	db := database.DB(ctx)
	factory, err := resolveFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create workspace file")
	}

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: factory.OrganizationID,
		FactoryID:      factory.ID,
		Filename:       req.GetFilename(),
		ContentType:    req.GetContentType(),
		CreatedByID:    createdByID,
	})
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create workspace file")
	}

	return &pb.CreateFactoryFileResponse{File: serializeFile(file, "", storedfiles.ContentUploadURL(file.ID))}, nil
}

func ListFactoryFiles(ctx context.Context, organizationID string, req *pb.ListFactoryFilesRequest) (*pb.ListFactoryFilesResponse, error) {
	db := database.DB(ctx)
	factory, err := resolveFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list workspace files")
	}

	records, err := models.ListReadyWorkspaceFiles(db, factory.ID)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list workspace files")
	}

	serialized, err := serializeReadyFiles(ctx, records, blob.UIDownloadTTL)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list workspace files")
	}
	return &pb.ListFactoryFilesResponse{Files: serialized}, nil
}

func CreateWorkOrderFile(ctx context.Context, organizationID string, req *pb.CreateWorkOrderFileRequest) (*pb.CreateWorkOrderFileResponse, error) {
	createdByID, err := parseCreatedBy(ctx)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create task file")
	}

	db := database.DB(ctx)
	factory, err := resolveFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create task file")
	}
	order, err := resolveWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create task file")
	}

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: factory.OrganizationID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		Filename:       req.GetFilename(),
		ContentType:    req.GetContentType(),
		CreatedByID:    createdByID,
	})
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to create task file")
	}

	return &pb.CreateWorkOrderFileResponse{File: serializeFile(file, "", storedfiles.ContentUploadURL(file.ID))}, nil
}

func ListWorkOrderFiles(ctx context.Context, organizationID string, req *pb.ListWorkOrderFilesRequest) (*pb.ListWorkOrderFilesResponse, error) {
	db := database.DB(ctx)
	factory, err := resolveFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list task files")
	}
	order, err := resolveWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list task files")
	}

	records, err := models.ListReadyTaskFiles(db, order.ID)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list task files")
	}

	serialized, err := serializeReadyFiles(ctx, records, blob.UIDownloadTTL)
	if err != nil {
		return nil, fileErrorToStatus(err, "failed to list task files")
	}
	return &pb.ListWorkOrderFilesResponse{Files: serialized}, nil
}

func serializeReadyFiles(ctx context.Context, records []models.File, ttl time.Duration) ([]*pb.File, error) {
	provider := blob.Current()
	out := make([]*pb.File, 0, len(records))
	for i := range records {
		downloadURL, err := storedfiles.DownloadURL(ctx, provider, &records[i], ttl)
		if err != nil {
			return nil, err
		}
		out = append(out, serializeFile(&records[i], downloadURL, ""))
	}
	return out, nil
}

func serializeFile(file *models.File, downloadURL, uploadURL string) *pb.File {
	serialized := &pb.File{
		Id:          file.ID.String(),
		Filename:    file.Filename,
		ContentType: file.ContentType,
		SizeBytes:   file.SizeBytes,
		Scope:       file.Scope,
		State:       file.State,
		DownloadUrl: downloadURL,
		UploadUrl:   uploadURL,
		CreatedAt:   timestamppb.New(file.CreatedAt),
	}
	if file.Checksum != nil {
		serialized.Checksum = *file.Checksum
	}
	return serialized
}

func parseCreatedBy(ctx context.Context) (uuid.UUID, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, errUnauthenticated
	}
	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid user id")
	}
	return createdByID, nil
}

func resolveFactory(db *gorm.DB, organizationID, factoryRef string) (*models.Factory, error) {
	orgID, err := parseID(organizationID, "organization id")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(factoryRef) == "" {
		return nil, invalidArgument("invalid factory id")
	}
	return models.FindFactoryByRef(db, orgID, factoryRef)
}

func resolveWorkOrder(db *gorm.DB, factory *models.Factory, orderRef string) (*models.FactoryWorkOrder, error) {
	if strings.TrimSpace(orderRef) == "" {
		return nil, invalidArgument("invalid order id")
	}
	return factory.FindWorkOrderByRef(db, orderRef)
}

func parseID(value, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid " + name)
	}
	return id, nil
}

func fileErrorToStatus(err error, internalMessage string) error {
	switch {
	case errors.Is(err, errUnauthenticated):
		return grpcerrors.Unauthenticated(nil, "user not authenticated")
	case errors.Is(err, models.ErrFactoryNotFound), errors.Is(err, models.ErrFactoryWorkOrderNotFound), errors.Is(err, models.ErrFileNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "resource not found")
	case errors.Is(err, models.ErrFileContentType), errors.Is(err, models.ErrFileInvalid), errors.Is(err, models.ErrFileNotReady), errors.Is(err, models.ErrFileForeignReference):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFileQuotaExceeded):
		return grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, errInvalidArgument):
		return grpcerrors.InvalidArgument(err, err.Error())
	default:
		return grpcerrors.Internal(err, internalMessage)
	}
}

var (
	errInvalidArgument = errors.New("invalid argument")
	errUnauthenticated = errors.New("unauthenticated")
)

func invalidArgument(message string) error {
	return errors.Join(errInvalidArgument, errors.New(message))
}

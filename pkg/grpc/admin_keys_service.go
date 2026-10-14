package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AdminKeysService struct {
	pb.UnimplementedKeysServer
	store    *encryption.Store
	migrator *encryption.Migrator
}

func NewAdminKeysService(store *encryption.Store, migrator *encryption.Migrator) *AdminKeysService {
	return &AdminKeysService{store: store, migrator: migrator}
}

func (s *AdminKeysService) ListKeys(ctx context.Context, _ *pb.ListKeysRequest) (*pb.ListKeysResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	keys, active, err := s.store.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list encryption keys")
	}
	response := &pb.ListKeysResponse{Keys: make([]*pb.DataKey, 0, len(keys)), ActiveKeyId: active.String()}
	for _, key := range keys {
		response.Keys = append(response.Keys, &pb.DataKey{
			Id: key.ID.String(), MasterKeyRef: key.MasterKeyRef, CreatedAt: timestamppb.New(key.CreatedAt),
		})
	}
	return response, nil
}

func (s *AdminKeysService) RotateKey(ctx context.Context, _ *pb.RotateKeyRequest) (*pb.RotateKeyResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	id, err := s.store.Rotate(ctx)
	if errors.Is(err, encryption.ErrMigrationUnfinished) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not rotate encryption key")
	}
	return &pb.RotateKeyResponse{ActiveKeyId: id.String()}, nil
}

func (s *AdminKeysService) RewrapKeys(ctx context.Context, req *pb.RewrapKeysRequest) (*pb.RewrapKeysResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	if err := s.store.Rewrap(ctx, req.MasterKeyRef); errors.Is(err, encryption.ErrMasterKeyNotConfigured) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else if err != nil {
		return nil, status.Error(codes.Internal, "could not rewrap encryption keys")
	}
	return &pb.RewrapKeysResponse{}, nil
}

func (s *AdminKeysService) StartMigration(ctx context.Context, _ *pb.StartMigrationRequest) (*pb.MigrationResponse, error) {
	if s.migrator == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	job, err := s.migrator.Start(ctx)
	if errors.Is(err, encryption.ErrMigrationAlreadyRunning) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not start encryption migration")
	}
	return migrationResponse(job), nil
}

func (s *AdminKeysService) GetMigration(ctx context.Context, req *pb.GetMigrationRequest) (*pb.MigrationResponse, error) {
	if s.migrator == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid migration ID")
	}
	job, err := s.migrator.Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "encryption migration not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not load encryption migration")
	}
	return migrationResponse(job), nil
}

func (s *AdminKeysService) RetryMigration(ctx context.Context, req *pb.RetryMigrationRequest) (*pb.MigrationResponse, error) {
	if s.migrator == nil {
		return nil, status.Error(codes.Unavailable, "application encryption keys are not configured")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid migration ID")
	}
	job, err := s.migrator.Retry(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "encryption migration not found")
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return migrationResponse(job), nil
}

func migrationResponse(job *models.ApplicationEncryptionMigration) *pb.MigrationResponse {
	response := &pb.MigrationResponse{
		Id: job.ID.String(), TargetKeyId: job.TargetKeyID.String(), State: job.State,
		StorageLocation: job.StorageLocation, Pass: int32(job.Pass), ErrorMessage: job.ErrorMessage,
		CreatedAt: timestamppb.New(job.CreatedAt), UpdatedAt: timestamppb.New(job.UpdatedAt),
	}
	if job.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*job.CompletedAt)
	}
	return response
}

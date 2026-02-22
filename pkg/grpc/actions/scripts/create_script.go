package scripts

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func CreateScript(ctx context.Context, organizationID string, script *pb.Script) (*pb.CreateScriptResponse, error) {
	if script.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "script name is required")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	createdBy := uuid.MustParse(userID)
	orgID := uuid.MustParse(organizationID)
	now := time.Now()

	manifest := datatypes.JSON([]byte("{}"))
	if script.ManifestJson != "" {
		manifest = datatypes.JSON([]byte(script.ManifestJson))
	}

	scriptStatus := models.ScriptStatusDraft
	if script.Status != "" {
		scriptStatus = script.Status
	}

	model := &models.Script{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           script.Name,
		Label:          script.Label,
		Description:    script.Description,
		Source:         script.Source,
		Manifest:       manifest,
		Status:         scriptStatus,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := database.Conn().Create(model).Error
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return nil, status.Error(codes.InvalidArgument, "a script with this name already exists in this organization")
		}
		return nil, err
	}

	return &pb.CreateScriptResponse{
		Script: SerializeScript(model),
	}, nil
}

package apps

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// slugPattern restricts app_slug segments to lowercase alphanumeric + underscore.
var slugPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

func CreateApp(ctx context.Context, organizationID uuid.UUID, orgSlug, displayName, appSlug, description string) (*pb.CreateAppResponse, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, status.Error(codes.InvalidArgument, "display_name is required")
	}

	appSlug = strings.TrimSpace(appSlug)
	if appSlug == "" {
		return nil, status.Error(codes.InvalidArgument, "app_slug is required")
	}

	if !slugPattern.MatchString(appSlug) {
		return nil, status.Error(codes.InvalidArgument, "app_slug must only contain lowercase letters, digits, and underscores")
	}

	fullSlug := fmt.Sprintf("%s-%s", orgSlug, appSlug)

	taken, err := models.IsAppSlugTaken(fullSlug)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check slug availability: %v", err)
	}
	if taken {
		return nil, status.Errorf(codes.AlreadyExists, "app slug %q is already taken", fullSlug)
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	createdBy := uuid.MustParse(userID)
	now := time.Now()

	app := &models.App{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		DisplayName:    displayName,
		Slug:           fullSlug,
		Description:    description,
		DefaultBranch:  "main",
		SyncStatus:     models.AppSyncStatusOk,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateApp(tx, app)
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create app: %v", err)
	}

	return &pb.CreateAppResponse{
		App: serializeApp(app),
	}, nil
}

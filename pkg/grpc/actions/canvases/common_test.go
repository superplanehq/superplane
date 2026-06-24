package canvases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

const testWebhookBaseURL = "http://localhost:3000/api/v1"

func createDraftVersionID(ctx context.Context, t *testing.T, orgID, canvasID, displayName string) string {
	t.Helper()

	response, err := CreateCanvasVersion(ctx, orgID, canvasID, displayName)
	require.NoError(t, err)
	require.NotNil(t, response.GetVersion())
	require.NotNil(t, response.GetVersion().GetMetadata())

	versionID := strings.TrimSpace(response.GetVersion().GetMetadata().GetId())
	require.NotEmpty(t, versionID)

	return versionID
}

func createCanvasWithNoopNode(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasName string) string {
	t.Helper()

	createCanvasResponse, err := CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		testWebhookBaseURL,
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: canvasName},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:        "node-1",
						Name:      "Initial Name",
						Component: "noop",
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		nil,
	)

	require.NoError(t, err)
	return createCanvasResponse.Canvas.Metadata.Id
}

func createDraftVersion(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasID string, nodeName string) string {
	t.Helper()

	versionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvasID, "")

	_, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test Canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:        "node-1",
						Name:      nodeName,
						Component: "noop",
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)
	return versionID
}

func findRegisteredDraftBranch(t *testing.T, canvasID uuid.UUID, branchName string) *models.CanvasVersion {
	t.Helper()

	var version models.CanvasVersion
	err := database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("git_branch = ?", branchName).
		Where("state = ?", models.CanvasVersionStateDraft).
		First(&version).
		Error
	require.NoError(t, err)

	return &version
}

func findRegisteredDraftBranchErr(canvasID uuid.UUID, branchName string) error {
	var version models.CanvasVersion
	return database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("git_branch = ?", branchName).
		Where("state = ?", models.CanvasVersionStateDraft).
		First(&version).
		Error
}

func structFromAnyMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}

func TestMapCanvasNameUniqueConstraintError(t *testing.T) {
	t.Run("maps workflow name unique violation to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "workflows_organization_id_name_key",
		})

		assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, func() string {
			_, msg, ok := grpcerrors.HandlerStatus(err)
			if ok {
				return msg
			}
			return err.Error()
		}())
	})

	t.Run("maps model duplicate name error to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(models.ErrCanvasNameAlreadyExists)

		assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, func() string {
			_, msg, ok := grpcerrors.HandlerStatus(err)
			if ok {
				return msg
			}
			return err.Error()
		}())
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		original := errors.New("other error")

		err := mapCanvasNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})
}

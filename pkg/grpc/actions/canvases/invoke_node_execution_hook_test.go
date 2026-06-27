package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
)

func Test__InvokeNodeExecutionHook__Approve(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "approval-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Name:   nodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "approval"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"items": []any{
						map[string]any{"type": "anyone"},
					},
				}),
			},
		},
		nil,
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(
		t,
		canvas.ID,
		nodeID,
		rootEvent.ID,
		rootEvent.ID,
		map[string]any{
			"items": []any{
				map[string]any{"type": "anyone"},
			},
		},
	)

	// Seed pending approval metadata directly so the hook has a record to act on.
	require.NoError(t, database.Conn().Model(execution).Update("metadata", map[string]any{
		"result": "pending",
		"records": []any{
			map[string]any{
				"index": 0,
				"type":  "anyone",
				"state": "pending",
			},
		},
	}).Error)

	authedCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated context -> Unauthenticated", func(t *testing.T) {
		_, err := InvokeNodeExecutionHook(
			context.Background(),
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			execution.ID,
			"approve",
			map[string]any{"index": float64(0)},
		)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
	})

	t.Run("unknown canvas -> NotFound", func(t *testing.T) {
		_, err := InvokeNodeExecutionHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			uuid.New(),
			execution.ID,
			"approve",
			map[string]any{"index": float64(0)},
		)
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("unknown execution -> NotFound", func(t *testing.T) {
		_, err := InvokeNodeExecutionHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			uuid.New(),
			"approve",
			map[string]any{"index": float64(0)},
		)
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("unknown hook -> NotFound", func(t *testing.T) {
		_, err := InvokeNodeExecutionHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			execution.ID,
			"this-hook-does-not-exist",
			map[string]any{},
		)
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("missing required hook parameter -> InvalidArgument (not Internal)", func(t *testing.T) {
		// Regression: missing/invalid parameters previously fell through to the
		// gateway error sanitizer as a generic Internal error, which turned every
		// client-side validation failure into an HTTP 500 (visible in Sentry as
		// HTTP 500 /api/v1/canvases/{id}/executions/{id}/hooks/approve issues).
		_, err := InvokeNodeExecutionHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			execution.ID,
			"approve",
			map[string]any{},
		)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		// The sanitizer must also map this to InvalidArgument when the error
		// reaches the grpc-gateway, i.e. a 400 instead of a 500.
		assert.Equal(t, codes.InvalidArgument, statusCodeForSanitized(t, err))
	})

	t.Run("approving anyone record succeeds", func(t *testing.T) {
		_, err := InvokeNodeExecutionHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			execution.ID,
			"approve",
			map[string]any{"index": float64(0)},
		)
		require.NoError(t, err)
	})
}

// statusCodeForSanitized mirrors what the grpc-gateway error handler does for
// arbitrary handler errors. This lets us assert that user-facing 4xx errors
// remain 4xx after sanitization rather than being collapsed into Internal.
func statusCodeForSanitized(t *testing.T, err error) codes.Code {
	t.Helper()

	code, _, ok := grpcerrors.HandlerStatus(err)
	if ok {
		return code
	}
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}
	return codes.Unknown
}

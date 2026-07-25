package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func newTestCanvasService(r *support.ResourceRegistry) *CanvasService {
	return NewCanvasService(
		r.AuthService,
		r.Registry,
		r.Encryptor,
		r.GitProvider,
		"http://localhost:8000",
		nil,
	)
}

func canvasServiceContext(orgID, userID string) context.Context {
	ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, orgID)
	return authentication.SetUserIdInMetadata(ctx, userID)
}

func testCanvasNodes() []models.CanvasNode {
	return []models.CanvasNode{
		{
			NodeID: "node-1",
			Name:   "Node 1",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
		},
	}
}

type crossOrgCanvasCase struct {
	name     string
	call     func(context.Context, *CanvasService, string) error
	wantCode codes.Code
}

func TestCanvasService_rejectsCanvasFromOtherOrganization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ownCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, testCanvasNodes(), nil)

	otherOrg := support.CreateOrganization(t, r, r.User)
	otherCanvas, _ := support.CreateCanvas(t, otherOrg.ID, r.User, testCanvasNodes(), nil)

	ctx := canvasServiceContext(r.Organization.ID.String(), r.User.String())
	svc := newTestCanvasService(r)
	otherCanvasID := otherCanvas.ID.String()

	cases := []crossOrgCanvasCase{
		{
			name: "DescribeCanvas",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.DescribeCanvas(ctx, &pb.DescribeCanvasRequest{Id: canvasID})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ListNodeExecutions",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ListNodeExecutions(ctx, &pb.ListNodeExecutionsRequest{
					CanvasId: canvasID,
					NodeId:   "node-1",
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ListNodeEvents",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ListNodeEvents(ctx, &pb.ListNodeEventsRequest{
					CanvasId: canvasID,
					NodeId:   "node-1",
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ListNodeQueueItems",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ListNodeQueueItems(ctx, &pb.ListNodeQueueItemsRequest{
					CanvasId: canvasID,
					NodeId:   "node-1",
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "DeleteNodeQueueItem",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.DeleteNodeQueueItem(ctx, &pb.DeleteNodeQueueItemRequest{
					CanvasId: canvasID,
					NodeId:   "node-1",
					ItemId:   uuid.New().String(),
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ListRuns",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ListRuns(ctx, &pb.ListRunsRequest{CanvasId: canvasID})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "DescribeRun",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.DescribeRun(ctx, &pb.DescribeRunRequest{
					CanvasId: canvasID,
					RunId:    uuid.New().String(),
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ListEventExecutions",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ListEventExecutions(ctx, &pb.ListEventExecutionsRequest{
					CanvasId: canvasID,
					EventId:  uuid.New().String(),
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "ResolveExecutionErrors",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.ResolveExecutionErrors(ctx, &pb.ResolveExecutionErrorsRequest{
					CanvasId:     canvasID,
					ExecutionIds: []string{uuid.New().String()},
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "CancelRun",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.CancelRun(ctx, &pb.CancelRunRequest{
					CanvasId: canvasID,
					RunId:    uuid.New().String(),
				})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "CancelExecution",
			call: func(ctx context.Context, svc *CanvasService, canvasID string) error {
				_, err := svc.CancelExecution(ctx, &pb.CancelExecutionRequest{
					CanvasId:    canvasID,
					ExecutionId: uuid.New().String(),
				})
				return err
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(ctx, svc, otherCanvasID)
			require.Error(t, err)

			code, msg, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok, "expected handler status error")
			assert.Equal(t, tc.wantCode, code)
			if tc.wantCode == codes.NotFound {
				assert.Equal(t, "canvas not found", msg)
			}
		})
	}

	t.Run("own organization canvas is still accessible", func(t *testing.T) {
		response, err := svc.DescribeCanvas(ctx, &pb.DescribeCanvasRequest{Id: ownCanvas.ID.String()})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, ownCanvas.ID.String(), response.Canvas.Metadata.Id)
	})
}

func TestCanvasService_findCanvas_rejectsInvalidCanvasID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := canvasServiceContext(r.Organization.ID.String(), r.User.String())
	svc := newTestCanvasService(r)

	_, err := svc.DescribeCanvas(ctx, &pb.DescribeCanvasRequest{Id: "not-a-uuid"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
}

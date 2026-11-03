package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__AddWorkOrderChecks(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	addCheck := func(check *pb.AddWorkOrderChecksRequest_Check) (*pb.AddWorkOrderChecksResponse, error) {
		return AddWorkOrderChecks(ctx, r.Organization.ID.String(), &pb.AddWorkOrderChecksRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Checks:    []*pb.AddWorkOrderChecksRequest_Check{check},
		})
	}
	check := func(key string, score float64) *pb.AddWorkOrderChecksRequest_Check {
		return &pb.AddWorkOrderChecksRequest_Check{
			Key:      key,
			Name:     "Tests pass",
			Score:    score,
			MaxScore: 10,
			Level:    pb.WorkOrderCheck_LEVEL_POSITIVE,
			Reason:   "All unit tests pass.",
		}
	}

	t.Run("adds a check that shows on the task", func(t *testing.T) {
		resp, err := addCheck(check("tests-pass", 9))

		require.NoError(t, err)
		require.Len(t, resp.Checks, 1)
		assert.Equal(t, "tests-pass", resp.Checks[0].Key)
		assert.Equal(t, models.FactoryWorkOrderCheckSourceAgent, resp.Checks[0].Source)
		assert.Equal(t, pb.WorkOrderCheck_LEVEL_POSITIVE, resp.Checks[0].Level)
		assert.Equal(t, "All unit tests pass.", resp.Checks[0].Summary)

		described, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, described.Order.Checks, 1)
		assert.Equal(t, 9.0, described.Order.Checks[0].Score)
	})

	t.Run("rejects a check without a level", func(t *testing.T) {
		input := check("no-level", 1)
		input.Level = pb.WorkOrderCheck_LEVEL_UNSPECIFIED

		_, err := addCheck(input)

		assertStatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("rejects a reserved key", func(t *testing.T) {
		_, err := addCheck(check(models.PlanningClarityCheckKey, 1))

		assertStatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("does not replace an automation check", func(t *testing.T) {
		_, err := order.ReportCheck(db, models.FactoryWorkOrderCheckParams{Key: "coverage", Name: "Coverage", Score: 5, MaxScore: 5})
		require.NoError(t, err)

		_, err = addCheck(check("coverage", 1))

		assertStatusCode(t, err, codes.FailedPrecondition)
		checks, err := order.ListChecks(db)
		require.NoError(t, err)
		for _, existing := range checks {
			if existing.Key == "coverage" {
				assert.Equal(t, 5.0, existing.Score)
			}
		}
	})

	t.Run("rejects a closed task", func(t *testing.T) {
		closed, err := factoryModel.CreateWorkOrder(db, "Done", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = closed.Close(db, models.FactoryWorkOrderResultRejected, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderChecks(ctx, r.Organization.ID.String(), &pb.AddWorkOrderChecksRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   closed.ID.String(),
			Checks:    []*pb.AddWorkOrderChecksRequest_Check{check("tests-pass", 9)},
		})

		assertStatusCode(t, err, codes.Aborted)
	})
}

func Test__RemoveWorkOrderChecks(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.ReportAgentChecks(db, []models.FactoryWorkOrderCheckParams{{Key: "tests-pass", Name: "Tests pass", Score: 1, MaxScore: 5}})
	require.NoError(t, err)
	_, err = order.ReportCheck(db, models.FactoryWorkOrderCheckParams{Key: "coverage", Name: "Coverage", Score: 5, MaxScore: 5})
	require.NoError(t, err)

	resp, err := RemoveWorkOrderChecks(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderChecksRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		Keys:      []string{"tests-pass", "coverage"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"tests-pass"}, resp.Removed)
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "coverage", checks[0].Key)

	_, err = RemoveWorkOrderChecks(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderChecksRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok, "expected a gRPC status error, got %v", err)
	assert.Equal(t, want, code)
}

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
	"gorm.io/gorm"
)

type stubIntakeItemSource struct {
	items []IntakeItem
	err   error
}

func (s stubIntakeItemSource) Search(context.Context, string, int) ([]IntakeItem, error) {
	return s.items, s.err
}

func (s stubIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, errIntakeItemNotFound
}

func Test__SearchFactoryIntakeItems(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	item := IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: "Handle duplicate refunds",
		Body:  "Retrying a refund posts twice.",
		URL:   "https://github.com/acme/payments/issues/12",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createIntake := func(t *testing.T, factory *models.Factory) *models.FactoryIntake {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		return intake
	}

	deps := func(items []IntakeItem, sourceErr error) IntakeDependencies {
		return IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				if sourceErr != nil {
					return nil, sourceErr
				}
				return stubIntakeItemSource{items: items}, nil
			},
		}
	}

	t.Run("returns items from the intake source", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		response, err := SearchFactoryIntakeItems(ctx, deps([]IntakeItem{item}, nil), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.GetItems(), 1)
		assert.Equal(t, item.ID, response.GetItems()[0].GetId())
		assert.Equal(t, item.Key, response.GetItems()[0].GetKey())
		assert.Equal(t, item.Title, response.GetItems()[0].GetTitle())
		assert.Equal(t, item.URL, response.GetItems()[0].GetUrl())
	})

	t.Run("unconnected intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		_, err := SearchFactoryIntakeItems(ctx, deps(nil, errIntakeNotConnected), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Connect this intake first.")
	})

	t.Run("unsupported intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		_, err := SearchFactoryIntakeItems(ctx, deps(nil, errIntakeSearchUnsupported), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "This intake cannot search items yet.")
	})
}

func Test__ImportFactoryIntakeItem(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	item := IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: "Handle duplicate refunds",
		Body:  "Retrying a refund posts twice.",
		URL:   "https://github.com/acme/payments/issues/12",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createIntake := func(t *testing.T, factory *models.Factory) *models.FactoryIntake {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		return intake
	}

	deps := IntakeDependencies{
		NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
			return stubIntakeItemSource{items: []IntakeItem{item}}, nil
		},
	}

	t.Run("creates a draft work order with the ticket origin", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		response, err := ImportFactoryIntakeItem(ctx, deps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    item.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, item.Title, response.GetOrder().GetTitle())
		assert.Equal(t, item.Body, response.GetOrder().GetDescription())
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, response.GetOrder().GetState())
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, item.URL, response.GetOrder().GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#12", response.GetOrder().GetOrigin().GetLabel())
		assert.Equal(t, r.User.String(), response.GetOrder().GetCreatedBy().GetUser().GetId())
		require.Len(t, response.GetOrder().GetAssignees(), 1)
		assert.Equal(t, r.User.String(), response.GetOrder().GetAssignees()[0].GetId())
	})

	t.Run("a second import of the same ticket creates a new work order", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)
		req := &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    item.ID,
		}

		first, err := ImportFactoryIntakeItem(ctx, deps, orgID, req)
		require.NoError(t, err)
		second, err := ImportFactoryIntakeItem(ctx, deps, orgID, req)
		require.NoError(t, err)
		assert.NotEqual(t, first.GetOrder().GetId(), second.GetOrder().GetId())
		assert.Equal(t, item.URL, second.GetOrder().GetOrigin().GetUrl())
	})
}

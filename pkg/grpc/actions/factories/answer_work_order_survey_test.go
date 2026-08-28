package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AnswerWorkOrderSurvey__StoresAnswersOnTheWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, _, err = order.CreateSurvey(db, models.FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions: []models.WorkOrderSurveyQuestion{
			{ID: "scope", Prompt: "Where?"},
			{ID: "tests", Prompt: "Add a test?"},
		},
	})
	require.NoError(t, err)

	resp, err := AnswerWorkOrderSurvey(ctx, r.Organization.ID.String(), &pb.AnswerWorkOrderSurveyRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		Answers:   []*pb.WorkOrderSurveyAnswer{{Id: "scope", Value: "Existing handler"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Order)
	assert.Nil(t, resp.Order.PendingSurvey)

	pending, err := order.PendingSurvey(db)
	assert.ErrorIs(t, err, models.ErrFactoryWorkOrderSurveyNotFound)
	assert.Nil(t, pending)
}

func Test__AnswerWorkOrderSurvey__RejectsWhenNoSurveyIsPending(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = AnswerWorkOrderSurvey(ctx, r.Organization.ID.String(), &pb.AnswerWorkOrderSurveyRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		Answers:   []*pb.WorkOrderSurveyAnswer{{Id: "scope", Value: "A"}},
	})
	require.Error(t, err)
}

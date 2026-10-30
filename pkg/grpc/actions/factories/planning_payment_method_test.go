package factories

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestPlanningFollowUpRequiresPaymentMethod(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	session := openAnalysisSession(t, r, db)
	plan := billingmodels.BillingPlan{Key: "planning_card_" + uuid.NewString()[:8], Name: "Card required", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	plan.RequirePaymentMethod = true
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	before, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[],"pagination":{"max_page":1}}`))
	}))
	defer server.Close()
	t.Setenv("POLAR_ACCESS_TOKEN", "test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	_, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{FactoryId: session.FactoryID.String(), SessionId: session.ID.String(), Text: "Continue work"})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	after, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, after, len(before), "a rejected prompt must not be queued")
}

func TestPlanningFollowUpRequiresActivePlan(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	session := openAnalysisSession(t, r, db)
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	before, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	_, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{FactoryId: session.FactoryID.String(), SessionId: session.ID.String(), Text: "Continue work"})
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	after, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, after, len(before), "a rejected prompt must not be queued")
}

package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAccountDependentBillingRejectsMissingIdentity(t *testing.T) {
	// No billing provider is installed: rejection must happen before dispatch.
	service := &OrganizationService{}
	for _, md := range []metadata.MD{nil, {}, metadata.Pairs("x-account-id", "")} {
		ctx := context.WithValue(t.Context(), authorization.DomainIdContextKey, "organization")
		if md != nil {
			ctx = metadata.NewIncomingContext(ctx, md)
		}
		_, err := service.SelectBillingPlan(ctx, &pb.SelectBillingPlanRequest{})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
		_, err = service.CreateHostedCreditCheckout(ctx, &pb.CreateHostedCreditCheckoutRequest{})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
		_, err = service.CreatePaymentMethodSession(ctx, &pb.CreatePaymentMethodSessionRequest{})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	}
}

package organizations

import (
	"testing"

	"github.com/stretchr/testify/require"
	coreorganizations "github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InvitationService = coreorganizations.InvitationService

var RemoveUser = coreorganizations.RemoveUser

const checkoutSeatLimitReason = "CHECKOUT_SEAT_LIMIT"

func requireInvitationReason(t *testing.T, err error, reason string) {
	t.Helper()
	requireInvitationErrorReason(t, err, codes.FailedPrecondition, reason)
}
func requireInvitationErrorReason(t *testing.T, err error, code codes.Code, reason string) {
	t.Helper()
	require.Equal(t, code, grpcerrors.Code(err))
	details := status.Convert(err).Details()
	require.Len(t, details, 1)
	require.Equal(t, reason, details[0].(*errdetails.ErrorInfo).Reason)
}

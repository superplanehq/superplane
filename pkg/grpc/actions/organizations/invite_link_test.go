package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestInviteLinkManagementUnavailable(t *testing.T) {
	for _, state := range []struct {
		name     string
		existing bool
		enabled  bool
	}{
		{name: "no link"},
		{name: "enabled", existing: true, enabled: true},
		{name: "disabled", existing: true},
	} {
		for _, action := range []struct {
			name string
			run  func(*testing.T, context.Context, string) error
		}{
			{name: "get", run: func(t *testing.T, ctx context.Context, orgID string) error {
				response, err := GetInviteLink(ctx, orgID)
				require.Nil(t, response)
				return err
			}},
			{name: "update", run: func(t *testing.T, ctx context.Context, orgID string) error {
				response, err := UpdateInviteLink(ctx, orgID, !state.enabled)
				require.Nil(t, response)
				return err
			}},
			{name: "reset", run: func(t *testing.T, ctx context.Context, orgID string) error {
				response, err := ResetInviteLink(ctx, orgID)
				require.Nil(t, response)
				return err
			}},
		} {
			t.Run(action.name+"/"+state.name, func(t *testing.T) {
				r := support.Setup(t)
				defer r.Close()
				db := database.DB(t.Context())
				var before *models.OrganizationInviteLink
				if state.existing {
					link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
					require.NoError(t, err)
					link.Enabled = state.enabled
					require.NoError(t, models.SaveInviteLink(db, link))
					before, err = models.FindInviteLinkByOrganizationID(db, r.Organization.ID.String())
					require.NoError(t, err)
				} else {
					require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationInviteLink{}).Error)
				}

				requireInviteLinksUnavailable(t, action.run(t, t.Context(), r.Organization.ID.String()))

				after, err := models.FindInviteLinkByOrganizationID(db, r.Organization.ID.String())
				if !state.existing {
					require.ErrorIs(t, err, gorm.ErrRecordNotFound)
					return
				}
				require.NoError(t, err)
				require.Equal(t, before, after)
			})
		}
	}
}

func requireInviteLinksUnavailable(t *testing.T, err error) {
	t.Helper()
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, code)
	require.Equal(t, "Invite links are no longer available. Use email invitations.", message)
}

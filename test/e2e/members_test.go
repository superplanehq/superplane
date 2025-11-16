package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestMembersInvitations(t *testing.T) {
	steps := &membersSteps{t: t}

	t.Run("inviting a new organization member", func(t *testing.T) {
		email := "e2e-member-invite@example.com"

		steps.start()
		steps.visitMembersPage()
		steps.fillInviteEmailTextarea(email)
		steps.submitInvitations()
		steps.assertInvitationPersistedInDB(email)
	})
}

type membersSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *membersSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *membersSteps) visitMembersPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/members")
}

func (s *membersSteps) fillInviteEmailTextarea(email string) {
	textarea := q.Locator(`textarea[placeholder="Email addresses, separated by commas"]`)
	s.session.FillIn(textarea, email)
	s.session.Sleep(300)
}

func (s *membersSteps) submitInvitations() {
	button := q.Locator(`button:has-text("Send Invitations")`)
	s.session.Click(button)
	s.session.Sleep(1500)
}

func (s *membersSteps) assertInvitationPersistedInDB(email string) {
	var invitation models.OrganizationInvitation
	err := database.Conn().Where("email = ? AND organization_id = ?", email, s.session.OrgID.String()).First(&invitation).Error
	require.NoError(s.t, err)
	require.Equal(s.t, email, invitation.Email)
	require.Equal(s.t, s.session.OrgID.String(), invitation.OrganizationID.String())
}

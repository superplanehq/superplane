package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestMembersInvitations(t *testing.T) {
	t.Run("owner can invite a member by email", func(t *testing.T) {
		steps := &membersSteps{t: t}
		steps.start()
		steps.visitMembersPage()
		steps.assertInviteMemberVisible()
	})

	t.Run("viewing existing organization members", func(t *testing.T) {
		steps := &membersSteps{t: t}
		steps.start()
		steps.visitMembersPage()
		steps.assertMembersCount("1 member")
		steps.assertMemberVisible("e2e@superplane.local")
	})
}

type membersSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *membersSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	s.session.Login()
}

func (s *membersSteps) visitMembersPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/members")
}

func (s *membersSteps) assertInviteMemberVisible() {
	s.session.AssertText("Invite member")
}

func (s *membersSteps) assertMembersCount(label string) {
	s.session.AssertText(label)
}

func (s *membersSteps) assertMemberVisible(email string) {
	s.session.AssertText(email)
}

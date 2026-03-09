package workers

import (
	"sort"
	"testing"

	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func Test__NotificationEmailConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	testEmailService := services.NewNoopEmailService()
	amqpURL := "amqp://guest:guest@rabbitmq:5672"

	consumer := NewNotificationEmailConsumer(amqpURL, testEmailService, r.AuthService)

	t.Run("should send notification email with deduped recipients", func(t *testing.T) {
		testEmailService.Reset()

		groupName := "engineering"
		err := r.AuthService.CreateGroup(
			r.Organization.ID.String(),
			models.DomainTypeOrganization,
			groupName,
			models.RoleOrgViewer,
			"Engineering",
			"Engineering",
		)
		require.NoError(t, err)

		groupUser := support.CreateUser(t, r, r.Organization.ID)
		err = r.AuthService.AddUserToGroup(
			r.Organization.ID.String(),
			models.DomainTypeOrganization,
			groupUser.ID.String(),
			groupName,
		)
		require.NoError(t, err)

		roleUser := support.CreateUser(t, r, r.Organization.ID)
		err = r.AuthService.AssignRole(
			roleUser.ID.String(),
			models.RoleOrgAdmin,
			r.Organization.ID.String(),
			models.DomainTypeOrganization,
		)
		require.NoError(t, err)

		payload, err := proto.Marshal(&protos.NotificationEmailRequested{
			OrganizationId: r.Organization.ID.String(),
			Title:          "Approval needed",
			Body:           "Please review the pending approval.",
			Url:            "https://app.superplane.com/approvals/123",
			UrlLabel:       "Review approval",
			Emails:         []string{groupUser.GetEmail(), "external@example.com"},
			Groups:         []string{groupName},
			Roles:          []string{models.RoleOrgAdmin},
		})
		require.NoError(t, err)

		err = consumer.Consume(tackle.NewFakeDelivery(payload))
		require.NoError(t, err)

		sentEmails := testEmailService.SentNotificationEmails()
		require.Len(t, sentEmails, 1)

		bcc := sentEmails[0].Bcc
		sort.Strings(bcc)

		expected := []string{groupUser.GetEmail(), roleUser.GetEmail(), "external@example.com"}
		sort.Strings(expected)

		assert.Equal(t, expected, bcc)
		assert.Equal(t, "Approval needed", sentEmails[0].Title)
		assert.Equal(t, "Please review the pending approval.", sentEmails[0].Body)
		assert.Equal(t, "https://app.superplane.com/approvals/123", sentEmails[0].URL)
		assert.Equal(t, "Review approval", sentEmails[0].URLLabel)
	})
}

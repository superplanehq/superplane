package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	organizations "github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

const (
	mailpitURL                   = "http://superplane-mailpit:8025"
	rabbitMQManagementURL        = "http://rabbitmq:15672"
	organizationMemberQueueVHost = "superplane_test"
)

// TestOrganizationMemberJoinedEmailMailpit exercises the complete notification
// path without mocks: accepting an invite publishes to RabbitMQ, the running
// email consumer sends via SMTP, and Mailpit exposes the delivered message.
func TestOrganizationMemberJoinedEmailMailpit(t *testing.T) {
	mailpit := newMailpitClient(mailpitURL)
	purgeOrganizationMemberJoinedQueues(t)
	mailpit.Clear(t)
	t.Cleanup(func() {
		purgeOrganizationMemberJoinedQueues(t)
		mailpit.Clear(t)
		require.NoError(t, models.DeleteEmailSettings(models.EmailProviderSMTP))
	})

	r := support.Setup(t)
	ownerEmail := setOwnerEmail(t, r)
	configureTestSMTP(t, "superplane-mailpit", 1025)

	invite := inviteLinkForOrganization(t, r.Organization.ID.String())
	member := createInvitationAccount(t, "member")
	response := acceptInvite(t, r, member, invite.Token.String())
	require.Equal(t, "joined", response.Fields["status"].GetStringValue())

	settingsURL := fmt.Sprintf("%s/%s/settings/members", ctx.baseURL, r.Organization.ID)
	first := mailpit.WaitForMemberJoinedCount(t, ownerEmail, member.Email, 1, 15*time.Second)
	assertMemberJoinedEmail(t, mailpit, first, ownerEmail, member, r.Organization.Name, settingsURL)

	response = acceptInvite(t, r, member, invite.Token.String())
	require.Equal(t, "already_member", response.Fields["status"].GetStringValue())
	mailpit.AssertMemberJoinedCountRemains(t, ownerEmail, member.Email, 1, 2*time.Second)

	memberUser, err := models.FindActiveUserByEmail(r.Organization.ID.String(), member.Email)
	require.NoError(t, err)
	_, err = organizations.RemoveUser(context.Background(), r.AuthService, r.Organization.ID.String(), memberUser.ID.String())
	require.NoError(t, err)

	response = acceptInvite(t, r, member, invite.Token.String())
	require.Equal(t, "joined", response.Fields["status"].GetStringValue())
	second := mailpit.WaitForMemberJoinedCount(t, ownerEmail, member.Email, 2, 15*time.Second)
	assertMemberJoinedEmail(t, mailpit, second, ownerEmail, member, r.Organization.Name, settingsURL)

	// A refused SMTP connection must not make invite acceptance fail. Restore the
	// Mailpit configuration afterwards so the delayed RabbitMQ retry drains.
	configureTestSMTP(t, "127.0.0.1", 1)
	failedDeliveryMember := createInvitationAccount(t, "delivery-failure")
	response = acceptInvite(t, r, failedDeliveryMember, invite.Token.String())
	require.Equal(t, "joined", response.Fields["status"].GetStringValue())
	_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), failedDeliveryMember.Email)
	require.NoError(t, err)
	mailpit.AssertMemberJoinedCountRemains(t, ownerEmail, failedDeliveryMember.Email, 0, time.Second)

	configureTestSMTP(t, "superplane-mailpit", 1025)
	third := mailpit.WaitForMemberJoinedCount(t, ownerEmail, failedDeliveryMember.Email, 1, 20*time.Second)
	assertMemberJoinedEmail(t, mailpit, third, ownerEmail, failedDeliveryMember, r.Organization.Name, settingsURL)
}

func acceptInvite(t *testing.T, r *support.ResourceRegistry, account *models.Account, token string) *structpb.Struct {
	t.Helper()
	response, err := organizations.AcceptInviteLinkWithUsage(t.Context(), r.AuthService, nil, account.ID.String(), token)
	require.NoError(t, err)
	return response
}

func configureTestSMTP(t *testing.T, host string, port int) {
	t.Helper()
	require.NoError(t, models.UpsertEmailSettings(&models.EmailSettings{
		Provider:      models.EmailProviderSMTP,
		SMTPHost:      host,
		SMTPPort:      port,
		SMTPFromName:  "SuperPlane",
		SMTPFromEmail: "notifications@superplane.local",
		SMTPUseTLS:    false,
	}))
}

func setOwnerEmail(t *testing.T, r *support.ResourceRegistry) string {
	t.Helper()
	email := support.RandomName("owner") + "@superplane.local"
	tx := database.DB(t.Context())
	require.NoError(t, tx.Model(r.Account).Update("email", email).Error)
	require.NoError(t, tx.Model(r.UserModel).Update("email", email).Error)
	r.Account.Email = email
	return email
}

func inviteLinkForOrganization(t *testing.T, organizationID string) *models.OrganizationInviteLink {
	t.Helper()
	invite, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), organizationID)
	if err == nil {
		return invite
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		require.NoError(t, err)
	}

	invite, err = models.CreateInviteLink(database.DB(t.Context()), mustUUID(t, organizationID))
	require.NoError(t, err)
	return invite
}

func createInvitationAccount(t *testing.T, prefix string) *models.Account {
	t.Helper()
	name := support.RandomName(prefix)
	account, err := models.CreateAccount(name, name+"@superplane.local")
	require.NoError(t, err)
	return account
}

func mustUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	require.NoError(t, err)
	return id
}

type mailpitClient struct {
	baseURL string
	client  *http.Client
}

type mailpitMailbox struct {
	Messages []mailpitMessage `json:"messages"`
}

type mailpitMessage struct {
	ID      string           `json:"ID"`
	From    mailpitAddress   `json:"From"`
	To      []mailpitAddress `json:"To"`
	Subject string           `json:"Subject"`
	Text    string           `json:"Text"`
	HTML    string           `json:"HTML"`
}

type mailpitAddress struct {
	Address string `json:"Address"`
}

func newMailpitClient(baseURL string) *mailpitClient {
	return &mailpitClient{baseURL: baseURL, client: &http.Client{Timeout: 5 * time.Second}}
}

func (c *mailpitClient) Clear(t *testing.T) {
	t.Helper()
	mailbox := c.Messages(t)
	if len(mailbox) == 0 {
		return
	}

	ids := make([]string, 0, len(mailbox))
	for _, message := range mailbox {
		ids = append(ids, message.ID)
	}
	payload, err := json.Marshal(map[string][]string{"IDs": ids})
	require.NoError(t, err)
	response := c.do(t, http.MethodDelete, "/api/v1/messages", bytes.NewReader(payload), http.StatusOK)
	response.Body.Close()
}

func (c *mailpitClient) WaitForMemberJoinedCount(t *testing.T, ownerEmail, memberEmail string, want int, timeout time.Duration) []mailpitMessage {
	t.Helper()
	var messages []mailpitMessage
	require.Eventually(t, func() bool {
		messages = c.MemberJoinedMessages(t, ownerEmail, memberEmail)
		return len(messages) == want
	}, timeout, 100*time.Millisecond)
	return messages
}

func (c *mailpitClient) AssertMemberJoinedCountRemains(t *testing.T, ownerEmail, memberEmail string, want int, duration time.Duration) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		require.Len(t, c.MemberJoinedMessages(t, ownerEmail, memberEmail), want)
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *mailpitClient) MemberJoinedMessages(t *testing.T, ownerEmail, memberEmail string) []mailpitMessage {
	t.Helper()
	var matches []mailpitMessage
	for _, summary := range c.Messages(t) {
		if len(summary.To) != 1 || summary.To[0].Address != ownerEmail || !strings.Contains(strings.ToLower(summary.Subject), "joined") {
			continue
		}
		message := c.Message(t, summary.ID)
		if strings.Contains(message.Text, memberEmail) {
			matches = append(matches, summary)
		}
	}
	return matches
}

func (c *mailpitClient) Messages(t *testing.T) []mailpitMessage {
	t.Helper()
	response := mailpitMailbox{}
	c.doJSON(t, http.MethodGet, "/api/v1/messages", nil, http.StatusOK, &response)
	return response.Messages
}

func (c *mailpitClient) Message(t *testing.T, id string) mailpitMessage {
	t.Helper()
	response := mailpitMessage{}
	c.doJSON(t, http.MethodGet, "/api/v1/message/"+url.PathEscape(id), nil, http.StatusOK, &response)
	return response
}

func (c *mailpitClient) doJSON(t *testing.T, method, path string, body io.Reader, expectedStatus int, destination any) {
	t.Helper()
	response := c.do(t, method, path, body, expectedStatus)
	defer response.Body.Close()
	require.NoError(t, json.NewDecoder(response.Body).Decode(destination))
}

func (c *mailpitClient) do(t *testing.T, method, path string, body io.Reader, expectedStatus int) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, body)
	require.NoError(t, err)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	require.NoError(t, err)
	if response.StatusCode != expectedStatus {
		defer response.Body.Close()
		content, readErr := io.ReadAll(response.Body)
		require.NoError(t, readErr)
		require.Equalf(t, expectedStatus, response.StatusCode, "Mailpit %s %s response: %s", method, path, content)
	}
	return response
}

func assertMemberJoinedEmail(t *testing.T, mailpit *mailpitClient, summaries []mailpitMessage, ownerEmail string, member *models.Account, organizationName, settingsURL string) {
	t.Helper()
	var message mailpitMessage
	found := false
	for _, summary := range summaries {
		candidate := mailpit.Message(t, summary.ID)
		if strings.Contains(candidate.Text, member.Email) {
			message = candidate
			found = true
			break
		}
	}
	require.Truef(t, found, "Mailpit did not contain an email for member %s", member.Email)
	require.Equal(t, "notifications@superplane.local", message.From.Address)
	require.Len(t, message.To, 1)
	require.Equal(t, ownerEmail, message.To[0].Address)
	require.Contains(t, strings.ToLower(message.Subject), "joined")
	require.Contains(t, message.Subject, organizationName)
	require.NotEmpty(t, message.Text)
	require.NotEmpty(t, message.HTML)
	for _, content := range []string{message.Text, message.HTML} {
		require.Contains(t, content, member.Name)
		require.Contains(t, content, member.Email)
		require.Contains(t, content, organizationName)
		require.Contains(t, content, settingsURL)
	}
}

func purgeOrganizationMemberJoinedQueues(t *testing.T) {
	t.Helper()
	queue := workers.OrganizationMemberJoinedEmailServiceName + "." + messages.OrganizationMemberJoinedRoutingKey
	for _, name := range []string{queue, queue + ".delay.10", queue + ".dead"} {
		endpoint := fmt.Sprintf("%s/api/queues/%s/%s/contents", rabbitMQManagementURL, organizationMemberQueueVHost, url.PathEscape(name))
		request, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, endpoint, nil)
		require.NoError(t, err)
		request.SetBasicAuth("guest", "guest")
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		response.Body.Close()
		require.Contains(t, []int{http.StatusNoContent, http.StatusNotFound}, response.StatusCode)
	}
}

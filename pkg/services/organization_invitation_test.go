package services

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/resend/resend-go/v3"
	"github.com/stretchr/testify/require"
)

func TestSMTPOrganizationInvitationContent(t *testing.T) {
	client := &fakeSMTPClient{}
	original := smtpDial
	smtpDial = func(string) (smtpClient, error) { return client, nil }
	t.Cleanup(func() { smtpDial = original })
	service := NewSMTPEmailService(&fakeSettingsProvider{settings: &SMTPSettings{Host: "smtp.example.com", Port: 25, FromName: "SuperPlane", FromEmail: "noreply@example.com"}}, filepath.Join("..", "..", "templates"))
	err := service.SendOrganizationInvitationEmail("recipient@example.com", OrganizationInvitationTemplateData{OrganizationName: "Engineering <Team>", InviterName: "Ana & Bo", Role: "Operator", ExpiresAt: "Tomorrow at noon UTC", AcceptanceURL: "https://example.com/invitations/token?x=a&y=b"})
	require.NoError(t, err)
	require.Equal(t, []string{"recipient@example.com"}, client.rcpt)
	message := client.message.String()
	for _, content := range []string{"Content-Type: text/plain", "Content-Type: text/html", "Engineering <Team>", "Engineering &lt;Team&gt;", "Ana & Bo", "Ana &amp; Bo", "Operator", "Tomorrow at noon UTC", "https://example.com/invitations/token?x=a&y=b", `href="https://example.com/invitations/token?x=a&amp;y=b"`} {
		require.Contains(t, message, content)
	}
}

type invitationTransport func(*http.Request) (*http.Response, error)

func (f invitationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResendOrganizationInvitationContent(t *testing.T) {
	var payload resend.SendEmailRequest
	calls := 0
	service := NewResendEmailService("re_test", "SuperPlane", "noreply@example.com", filepath.Join("..", "..", "templates"))
	service.client = resend.NewCustomClient(&http.Client{Transport: invitationTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/emails", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"email-123"}`))}, nil
	})}, "re_test")
	err := service.SendOrganizationInvitationEmail("recipient@example.com", OrganizationInvitationTemplateData{OrganizationName: "Engineering", InviterName: "Ana", Role: "Operator", ExpiresAt: "Tomorrow at noon UTC", AcceptanceURL: "https://example.com/invitations/token"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, []string{"recipient@example.com"}, payload.To)
	require.Equal(t, "Invitation to Engineering on SuperPlane", payload.Subject)
	for _, body := range []string{payload.Text, payload.Html} {
		for _, content := range []string{"Engineering", "Ana", "Operator", "Tomorrow at noon UTC", "https://example.com/invitations/token"} {
			require.Contains(t, body, content)
		}
	}
}

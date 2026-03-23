package services

import "sync"

type SentInvitationEmail struct {
	ToEmail          string
	OrganizationName string
	InvitationLink   string
	InviterEmail     string
}

type SentNotificationEmail struct {
	Bcc      []string
	Title    string
	Body     string
	URL      string
	URLLabel string
}

type SentMagicCodeEmail struct {
	ToEmail   string
	Code      string
	MagicLink string
}

type NoopEmailService struct {
	mu                sync.Mutex
	invitationEmails  []SentInvitationEmail
	notificationEmail []SentNotificationEmail
	magicCodeEmails   []SentMagicCodeEmail
}

func NewNoopEmailService() *NoopEmailService {
	return &NoopEmailService{
		invitationEmails:  []SentInvitationEmail{},
		notificationEmail: []SentNotificationEmail{},
		magicCodeEmails:   []SentMagicCodeEmail{},
	}
}

func (s *NoopEmailService) SendInvitationEmail(toEmail, organizationName, invitationLink, inviterEmail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.invitationEmails = append(s.invitationEmails, SentInvitationEmail{
		ToEmail:          toEmail,
		OrganizationName: organizationName,
		InvitationLink:   invitationLink,
		InviterEmail:     inviterEmail,
	})
	return nil
}

func (s *NoopEmailService) SendMagicCodeEmail(toEmail, code, magicLink string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.magicCodeEmails = append(s.magicCodeEmails, SentMagicCodeEmail{
		ToEmail:   toEmail,
		Code:      code,
		MagicLink: magicLink,
	})
	return nil
}

func (s *NoopEmailService) SendNotificationEmail(bccEmails []string, title, body, url, urlLabel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bccCopy := make([]string, len(bccEmails))
	copy(bccCopy, bccEmails)

	s.notificationEmail = append(s.notificationEmail, SentNotificationEmail{
		Bcc:      bccCopy,
		Title:    title,
		Body:     body,
		URL:      url,
		URLLabel: urlLabel,
	})

	return nil
}

func (s *NoopEmailService) SentInvitationEmails() []SentInvitationEmail {
	s.mu.Lock()
	defer s.mu.Unlock()

	emails := make([]SentInvitationEmail, len(s.invitationEmails))
	copy(emails, s.invitationEmails)
	return emails
}

func (s *NoopEmailService) SentMagicCodeEmails() []SentMagicCodeEmail {
	s.mu.Lock()
	defer s.mu.Unlock()

	emails := make([]SentMagicCodeEmail, len(s.magicCodeEmails))
	copy(emails, s.magicCodeEmails)
	return emails
}

func (s *NoopEmailService) SentNotificationEmails() []SentNotificationEmail {
	s.mu.Lock()
	defer s.mu.Unlock()

	emails := make([]SentNotificationEmail, len(s.notificationEmail))
	copy(emails, s.notificationEmail)
	return emails
}

func (s *NoopEmailService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.invitationEmails = []SentInvitationEmail{}
	s.notificationEmail = []SentNotificationEmail{}
	s.magicCodeEmails = []SentMagicCodeEmail{}
}

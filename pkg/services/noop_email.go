package services

import "sync"

type SentInvitationEmail struct {
	ToEmail          string
	OrganizationName string
	InvitationLink   string
	InviterEmail     string
}

type SentMagicCodeEmail struct {
	ToEmail   string
	Code      string
	MagicLink string
}

type NoopEmailService struct {
	mu               sync.Mutex
	invitationEmails []SentInvitationEmail
	magicCodeEmails  []SentMagicCodeEmail
}

func NewNoopEmailService() *NoopEmailService {
	return &NoopEmailService{
		invitationEmails: []SentInvitationEmail{},
		magicCodeEmails:  []SentMagicCodeEmail{},
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

func (s *NoopEmailService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.invitationEmails = []SentInvitationEmail{}
	s.magicCodeEmails = []SentMagicCodeEmail{}
}

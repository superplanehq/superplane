package services

import "sync"

type SentMagicCodeEmail struct {
	ToEmail   string
	Code      string
	MagicLink string
}

type SentOrganizationMemberJoinedEmail struct {
	ToEmail          string
	MemberName       string
	MemberEmail      string
	OrganizationName string
	SettingsURL      string
}

type NoopEmailService struct {
	mu                             sync.Mutex
	magicCodeEmails                []SentMagicCodeEmail
	organizationMemberJoinedEmails []SentOrganizationMemberJoinedEmail
}

func NewNoopEmailService() *NoopEmailService {
	return &NoopEmailService{
		magicCodeEmails:                []SentMagicCodeEmail{},
		organizationMemberJoinedEmails: []SentOrganizationMemberJoinedEmail{},
	}
}

func (s *NoopEmailService) SendOrganizationMemberJoinedEmail(toEmail, memberName, memberEmail, organizationName, settingsURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.organizationMemberJoinedEmails = append(s.organizationMemberJoinedEmails, SentOrganizationMemberJoinedEmail{
		ToEmail:          toEmail,
		MemberName:       memberName,
		MemberEmail:      memberEmail,
		OrganizationName: organizationName,
		SettingsURL:      settingsURL,
	})
	return nil
}

func (s *NoopEmailService) SentOrganizationMemberJoinedEmails() []SentOrganizationMemberJoinedEmail {
	s.mu.Lock()
	defer s.mu.Unlock()
	emails := make([]SentOrganizationMemberJoinedEmail, len(s.organizationMemberJoinedEmails))
	copy(emails, s.organizationMemberJoinedEmails)
	return emails
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

	s.magicCodeEmails = []SentMagicCodeEmail{}
	s.organizationMemberJoinedEmails = []SentOrganizationMemberJoinedEmail{}
}

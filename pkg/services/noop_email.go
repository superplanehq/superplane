package services

import "sync"

type SentMagicCodeEmail struct {
	ToEmail   string
	Code      string
	MagicLink string
}

type NoopEmailService struct {
	mu              sync.Mutex
	magicCodeEmails []SentMagicCodeEmail
}

func NewNoopEmailService() *NoopEmailService {
	return &NoopEmailService{
		magicCodeEmails: []SentMagicCodeEmail{},
	}
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
}

package services

import "sync"

type SentMagicCodeEmail struct {
	ToEmail   string
	Code      string
	MagicLink string
}

type SentWorkOrderNotificationEmail struct {
	ToEmail string
	Subject string
	Data    WorkOrderNotificationTemplateData
}

type NoopEmailService struct {
	mu                          sync.Mutex
	magicCodeEmails             []SentMagicCodeEmail
	workOrderNotificationEmails []SentWorkOrderNotificationEmail
}

func NewNoopEmailService() *NoopEmailService {
	return &NoopEmailService{
		magicCodeEmails:             []SentMagicCodeEmail{},
		workOrderNotificationEmails: []SentWorkOrderNotificationEmail{},
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

func (s *NoopEmailService) SendWorkOrderNotificationEmail(
	toEmail, subject string,
	data WorkOrderNotificationTemplateData,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workOrderNotificationEmails = append(s.workOrderNotificationEmails, SentWorkOrderNotificationEmail{
		ToEmail: toEmail,
		Subject: subject,
		Data:    data,
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

func (s *NoopEmailService) SentWorkOrderNotificationEmails() []SentWorkOrderNotificationEmail {
	s.mu.Lock()
	defer s.mu.Unlock()

	emails := make([]SentWorkOrderNotificationEmail, len(s.workOrderNotificationEmails))
	copy(emails, s.workOrderNotificationEmails)
	return emails
}

func (s *NoopEmailService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.magicCodeEmails = []SentMagicCodeEmail{}
	s.workOrderNotificationEmails = []SentWorkOrderNotificationEmail{}
}

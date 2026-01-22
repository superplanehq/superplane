package smtp

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromName  string
	FromEmail string
	UseTLS    bool
}

type Email struct {
	To        []string
	Cc        []string
	Bcc       []string
	Subject   string
	TextBody  string
	HTMLBody  string
	FromName  string
	FromEmail string
	ReplyTo   string
}

type SendResult struct {
	Success   bool      `json:"success"`
	To        []string  `json:"to"`
	Cc        []string  `json:"cc,omitempty"`
	Bcc       []string  `json:"bcc,omitempty"`
	Subject   string    `json:"subject"`
	SentAt    time.Time `json:"sentAt"`
	FromEmail string    `json:"fromEmail"`
}

// smtpClient interface for testing
type smtpClient interface {
	Hello(localName string) error
	Extension(ext string) (bool, string)
	StartTLS(config *tls.Config) error
	Auth(auth smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

type smtpClientAdapter struct {
	client *smtp.Client
}

func (c *smtpClientAdapter) Hello(localName string) error {
	return c.client.Hello(localName)
}

func (c *smtpClientAdapter) Extension(ext string) (bool, string) {
	return c.client.Extension(ext)
}

func (c *smtpClientAdapter) StartTLS(config *tls.Config) error {
	return c.client.StartTLS(config)
}

func (c *smtpClientAdapter) Auth(auth smtp.Auth) error {
	return c.client.Auth(auth)
}

func (c *smtpClientAdapter) Mail(from string) error {
	return c.client.Mail(from)
}

func (c *smtpClientAdapter) Rcpt(to string) error {
	return c.client.Rcpt(to)
}

func (c *smtpClientAdapter) Data() (io.WriteCloser, error) {
	return c.client.Data()
}

func (c *smtpClientAdapter) Quit() error {
	return c.client.Quit()
}

func (c *smtpClientAdapter) Close() error {
	return c.client.Close()
}

type smtpDialer func(addr string) (smtpClient, error)

var smtpDial smtpDialer = func(addr string) (smtpClient, error) {
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}
	return &smtpClientAdapter{client: client}, nil
}

func NewClient(ctx core.AppInstallationContext) (*Client, error) {
	host, err := ctx.GetConfig("host")
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	if len(host) == 0 {
		return nil, fmt.Errorf("host is required")
	}

	portBytes, err := ctx.GetConfig("port")
	if err != nil {
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	var port int
	if _, err := fmt.Sscanf(string(portBytes), "%d", &port); err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}

	fromEmail, err := ctx.GetConfig("fromEmail")
	if err != nil {
		return nil, fmt.Errorf("failed to get fromEmail: %w", err)
	}
	if len(fromEmail) == 0 {
		return nil, fmt.Errorf("fromEmail is required")
	}

	// Optional fields
	username, _ := ctx.GetConfig("username")
	password, _ := ctx.GetConfig("password")
	fromName, _ := ctx.GetConfig("fromName")

	useTLS := true
	useTLSBytes, err := ctx.GetConfig("useTLS")
	if err == nil && len(useTLSBytes) > 0 {
		useTLS = string(useTLSBytes) == "true"
	}

	return &Client{
		Host:      string(host),
		Port:      port,
		Username:  string(username),
		Password:  string(password),
		FromName:  string(fromName),
		FromEmail: string(fromEmail),
		UseTLS:    useTLS,
	}, nil
}

// Verify tests the SMTP connection and authentication
func (c *Client) Verify() error {
	conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Quit()
}

// SendEmail sends an email using the SMTP server
func (c *Client) SendEmail(email Email) (*SendResult, error) {
	// Determine sender info
	fromName := c.FromName
	if email.FromName != "" {
		fromName = email.FromName
	}

	fromEmail := c.FromEmail
	if email.FromEmail != "" {
		fromEmail = email.FromEmail
	}

	// Build recipients list
	recipients := make([]string, 0, len(email.To)+len(email.Cc)+len(email.Bcc))
	recipients = append(recipients, email.To...)
	recipients = append(recipients, email.Cc...)
	recipients = append(recipients, email.Bcc...)

	if len(recipients) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}

	// Build message
	message, err := c.buildMessage(email, fromName, fromEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to build message: %w", err)
	}

	// Connect and send
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.Mail(fromEmail); err != nil {
		return nil, fmt.Errorf("MAIL FROM failed: %w", err)
	}

	for _, recipient := range recipients {
		if err := conn.Rcpt(recipient); err != nil {
			return nil, fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	writer, err := conn.Data()
	if err != nil {
		return nil, fmt.Errorf("DATA command failed: %w", err)
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close data writer: %w", err)
	}

	if err := conn.Quit(); err != nil {
		return nil, fmt.Errorf("QUIT failed: %w", err)
	}

	return &SendResult{
		Success:   true,
		To:        email.To,
		Cc:        email.Cc,
		Bcc:       email.Bcc,
		Subject:   email.Subject,
		SentAt:    time.Now().UTC(),
		FromEmail: fromEmail,
	}, nil
}

func (c *Client) connect() (smtpClient, error) {
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := smtpDial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	if err := conn.Hello(c.Host); err != nil {
		conn.Close()
		return nil, fmt.Errorf("HELLO failed: %w", err)
	}

	if c.UseTLS {
		if ok, _ := conn.Extension("STARTTLS"); !ok {
			conn.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}

		tlsConfig := &tls.Config{
			ServerName: c.Host,
			MinVersion: tls.VersionTLS12,
		}

		if err := conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if c.Username != "" {
		auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
		if err := conn.Auth(auth); err != nil {
			conn.Close()
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	return conn, nil
}

func (c *Client) buildMessage(email Email, fromName, fromEmail string) (string, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return "", err
	}

	// Format from address
	from := fromEmail
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}

	// Build headers
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("Subject: %s", email.Subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"", boundary),
	}

	if len(email.To) > 0 {
		headers = append(headers, fmt.Sprintf("To: %s", strings.Join(email.To, ", ")))
	}

	if len(email.Cc) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(email.Cc, ", ")))
	}

	if email.ReplyTo != "" {
		headers = append(headers, fmt.Sprintf("Reply-To: %s", email.ReplyTo))
	}

	// Build message body
	message := strings.Join(headers, "\r\n") + "\r\n\r\n"

	// Determine text and HTML bodies
	textBody := email.TextBody
	htmlBody := email.HTMLBody

	// Auto-generate missing body type
	if textBody == "" && htmlBody != "" {
		textBody = stripHTML(htmlBody)
	}
	if htmlBody == "" && textBody != "" {
		htmlBody = wrapTextInHTML(textBody)
	}

	// Add text part
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n"
	message += textBody + "\r\n\r\n"

	// Add HTML part
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	message += htmlBody + "\r\n\r\n"

	// Close boundary
	message += fmt.Sprintf("--%s--\r\n", boundary)

	return message, nil
}

func randomBoundary() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func stripHTML(html string) string {
	// Simple HTML tag stripping - for basic conversion
	result := html

	// Replace common block elements with newlines
	blockTags := []string{"</p>", "</div>", "</br>", "<br>", "<br/>", "<br />"}
	for _, tag := range blockTags {
		result = strings.ReplaceAll(result, tag, "\n")
	}

	// Remove all remaining HTML tags
	var output strings.Builder
	inTag := false
	for _, r := range result {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			output.WriteRune(r)
		}
	}

	// Clean up whitespace
	lines := strings.Split(output.String(), "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

func wrapTextInHTML(text string) string {
	// Escape HTML entities
	escaped := strings.ReplaceAll(text, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")

	// Convert newlines to <br>
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="font-family: sans-serif; line-height: 1.5;">
%s
</body>
</html>`, escaped)
}

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const maxDiscordWebhookContentRunes = 2000

type DiscordWebhookClient struct {
	WebhookURL string
	HTTPClient *http.Client
}

func NewDiscordWebhookClient(webhookURL string) *DiscordWebhookClient {
	return &DiscordWebhookClient{
		WebhookURL: strings.TrimSpace(webhookURL),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *DiscordWebhookClient) Enabled() bool {
	return c != nil && c.WebhookURL != ""
}

func (c *DiscordWebhookClient) SendSupportFeedback(feedback SupportFeedback) error {
	if !c.Enabled() {
		return nil
	}

	content := FormatDiscordFeedbackContent(feedback)
	if feedback.Attachment == nil {
		return c.postJSON(map[string]string{"content": content})
	}
	return c.postMultipart(content, feedback.Attachment)
}

func FormatDiscordFeedbackContent(feedback SupportFeedback) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s**\n", FeedbackCategoryLabel(feedback.Category))
	fmt.Fprintf(&b, "From: %s\n", discordFromLine(feedback))
	if feedback.OrganizationName != "" || feedback.OrganizationID != "" {
		fmt.Fprintf(&b, "Organization: %s\n", discordOrganizationLine(feedback))
	}
	if feedback.PagePath != "" {
		fmt.Fprintf(&b, "Page: `%s`\n", feedback.PagePath)
	}
	b.WriteString("\n")
	b.WriteString(feedback.Details)
	if feedback.Attachment != nil {
		fmt.Fprintf(&b, "\n\nAttachment: %s", feedback.Attachment.Filename)
	}

	content := b.String()
	if utf8.RuneCountInString(content) <= maxDiscordWebhookContentRunes {
		return content
	}
	runes := []rune(content)
	return string(runes[:maxDiscordWebhookContentRunes-1]) + "…"
}

func discordFromLine(feedback SupportFeedback) string {
	name := strings.TrimSpace(feedback.UserName)
	email := strings.TrimSpace(feedback.UserEmail)
	switch {
	case name != "" && email != "":
		return name + " <" + email + ">"
	case email != "":
		return email
	case name != "":
		return name
	default:
		return "Unknown user"
	}
}

func discordOrganizationLine(feedback SupportFeedback) string {
	name := strings.TrimSpace(feedback.OrganizationName)
	id := strings.TrimSpace(feedback.OrganizationID)
	switch {
	case name != "" && id != "":
		return name + " (" + id + ")"
	case name != "":
		return name
	default:
		return id
	}
}

func (c *DiscordWebhookClient) postJSON(payload map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal discord webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create discord webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req)
}

func (c *DiscordWebhookClient) postMultipart(content string, attachment *SupportFeedbackAttachment) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return fmt.Errorf("marshal discord webhook payload: %w", err)
	}
	if err := writer.WriteField("payload_json", string(payload)); err != nil {
		return fmt.Errorf("write discord payload_json: %w", err)
	}

	fileWriter, err := writer.CreateFormFile("files[0]", attachment.Filename)
	if err != nil {
		return fmt.Errorf("create discord file part: %w", err)
	}
	if _, err := fileWriter.Write(attachment.Content); err != nil {
		return fmt.Errorf("write discord file part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close discord multipart body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, &body)
	if err != nil {
		return fmt.Errorf("create discord webhook request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return c.do(req)
}

func (c *DiscordWebhookClient) do(req *http.Request) error {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	limited := io.LimitReader(resp.Body, 1024)
	responseBody, _ := io.ReadAll(limited)
	return fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
}

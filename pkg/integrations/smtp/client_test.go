package smtp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMessage_StripsCRLFFromHeaders(t *testing.T) {
	client := &Client{}
	msg, err := client.buildMessage(Email{
		To:        []string{"to@example.com\r\nBcc: attacker@external.com"},
		Cc:        []string{"cc@example.com\r\nFrom: spoofed@victim.com"},
		Subject:   "Urgent\r\nFrom: ceo@victim.com\r\nBcc: attacker@external.com",
		TextBody:  "plain body",
		ReplyTo:   "reply@example.com\r\nCc: evil@external.com",
		FromName:  "Sender\r\nBcc: evil@external.com",
		FromEmail: "sender@example.com",
	}, "Sender\r\nBcc: evil@external.com", "sender@example.com")
	require.NoError(t, err)

	assert.Contains(t, msg, "Subject: UrgentFrom: ceo@victim.comBcc: attacker@external.com")
	assert.Contains(t, msg, "From: SenderBcc: evil@external.com <sender@example.com>")
	assert.Contains(t, msg, "To: to@example.comBcc: attacker@external.com")
	assert.Contains(t, msg, "Cc: cc@example.comFrom: spoofed@victim.com")
	assert.Contains(t, msg, "Reply-To: reply@example.comCc: evil@external.com")
	assert.NotContains(t, msg, "\r\nFrom: ceo@victim.com")
	assert.NotContains(t, msg, "\r\nBcc: attacker@external.com")
	assert.Equal(t, 1, countHeaderLines(msg, "From:"))
	assert.Equal(t, 1, countHeaderLines(msg, "Subject:"))
	assert.Equal(t, 1, countHeaderLines(msg, "To:"))
	assert.Equal(t, 1, countHeaderLines(msg, "Cc:"))
	assert.Equal(t, 1, countHeaderLines(msg, "Reply-To:"))
}

func countHeaderLines(message, headerPrefix string) int {
	headerEnd := strings.Index(message, "\r\n\r\n")
	if headerEnd < 0 {
		return 0
	}

	count := 0
	for _, line := range strings.Split(message[:headerEnd], "\r\n") {
		if strings.HasPrefix(line, headerPrefix) {
			count++
		}
	}
	return count
}

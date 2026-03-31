package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestDeleteEmailSettings(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	require.NoError(t, UpsertEmailSettings(&EmailSettings{
		Provider:      EmailProviderSMTP,
		SMTPHost:      "smtp.example.com",
		SMTPPort:      587,
		SMTPFromEmail: "noreply@example.com",
		SMTPUseTLS:    true,
	}))

	require.NoError(t, DeleteEmailSettings(EmailProviderSMTP))

	_, err := FindEmailSettings(EmailProviderSMTP)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

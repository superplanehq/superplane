package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func strPtr(s string) *string { return &s }

func TestAccountSurveyResponse(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("stores a complete submission", func(t *testing.T) {
		acct, err := CreateAccount("Alice", "alice-survey@example.com")
		require.NoError(t, err)

		resp, err := CreateAccountSurveyResponseInTransaction(database.Conn(), AccountSurveyResponseInput{
			AccountID:     acct.ID,
			SurveyType:    SurveyTypeSignup,
			Skipped:       false,
			SourceChannel: strPtr(SourceChannelSearch),
			Role:          strPtr(RoleEngineer),
			UseCase:       strPtr("schedule ML jobs"),
		})
		require.NoError(t, err)
		assert.Equal(t, acct.ID, resp.AccountID)
		assert.Equal(t, "signup", resp.SurveyType)
		assert.False(t, resp.Skipped)
		require.NotNil(t, resp.SourceChannel)
		assert.Equal(t, "search", *resp.SourceChannel)
	})

	t.Run("stores a minimal skipped row", func(t *testing.T) {
		acct, err := CreateAccount("Bob", "bob-survey@example.com")
		require.NoError(t, err)

		resp, err := CreateAccountSurveyResponseInTransaction(database.Conn(), AccountSurveyResponseInput{
			AccountID:  acct.ID,
			SurveyType: SurveyTypeSignup,
			Skipped:    true,
		})
		require.NoError(t, err)
		assert.True(t, resp.Skipped)
		assert.Nil(t, resp.SourceChannel)
		assert.Nil(t, resp.Role)
		assert.Nil(t, resp.UseCase)
	})

	t.Run("enforces UNIQUE(account_id)", func(t *testing.T) {
		acct, err := CreateAccount("Carol", "carol-survey@example.com")
		require.NoError(t, err)

		_, err = CreateAccountSurveyResponseInTransaction(database.Conn(), AccountSurveyResponseInput{
			AccountID:  acct.ID,
			SurveyType: SurveyTypeSignup,
			Skipped:    true,
		})
		require.NoError(t, err)

		_, err = CreateAccountSurveyResponseInTransaction(database.Conn(), AccountSurveyResponseInput{
			AccountID:  acct.ID,
			SurveyType: SurveyTypeSignup,
			Skipped:    true,
		})
		assert.Error(t, err, "second insert for same account must violate UNIQUE(account_id)")
	})
}

func TestIsValidSourceChannel(t *testing.T) {
	assert.True(t, IsValidSourceChannel("search"))
	assert.True(t, IsValidSourceChannel("other"))
	assert.False(t, IsValidSourceChannel(""))
	assert.False(t, IsValidSourceChannel("tiktok"))
}

func TestIsValidRole(t *testing.T) {
	assert.True(t, IsValidRole("engineer"))
	assert.True(t, IsValidRole("other"))
	assert.False(t, IsValidRole(""))
	assert.False(t, IsValidRole("poet"))
}

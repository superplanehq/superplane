package public

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

// freshSelfSignupAccount creates an account and returns it alongside a signed
// JWT cookie value authenticated as that account.
func freshSelfSignupAccount(t *testing.T, name, email string) (*models.Account, string) {
	t.Helper()
	acct, err := models.CreateAccount(name, email)
	require.NoError(t, err)

	signer := jwt.NewSigner("test-client-secret")
	token, err := signer.Generate(acct.ID.String(), time.Hour)
	require.NoError(t, err)
	return acct, token
}

func postSurvey(t *testing.T, server *Server, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, "/signup-survey", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	}
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func Test__PostSignupSurvey_SubmitHappyPath(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	acct, token := freshSelfSignupAccount(t, "Gail", "gail-survey@example.com")

	rec := postSurvey(t, server, token, `{
		"skipped": false,
		"source_channel": "search",
		"role": "engineer",
		"use_case": "schedule ML jobs"
	}`)
	require.Equal(t, http.StatusNoContent, rec.Code)

	var row models.AccountSurveyResponse
	require.NoError(t, database.Conn().Where("account_id = ?", acct.ID).First(&row).Error)
	assert.False(t, row.Skipped)
	require.NotNil(t, row.SourceChannel)
	assert.Equal(t, "search", *row.SourceChannel)
	require.NotNil(t, row.Role)
	assert.Equal(t, "engineer", *row.Role)
	require.NotNil(t, row.UseCase)
	assert.Equal(t, "schedule ML jobs", *row.UseCase)
}

func Test__PostSignupSurvey_SkipPath(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	acct, token := freshSelfSignupAccount(t, "Hugo", "hugo-survey@example.com")

	rec := postSurvey(t, server, token, `{"skipped": true}`)
	require.Equal(t, http.StatusNoContent, rec.Code)

	var row models.AccountSurveyResponse
	require.NoError(t, database.Conn().Where("account_id = ?", acct.ID).First(&row).Error)
	assert.True(t, row.Skipped)
	assert.Nil(t, row.SourceChannel)
}

func Test__PostSignupSurvey_IdempotentOnSecondSubmit(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	acct, token := freshSelfSignupAccount(t, "Ivy", "ivy-survey@example.com")

	require.Equal(t, http.StatusNoContent, postSurvey(t, server, token, `{"skipped": true}`).Code)
	require.Equal(t, http.StatusNoContent, postSurvey(t, server, token, `{"skipped": true}`).Code)

	var count int64
	require.NoError(t, database.Conn().Model(&models.AccountSurveyResponse{}).
		Where("account_id = ?", acct.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "second submit must not insert a duplicate row")
}

func Test__PostSignupSurvey_RejectsInvalidEnum(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	_, token := freshSelfSignupAccount(t, "Jan", "jan-survey@example.com")

	rec := postSurvey(t, server, token, `{"skipped": false, "source_channel": "tiktok"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__PostSignupSurvey_RejectsSkippedWithFields(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	_, token := freshSelfSignupAccount(t, "Kim", "kim-survey@example.com")

	rec := postSurvey(t, server, token, `{"skipped": true, "source_channel": "search"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__PostSignupSurvey_Unauthenticated(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	t.Setenv("SIGNUP_SURVEY_ENABLED", "yes")

	rec := postSurvey(t, server, "", `{"skipped":true}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

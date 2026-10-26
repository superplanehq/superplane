package authentication

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestInvitationSignupWithRegistrationDisabled(t *testing.T) {
	for _, method := range []string{"password", "magic code", "provider"} {
		for _, scenario := range []struct {
			name       string
			state      string
			expired    bool
			wrongEmail bool
			sharedLink bool
			replaced   bool
			allowed    bool
		}{
			{name: "invited email", state: models.InvitationStatePending, allowed: true},
			{name: "different email", state: models.InvitationStatePending, wrongEmail: true},
			{name: "expired invitation", state: models.InvitationStatePending, expired: true},
			{name: "canceled invitation", state: models.InvitationStateCanceled},
			{name: "swept expired invitation", state: models.InvitationStateExpired},
			{name: "accepted invitation", state: models.InvitationStateAccepted},
			{name: "shared invite link", sharedLink: true},
			{name: "replaced invitation", state: models.InvitationStatePending, replaced: true},
		} {
			t.Run(method+"/"+scenario.name, func(t *testing.T) {
				handler, registry := setupAuthHandler(t, true)
				handler.passwordLoginEnabled = true
				handler.magicCodeEnabled = method == "magic code"
				email := "invited@example.com"
				token := uuid.NewString()
				redirect := "/invitations/" + token
				if scenario.sharedLink {
					link := support.SeedHistoricalInviteLink(t, registry.Organization.ID, true)
					token = link.Token.String()
					redirect = "/invite/" + token
				} else {
					expiresAt := time.Now().Add(time.Hour)
					if scenario.expired {
						expiresAt = time.Now().Add(-time.Hour)
					}
					hash := crypto.HashToken(token)
					invitation := models.OrganizationInvitation{
						OrganizationID: registry.Organization.ID,
						Email:          email,
						Role:           models.RoleOrgOperator,
						InvitedBy:      registry.User,
						TokenHash:      &hash,
						State:          scenario.state,
						SentAt:         time.Now(),
						ExpiresAt:      expiresAt,
					}
					require.NoError(t, invitation.Save(database.Conn()))
					if scenario.replaced {
						require.NoError(t, database.Conn().Exec("INSERT INTO organization_invitation_replaced_tokens (invitation_id, token_hash) VALUES (?, ?)", invitation.ID, hash).Error)
						currentHash := crypto.HashToken(uuid.NewString())
						invitation.TokenHash = &currentHash
						require.NoError(t, invitation.Save(database.Conn()))
					}
				}
				if scenario.wrongEmail {
					email = "someone-else@example.com"
				}

				recorder := httptest.NewRecorder()
				switch method {
				case "password":
					form := url.Values{"name": {"Invited person"}, "email": {strings.ToUpper(email)}, "password": {"password123"}, "invite_token": {token}}
					if scenario.allowed {
						code := "654321"
						_, err := models.CreateAccountMagicCode(email, crypto.HashToken(code), time.Now().Add(time.Hour))
						require.NoError(t, err)
						denied := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodPost, "/signup?redirect="+url.QueryEscape(redirect), strings.NewReader(form.Encode()))
						request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
						handler.handlePasswordSignup(denied, request)
						_, err = models.FindAccountByEmail(email)
						require.ErrorIs(t, err, gorm.ErrRecordNotFound)
						require.Empty(t, cookieValue(denied, "account_token"))
						require.NotEqual(t, http.StatusSeeOther, denied.Code)
						form.Set("email_code", code)
					}
					request := httptest.NewRequest(http.MethodPost, "/signup?redirect="+url.QueryEscape(redirect), strings.NewReader(form.Encode()))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					handler.handlePasswordSignup(recorder, request)
				case "magic code":
					code := "123456"
					_, err := models.CreateAccountMagicCode(email, crypto.HashToken(code), time.Now().Add(time.Hour))
					require.NoError(t, err)
					form := url.Values{"email": {strings.ToUpper(email)}, "code": {code}}
					request := httptest.NewRequest(http.MethodPost, "/auth/magic-code/verify?redirect="+url.QueryEscape(redirect), strings.NewReader(form.Encode()))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					handler.handleMagicCodeVerify(recorder, request)
				case "provider":
					request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+url.QueryEscape(redirect), nil), map[string]string{"provider": models.ProviderGoogle})
					handler.completeProviderAuth(recorder, request, goth.User{UserID: "invited-person", Email: strings.ToUpper(email), Name: "Invited person", Provider: models.ProviderGoogle})
				}

				account, err := models.FindAccountByEmail(email)
				if !scenario.allowed {
					require.ErrorIs(t, err, gorm.ErrRecordNotFound)
					require.Empty(t, cookieValue(recorder, "account_token"))
					if method == "provider" {
						require.Equal(t, http.StatusSeeOther, recorder.Code)
						require.Contains(t, recorder.Header().Get("Location"), "auth_error=signup_disabled")
						return
					}
					require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
					return
				}
				require.NoError(t, err)
				require.Equal(t, account.ID.String(), sessionAccountID(t, recorder, handler.jwtSigner))
				location, err := url.Parse(recorder.Header().Get("Location"))
				require.NoError(t, err)
				if method == "magic code" {
					require.Equal(t, redirect, location.Query().Get("redirect"))
					return
				}
				require.Equal(t, redirect, location.Path)
			})
		}
	}
}

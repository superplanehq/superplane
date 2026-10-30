package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/e2e/session"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMemberSeatReuse(t *testing.T) {
	steps := &memberSeatReuseSteps{t: t}
	steps.givenAFourSeatOrganizationIsFull()
	steps.givenTheMemberHasWorkingCredentials()
	steps.assertInvitationSeats(4, 0, 0)
	steps.whenTheOwnerRemovesTheMember()
	steps.thenTheSameCredentialsLoseAccess()
	steps.thenTheMembersPageShowsAnAvailableSeat()
	steps.assertInvitationSeats(3, 0, 1)
	steps.whenTheOwnerInvitesAReplacement()
	steps.assertInvitationSeats(3, 1, 0)
	steps.whenTheReplacementAcceptsTheDisplayedInvitation()
	steps.thenTheReplacementIsAVisibleMember()
	steps.assertInvitationSeats(4, 0, 0)
}

type memberSeatReuseSteps struct {
	t             *testing.T
	session       *session.TestSession
	member        *models.User
	replacement   *models.Account
	ownerCookie   *http.Cookie
	credentials   []*http.Request
	acceptanceURL string
	client        http.Client
}

func (s *memberSeatReuseSteps) givenAFourSeatOrganizationIsFull() {
	polar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	s.t.Cleanup(polar.Close)
	s.t.Setenv("POLAR_API_BASE_URL", polar.URL)
	s.t.Setenv("POLAR_ACCESS_TOKEN", "seat-reuse-test")
	s.t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	s.client.Timeout = 5 * time.Second
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	db := database.DB(s.t.Context())
	maximum, purchased := 50, 4
	product := "product_" + uuid.NewString()
	plan := billingmodels.BillingPlan{
		Key: "seat_reuse_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: 2000, SeatLimit: &maximum, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(s.t, billingmodels.SaveBillingPlan(db, &plan))
	require.NoError(s.t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", s.session.OrgID).Updates(map[string]any{
		"plan_id": plan.ID, "plan": plan.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
		"polar_subscription_id": "sub_" + uuid.NewString(), "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		"purchased_seats": purchased, "current_period_end": time.Now().Add(30 * 24 * time.Hour),
	}).Error)
	auth, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	for i := 0; i < 3; i++ {
		account, err := models.CreateAccount(fmt.Sprintf("Seat member %d", i), fmt.Sprintf("seat-member-%d@superplane.local", i))
		require.NoError(s.t, err)
		member, err := models.CreateUser(s.session.OrgID, account.ID, account.Email, account.Name)
		require.NoError(s.t, err)
		require.NoError(s.t, auth.AssignRole(member.ID.String(), models.RoleOrgOperator, s.session.OrgID.String(), models.DomainTypeOrganization))
		if i == 0 {
			s.member = member
		}
	}
	s.replacement, err = models.CreateAccount("Replacement member", "seat-replacement@superplane.local")
	require.NoError(s.t, err)
	s.session.Login()
	s.ownerCookie = s.currentSessionCookie()
}

func (s *memberSeatReuseSteps) givenTheMemberHasWorkingCredentials() {
	owner := s.session.Account
	memberAccount, err := models.FindAccountByID(s.member.AccountID.String())
	require.NoError(s.t, err)
	s.session.Account = memberAccount
	s.session.Login()
	cookie := s.currentSessionCookie()
	s.session.Account = owner
	s.session.Login()

	plaintext, err := crypto.Base64String(64)
	require.NoError(s.t, err)
	token := models.NewUserAPIToken(s.member.ID, "Removed member automation", crypto.HashToken(plaintext))
	require.NoError(s.t, models.CreateUserAPIToken(database.DB(s.t.Context()), token))
	sessionRequest := s.apiRequest("/api/v1/me")
	sessionRequest.AddCookie(cookie)
	tokenRequest := s.apiRequest("/api/v1/me")
	tokenRequest.Header.Set("Authorization", "Bearer "+plaintext)
	s.credentials = []*http.Request{sessionRequest, tokenRequest}
	for _, request := range s.credentials {
		response, err := s.client.Do(request)
		require.NoError(s.t, err)
		func() {
			defer response.Body.Close()
			require.Equal(s.t, http.StatusOK, response.StatusCode)
			var body struct {
				User struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"user"`
			}
			require.NoError(s.t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(s.t, s.member.ID.String(), body.User.ID)
			require.Equal(s.t, memberAccount.Email, body.User.Email)
		}()
	}
}

func (s *memberSeatReuseSteps) whenTheOwnerRemovesTheMember() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/members")
	s.session.AssertText("4 of 4 seats used")
	page := s.session.Page()
	require.NoError(s.t, page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Member actions for " + s.member.Name, Exact: pw.Bool(true)}).Click())
	response, err := page.ExpectResponse("**/api/v1/organizations/*/users/"+s.member.ID.String(), func() error {
		return page.GetByRole("menuitem", pw.PageGetByRoleOptions{Name: "Remove", Exact: pw.Bool(true)}).Click()
	})
	require.NoError(s.t, err)
	require.Equal(s.t, http.MethodDelete, response.Request().Method())
	require.Equal(s.t, http.StatusOK, response.Status())
}

func (s *memberSeatReuseSteps) thenTheSameCredentialsLoseAccess() {
	for i, request := range s.credentials {
		response, err := s.client.Do(request)
		require.NoError(s.t, err)
		require.NoError(s.t, response.Body.Close())
		require.Equal(s.t, []int{http.StatusNotFound, http.StatusUnauthorized}[i], response.StatusCode)
	}
}

func (s *memberSeatReuseSteps) thenTheMembersPageShowsAnAvailableSeat() {
	require.NoError(s.t, s.session.Page().GetByRole("row").Filter(pw.LocatorFilterOptions{HasText: s.member.GetEmail()}).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
	s.session.AssertText("3 of 4 seats used")
}

func (s *memberSeatReuseSteps) whenTheOwnerInvitesAReplacement() {
	page := s.session.Page()
	require.NoError(s.t, page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Invite member", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, page.GetByLabel("Email address", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Fill(s.replacement.Email))
	response, err := page.ExpectResponse("**/api/v1/organizations/*/invitations", func() error {
		return page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Send invitation", Exact: pw.Bool(true)}).Click()
	})
	require.NoError(s.t, err)
	require.Equal(s.t, http.MethodPost, response.Request().Method())
	require.Equal(s.t, http.StatusOK, response.Status())
	link := page.GetByLabel("Invitation link for "+s.replacement.Email, pw.PageGetByLabelOptions{Exact: pw.Bool(true)})
	require.NoError(s.t, link.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
	s.acceptanceURL, err = link.InputValue()
	require.NoError(s.t, err)
	require.True(s.t, strings.HasPrefix(s.acceptanceURL, s.session.BaseURL+"/invitations/"))
	require.NoError(s.t, page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Done", Exact: pw.Bool(true)}).Click())
	s.session.AssertText(s.replacement.Email)
}

func (s *memberSeatReuseSteps) whenTheReplacementAcceptsTheDisplayedInvitation() {
	s.session.Account = s.replacement
	s.session.Login()
	response, err := s.session.Page().ExpectResponse("**/api/v1/invitations/*/accept", func() error {
		s.session.Visit(strings.TrimPrefix(s.acceptanceURL, s.session.BaseURL))
		return nil
	})
	require.NoError(s.t, err)
	require.Equal(s.t, http.MethodPost, response.Request().Method())
	require.Equal(s.t, http.StatusOK, response.Status())
	require.NoError(s.t, s.session.Page().WaitForURL("**/"+s.session.OrgSlug+"**"))
}

func (s *memberSeatReuseSteps) thenTheReplacementIsAVisibleMember() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/members")
	row := s.session.Page().GetByRole("row").Filter(pw.LocatorFilterOptions{HasText: s.replacement.Email})
	require.NoError(s.t, row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Member actions for " + s.replacement.Name, Exact: pw.Bool(true)}).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
	s.session.AssertText("4 of 4 seats used")
}

func (s *memberSeatReuseSteps) assertInvitationSeats(occupied, pending, invitable int64) {
	request := s.apiRequest("/api/v1/organizations/" + s.session.OrgID.String() + "/invitations")
	request.AddCookie(s.ownerCookie)
	response, err := s.client.Do(request)
	require.NoError(s.t, err)
	defer response.Body.Close()
	require.Equal(s.t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(s.t, err)
	var invitations pb.ListInvitationsResponse
	require.NoError(s.t, protojson.Unmarshal(body, &invitations))
	seats := invitations.Seats
	require.NotNil(s.t, seats)
	require.True(s.t, seats.PerSeat)
	require.NotNil(s.t, seats.Purchased)
	require.EqualValues(s.t, 4, *seats.Purchased)
	require.Equal(s.t, occupied, seats.Occupied)
	require.Equal(s.t, pending, seats.PendingInvitations)
	require.NotNil(s.t, seats.Invitable)
	require.Equal(s.t, invitable, *seats.Invitable)
}

func (s *memberSeatReuseSteps) apiRequest(path string) *http.Request {
	request, err := http.NewRequestWithContext(s.t.Context(), http.MethodGet, s.session.BaseURL+path, nil)
	require.NoError(s.t, err)
	request.Header.Set("x-organization-id", s.session.OrgID.String())
	return request
}

func (s *memberSeatReuseSteps) currentSessionCookie() *http.Cookie {
	cookies, err := s.session.Page().Context().Cookies(s.session.BaseURL)
	require.NoError(s.t, err)
	for _, cookie := range cookies {
		if cookie.Name == "account_token" {
			return &http.Cookie{Name: cookie.Name, Value: cookie.Value}
		}
	}
	s.t.Fatal("account session cookie is missing")
	return nil
}

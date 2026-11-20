package sso

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestSCIMAdoptsAJITIdentity(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "ada-subject", "ada@example.com"))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)

	filter := f.scim(http.MethodGet, conn, created.Secret, "/Users?filter=userName%20eq%20%22ada@example.com%22", "")
	require.Equal(t, http.StatusOK, filter.Code, filter.Body.String())
	require.Contains(t, filter.Body.String(), `"totalResults":1`)
	require.Contains(t, filter.Body.String(), `"userName":"ada@example.com"`)

	body := `{"userName":"ada@example.com","externalId":"okta-ada","active":true,"emails":[{"value":"ada@example.com","primary":true}]}`
	response := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "ada-subject")
	require.NoError(t, err)
	require.NotNil(t, identity.SCIMUserName)
	require.Equal(t, "ada@example.com", *identity.SCIMUserName)
	require.Equal(t, models.SSOProvisionedBySCIM, identity.ProvisionedBy)
	member, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.NoError(t, err)
	require.NotNil(t, member.ProvisionedBy)
	require.Equal(t, models.SSOProvisionedBySCIM, *member.ProvisionedBy)
}

func TestSCIMDoesNotReplaceAnotherUserName(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "ada-subject", "ada@example.com"))
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "ada-subject")
	require.NoError(t, err)
	userName := "ada@example.com"
	identity.SCIMUserName = &userName
	require.NoError(t, identity.Save(database.Conn()))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)

	body := `{"userName":"ada.other@example.com","externalId":"okta-other","active":true,"emails":[{"value":"ada@example.com","primary":true}]}`
	response := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "uniqueness")

	again, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "ada-subject")
	require.NoError(t, err)
	require.NotNil(t, again.SCIMUserName)
	require.Equal(t, "ada@example.com", *again.SCIMUserName)
	require.Nil(t, again.ExternalID)
}

func TestSCIMUserValidationAndUpdates(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.Equal(t, models.SSOStatusDraft, conn.Status)
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	secret := created.Secret

	config := f.scim(http.MethodGet, conn, secret, "/ServiceProviderConfig", "")
	require.Equal(t, http.StatusOK, config.Code, config.Body.String())
	require.Contains(t, config.Body.String(), `"supported":true`)
	schemas := f.scim(http.MethodGet, conn, secret, "/Schemas", "")
	require.Equal(t, http.StatusOK, schemas.Code)
	require.Contains(t, schemas.Body.String(), "User")

	emptyName := f.scim(http.MethodPost, conn, secret, "/Users", `{"userName":"","emails":[{"value":"ada@example.com"}]}`)
	require.Equal(t, http.StatusBadRequest, emptyName.Code)
	require.Contains(t, emptyName.Body.String(), "invalidValue")

	badEmail := f.scim(http.MethodPost, conn, secret, "/Users", `{"userName":"foo@","emails":[{"value":"foo@"}]}`)
	require.Equal(t, http.StatusBadRequest, badEmail.Code)
	require.Contains(t, badEmail.Body.String(), "invalidValue")

	body := `{"userName":"ada@example.com","externalId":"ext-ada","displayName":"Ada","active":true,"emails":[{"value":"ada@example.com","primary":true}]}`
	createdUser := f.scim(http.MethodPost, conn, secret, "/Users", body)
	require.Equal(t, http.StatusCreated, createdUser.Code, createdUser.Body.String())
	userID := jsonString(t, createdUser.Body.String(), "id")

	got := f.scim(http.MethodGet, conn, secret, "/Users/"+userID, "")
	require.Equal(t, http.StatusOK, got.Code)
	require.Contains(t, got.Body.String(), "ada@example.com")

	otherOrg, err := models.CreateOrganization(support.RandomName("scim"), "Other")
	require.NoError(t, err)
	otherView, err := f.service.PutConnection(f.ctx(), otherOrg.ID, f.r.Account.ID, connectionInput(true))
	require.NoError(t, err)
	otherConn, err := models.FindSSOConnectionByOrganization(database.Conn(), otherOrg.ID)
	require.NoError(t, err)
	require.Equal(t, otherView.ID, otherConn.ID)
	otherToken, err := f.service.CreateSCIMToken(f.ctx(), otherOrg.ID, f.r.Account.ID, "Other")
	require.NoError(t, err)
	foreign := f.scim(http.MethodGet, conn, otherToken.Secret, "/Users", "")
	require.Equal(t, http.StatusUnauthorized, foreign.Code)
	missing := f.scim(http.MethodGet, otherConn, otherToken.Secret, "/Users/"+userID, "")
	require.Equal(t, http.StatusNotFound, missing.Code)

	byExternal := f.scim(http.MethodGet, conn, secret, "/Users?filter=externalId%20eq%20%22ext-ada%22", "")
	require.Contains(t, byExternal.Body.String(), `"totalResults":1`)
	duplicateExternal := f.scim(http.MethodPost, conn, secret, "/Users", `{"userName":"bea@example.com","externalId":"ext-ada","emails":[{"value":"bea@example.com"}]}`)
	require.Equal(t, http.StatusConflict, duplicateExternal.Code)
	require.Contains(t, duplicateExternal.Body.String(), "uniqueness")

	page := f.scim(http.MethodGet, conn, secret, "/Users?startIndex=1&count=1", "")
	require.Equal(t, http.StatusOK, page.Code)
	require.Contains(t, page.Body.String(), `"itemsPerPage":1`)

	noEmail := f.scim(http.MethodPut, conn, secret, "/Users/"+userID, `{"userName":"not-an-email","active":true}`)
	require.Equal(t, http.StatusBadRequest, noEmail.Code)
	require.Contains(t, noEmail.Body.String(), "invalidValue")

	emptyEmails := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"Operations":[{"op":"replace","path":"emails","value":[]}]}`)
	require.Equal(t, http.StatusBadRequest, emptyEmails.Code)
	require.Contains(t, emptyEmails.Body.String(), "invalidValue")

	renamed := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"operations":[{"op":"replace","path":"name","value":{"givenName":"Ada","familyName":"Lovelace"}},{"op":"add","path":"schemas","value":["ignored"]}]}`)
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	require.Contains(t, renamed.Body.String(), "Ada Lovelace")

	member, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.NoError(t, err)
	require.NoError(t, database.Conn().Create(models.NewUserAPIToken(member.ID, "laptop", "hash")).Error)

	identityID, err := uuid.Parse(userID)
	require.NoError(t, err)
	identity, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identityID)
	require.NoError(t, err)
	identity.Active = false
	require.NoError(t, identity.Save(database.Conn()))
	again := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"OPERATIONS":[{"op":"replace","path":"active","value":false}]}`)
	require.Equal(t, http.StatusOK, again.Code, again.Body.String())
	saved, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identity.ID)
	require.NoError(t, err)
	require.False(t, saved.Active)
	require.NotNil(t, saved.SessionsRevokedAt)
	_, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.Error(t, err)
	var tokens int64
	require.NoError(t, database.Conn().Model(&models.UserAPIToken{}).Where("user_id = ?", member.ID).Count(&tokens).Error)
	require.Zero(t, tokens)

	removed := f.scim(http.MethodDelete, conn, secret, "/Users/"+userID, "")
	require.Equal(t, http.StatusNoContent, removed.Code)

	conn.DefaultRole = models.RoleOrgMaintainer
	require.NoError(t, conn.Save(database.Conn()))
	restored := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"Operations":[{"op":"replace","path":"active","value":true}]}`)
	require.Equal(t, http.StatusOK, restored.Code, restored.Body.String())
	member, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.NoError(t, err)
	roles, err := f.r.AuthService.GetUserRolesForOrg(f.ctx(), member.ID.String(), f.r.Organization.ID.String())
	require.NoError(t, err)
	require.True(t, roleNames(roles, models.RoleOrgMaintainer))

	f.service.Entitlements = licensing.Community
	read := f.scim(http.MethodGet, conn, secret, "/Users/"+userID, "")
	require.Equal(t, http.StatusOK, read.Code)
	off := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"Operations":[{"op":"replace","path":"active","value":false}]}`)
	require.Equal(t, http.StatusOK, off.Code, off.Body.String())
	on := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, `{"Operations":[{"op":"replace","path":"active","value":true}]}`)
	require.Equal(t, http.StatusForbidden, on.Code)
	require.Contains(t, on.Body.String(), "SSO is not licensed for this organization")

	_, err = f.service.DisableConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	disabled := f.scim(http.MethodGet, conn, secret, "/Users", "")
	require.Equal(t, http.StatusForbidden, disabled.Code)
}

func TestSCIMRevokedToken(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	require.NoError(t, f.service.RevokeSCIMToken(f.ctx(), f.r.Organization.ID, created.Token.ID))
	response := f.scim(http.MethodGet, conn, created.Secret, "/Users", "")
	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestSCIMCreateInactiveRemovesTheAdoptedMember(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "ada-subject", "ada@example.com"))
	member, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.NoError(t, err)
	require.NoError(t, database.Conn().Create(models.NewUserAPIToken(member.ID, "laptop", "hash")).Error)
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)

	body := `{"userName":"ada@example.com","active":false,"emails":[{"value":"ada@example.com","primary":true}]}`
	response := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"active":false`)

	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "ada-subject")
	require.NoError(t, err)
	require.False(t, identity.Active)
	require.NotNil(t, identity.SessionsRevokedAt)
	_, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.Error(t, err)
	var tokens int64
	require.NoError(t, database.Conn().Model(&models.UserAPIToken{}).Where("user_id = ?", member.ID).Count(&tokens).Error)
	require.Zero(t, tokens)

	require.NoError(t, f.signIn(conn, "owner-subject", "owner@example.com"))
	ownerBody := `{"userName":"owner@example.com","active":false,"emails":[{"value":"owner@example.com","primary":true}]}`
	ownerResponse := f.scim(http.MethodPost, conn, created.Secret, "/Users", ownerBody)
	require.Equal(t, http.StatusCreated, ownerResponse.Code, ownerResponse.Body.String())
	require.Contains(t, ownerResponse.Body.String(), `"active":false`)
	ownerIdentity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "owner-subject")
	require.NoError(t, err)
	require.False(t, ownerIdentity.Active)
	require.NotNil(t, ownerIdentity.SessionsRevokedAt)
	owner, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), "owner@example.com")
	require.NoError(t, err)
	require.True(t, owner.IsOwner)
}

func TestSCIMInactiveCreateKeepsTheAcceptedUserName(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "ada-subject", "ada@example.com"))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)

	names := []string{"ada@example.com", "ada.other@example.com"}
	codes := make([]int, len(names))
	bodies := make([]string, len(names))
	start := make(chan struct{})
	var group sync.WaitGroup
	for i, name := range names {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			body := `{"userName":"` + name + `","active":false,"emails":[{"value":"ada@example.com","primary":true}]}`
			response := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
			codes[i] = response.Code
			bodies[i] = response.Body.String()
		}()
	}
	close(start)
	group.Wait()

	require.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, codes)
	var accepted string
	for i, code := range codes {
		if code == http.StatusCreated {
			require.Contains(t, bodies[i], `"active":false`)
			accepted = names[i]
		} else {
			require.Contains(t, bodies[i], "uniqueness")
		}
	}
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "ada-subject")
	require.NoError(t, err)
	require.NotNil(t, identity.SCIMUserName)
	require.Equal(t, accepted, *identity.SCIMUserName)
	require.False(t, identity.Active)
	_, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.Error(t, err)
}

func TestSCIMUserNameIsUniqueIgnoringCase(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	other, err := models.CreateAccount("Other", "other@example.com")
	require.NoError(t, err)
	firstName := "Ada@example.com"
	require.NoError(t, database.Conn().Create(&models.SSOIdentity{
		ID: uuid.New(), ConnectionID: conn.ID, OrganizationID: f.r.Organization.ID,
		AccountID: f.r.Account.ID, SCIMUserName: &firstName, Email: "ada@example.com",
		Name: "Ada", Active: true, ProvisionedBy: models.SSOProvisionedBySCIM,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	folded := "ada@example.com"
	err = database.Conn().Create(&models.SSOIdentity{
		ID: uuid.New(), ConnectionID: conn.ID, OrganizationID: f.r.Organization.ID,
		AccountID: other.ID, SCIMUserName: &folded, Email: "other@example.com",
		Name: "Other", Active: true, ProvisionedBy: models.SSOProvisionedBySCIM,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)

	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	duplicate := f.scim(http.MethodPost, conn, created.Secret, "/Users", `{"userName":"ADA@example.com","emails":[{"value":"ada.other@example.com","primary":true}]}`)
	require.Equal(t, http.StatusConflict, duplicate.Code, duplicate.Body.String())
	require.Contains(t, duplicate.Body.String(), "uniqueness")

	second := f.scim(http.MethodPost, conn, created.Secret, "/Users", `{"userName":"bea@example.com","emails":[{"value":"bea@example.com","primary":true}]}`)
	require.Equal(t, http.StatusCreated, second.Code, second.Body.String())
	secondID := jsonString(t, second.Body.String(), "id")
	renamed := f.scim(http.MethodPatch, conn, created.Secret, "/Users/"+secondID, `{"Operations":[{"op":"replace","path":"userName","value":"ADA@example.com"}]}`)
	require.Equal(t, http.StatusConflict, renamed.Code, renamed.Body.String())
	require.Contains(t, renamed.Body.String(), "uniqueness")
	replaced := f.scim(http.MethodPut, conn, created.Secret, "/Users/"+secondID, `{"userName":"ADA@example.com","active":true,"emails":[{"value":"bea@example.com","primary":true}]}`)
	require.Equal(t, http.StatusConflict, replaced.Code, replaced.Body.String())
	require.Contains(t, replaced.Body.String(), "uniqueness")
}

func roleNames(roles []*authorization.RoleDefinition, name string) bool {
	for _, role := range roles {
		if role != nil && role.Name == name {
			return true
		}
	}
	return false
}

func TestSCIMCreateWithoutALicenseAfterUniqueness(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	body := `{"userName":"ada@example.com","active":false,"emails":[{"value":"ada@example.com"}]}`
	first := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	_, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), "ada@example.com")
	require.Error(t, err)

	f.service.Entitlements = licensing.Community
	duplicate := f.scim(http.MethodPost, conn, created.Secret, "/Users", body)
	require.Equal(t, http.StatusConflict, duplicate.Code)
	fresh := f.scim(http.MethodPost, conn, created.Secret, "/Users", `{"userName":"bea@example.com","emails":[{"value":"bea@example.com"}]}`)
	require.Equal(t, http.StatusForbidden, fresh.Code)
}

func TestSCIMCreateConflictsAndSeatLimit(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	_, err = models.CreateAccount("Existing", "existing@other.test")
	require.NoError(t, err)
	conflict := f.scim(http.MethodPost, conn, created.Secret, "/Users", `{"userName":"existing@other.test","emails":[{"value":"existing@other.test"}]}`)
	require.Equal(t, http.StatusConflict, conflict.Code, conflict.Body.String())
	require.Contains(t, conflict.Body.String(), "verified")

	limit := 1
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, &limit))
	seats := f.scim(http.MethodPost, conn, created.Secret, "/Users", `{"userName":"seat@example.com","emails":[{"value":"seat@example.com"}]}`)
	require.Equal(t, http.StatusConflict, seats.Code, seats.Body.String())
	require.Contains(t, seats.Body.String(), "seat")
}

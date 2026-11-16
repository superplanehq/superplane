package sso

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type fakeTXT struct {
	values map[string][]string
}

func (f *fakeTXT) LookupTXT(_ context.Context, name string) ([]string, error) {
	if f.values == nil {
		return nil, nil
	}
	return f.values[name], nil
}

func (f *fakeTXT) set(name string, values ...string) {
	if f.values == nil {
		f.values = map[string][]string{}
	}
	f.values[name] = values
}

type flow struct {
	t       *testing.T
	r       *support.ResourceRegistry
	service *Service
	lookup  *fakeTXT
}

func newFlow(t *testing.T) *flow {
	t.Helper()
	resources := support.Setup(t)
	lookup := &fakeTXT{}
	httpContext, err := registry.NewHTTPContext(registry.HTTPOptions{})
	require.NoError(t, err)
	service := New(Config{
		Encryptor:    resources.Encryptor,
		Entitlements: licensingtest.EnterpriseService(licensing.FeatureSSO),
		Auth:         resources.AuthService,
		Signer:       jwt.NewSigner("test-sso-secret"),
		HTTP:         httpContext,
		BaseURL:      "https://app.example.com",
		Lookup:       lookup,
	})
	return &flow{t: t, r: resources, service: service, lookup: lookup}
}

func (f *flow) ctx() context.Context {
	return context.Background()
}

func connectionInput(jit bool) enterprise.SSOConnectionInput {
	return enterprise.SSOConnectionInput{
		Preset:           models.SSOPresetOkta,
		DisplayName:      "Okta",
		Issuer:           "https://example.okta.com",
		ClientID:         "client-1",
		ClientSecret:     "secret-1",
		JITEnabled:       jit,
		DefaultRole:      models.RoleOrgOperator,
		Enforcement:      models.SSOEnforcementOptional,
		AllowOwnerBypass: true,
	}
}

func (f *flow) put(input enterprise.SSOConnectionInput) *models.SSOConnection {
	f.t.Helper()
	view, err := f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, input)
	require.NoError(f.t, err)
	conn, err := models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	require.Equal(f.t, view.ID, conn.ID)
	require.True(f.t, view.ClientSecretSet)
	require.NotContains(f.t, mustJSON(view), "secret-1")
	return conn
}

func (f *flow) verifyExample() {
	f.t.Helper()
	created, err := f.service.CreateDomain(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Example.com")
	require.NoError(f.t, err)
	require.Equal(f.t, "example.com", created.Domain)
	f.lookup.set(created.TXTName, created.TXTValue)
	verified, err := f.service.VerifyDomain(f.ctx(), f.r.Organization.ID, created.ID)
	require.NoError(f.t, err)
	require.NotNil(f.t, verified.VerifiedAt)
	require.Equal(f.t, "dns", verified.VerificationMethod)
}

func (f *flow) markTested(conn *models.SSOConnection) {
	f.t.Helper()
	now := time.Now()
	raw, err := json.Marshal(models.SSOTestResultDocument{OK: true, Subject: "sub", Email: "test@example.com", Name: "Test"})
	require.NoError(f.t, err)
	conn.LastTestedAt = &now
	conn.LastTestResult = raw
	require.NoError(f.t, conn.Save(database.Conn()))
}

func (f *flow) setOwnerEmail(email string) {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(f.r.Account).Update("email", email).Error)
	f.r.Account.Email = email
	require.NoError(f.t, database.Conn().Model(&models.User{}).Where("id = ?", f.r.User).Update("email", email).Error)
}

func (f *flow) signIn(conn *models.SSOConnection, subject, email string) error {
	f.t.Helper()
	_, err := f.service.pipeline().SignIn(f.ctx(), conn, core.ExternalIdentity{
		Subject: subject,
		Email:   email,
		Name:    "Member",
	}, "/"+f.r.Organization.Slug, nil)
	return err
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func TestIssuerChangeKeepsIdentitiesUntilTheConnectionIsReplaced(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	input := connectionInput(true)
	conn := f.put(input)

	withoutIdentities := input
	withoutIdentities.Issuer = "https://other.okta.com"
	withoutIdentities.ClientSecret = ""
	changed, err := f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, withoutIdentities)
	require.NoError(t, err)
	require.Equal(t, "https://other.okta.com", changed.Issuer)

	original := input
	original.ClientSecret = ""
	_, err = f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, original)
	require.NoError(t, err)

	subject := "same-subject"
	require.NoError(t, database.Conn().Create(&models.SSOIdentity{
		ConnectionID: conn.ID, OrganizationID: f.r.Organization.ID, AccountID: f.r.Account.ID,
		Subject: &subject, Email: "ada@example.com", Active: true,
		ProvisionedBy: models.SSOProvisionedByJIT, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	_, err = f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, withoutIdentities)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Delete this connection and create a new one before you change the issuer.")

	conn, err = models.FindSSOConnectionByID(database.Conn(), conn.ID)
	require.NoError(t, err)
	f.markTested(conn)
	activated, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.SSOStatusActive, activated.Status)

	clientChange := input
	clientChange.ClientID = "client-2"
	clientChange.ClientSecret = ""
	view, err := f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, clientChange)
	require.NoError(t, err)
	require.Equal(t, models.SSOStatusDraft, view.Status)
	require.Equal(t, "client-2", view.ClientID)
}

func TestSaveKeepsCustomClaims(t *testing.T) {
	f := newFlow(t)
	input := connectionInput(true)
	input.Scopes = []string{"openid", "groups"}
	input.SubjectClaim = "uid"
	input.EmailClaim = "mail"
	input.NameClaim = "displayName"
	conn := f.put(input)

	input.ClientSecret = ""
	input.MaxSessionAgeSeconds = 7200
	_, err := f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, input)
	require.NoError(t, err)
	saved, err := models.FindSSOConnectionByID(database.Conn(), conn.ID)
	require.NoError(t, err)
	cfg, err := saved.OIDCConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"openid", "groups"}, cfg.Scopes)
	require.Equal(t, "uid", cfg.SubjectClaim)
	require.Equal(t, "mail", cfg.EmailClaim)
	require.Equal(t, "displayName", cfg.NameClaim)
	require.Equal(t, 7200, saved.MaxSessionAgeSeconds)
}

func TestSignInRefusesAnIdentityDeactivatedBeforeTheLock(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("ada@example.com")
	conn := f.put(connectionInput(true))
	subject := "user-1"
	identity := &models.SSOIdentity{
		ConnectionID: conn.ID, OrganizationID: f.r.Organization.ID, AccountID: f.r.Account.ID,
		Subject: &subject, Email: "ada@example.com", Active: true,
		ProvisionedBy: models.SSOProvisionedByJIT, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, database.Conn().Create(identity).Error)
	f.service.beforeIdentityLock = func(_ *gorm.DB, id uuid.UUID) {
		require.NoError(t, database.Conn().Model(&models.SSOIdentity{}).Where("id = ?", id).Update("active", false).Error)
	}
	err := f.signIn(conn, "user-1", "ada@example.com")
	require.Equal(t, core.CodeAccessDenied, core.CodeOf(err))
	saved, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "user-1")
	require.NoError(t, err)
	require.False(t, saved.Active)
}

func TestConnectionAndDomainRules(t *testing.T) {
	f := newFlow(t)

	_, err := f.service.CreateDomain(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "gmail.com")
	require.Error(t, err)

	f.verifyExample()
	conn := f.put(connectionInput(true))
	_, err = f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.Error(t, err)

	f.markTested(conn)
	activated, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.SSOStatusActive, activated.Status)

	domains, err := f.service.ListDomains(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, domains, 1)
	err = f.service.DeleteDomain(f.ctx(), f.r.Organization.ID, domains[0].ID)
	require.Error(t, err)
}

func TestDomainHeldByAnotherOrganization(t *testing.T) {
	f := newFlow(t)
	created, err := f.service.CreateDomain(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "owned.example")
	require.NoError(t, err)
	f.lookup.set(created.TXTName, created.TXTValue)
	_, err = f.service.VerifyDomain(f.ctx(), f.r.Organization.ID, created.ID)
	require.NoError(t, err)

	other, err := models.CreateOrganization(support.RandomName("org"), "Other")
	require.NoError(t, err)
	otherDomain, err := f.service.CreateDomain(f.ctx(), other.ID, f.r.Account.ID, "owned.example")
	require.NoError(t, err)
	f.lookup.set(otherDomain.TXTName, otherDomain.TXTValue)
	_, err = f.service.VerifyDomain(f.ctx(), other.ID, otherDomain.ID)
	require.Error(t, err)
	_, err = f.service.AdminVerifyDomain(f.ctx(), other.ID, otherDomain.ID, f.r.Account.ID)
	require.Error(t, err)
}

func TestAdminVerifyDomain(t *testing.T) {
	f := newFlow(t)
	created, err := f.service.CreateDomain(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "manual.example")
	require.NoError(t, err)
	verified, err := f.service.AdminVerifyDomain(f.ctx(), f.r.Organization.ID, created.ID, f.r.Account.ID)
	require.NoError(t, err)
	require.Equal(t, "admin", verified.VerificationMethod)
	require.NotNil(t, verified.VerifiedAt)
}

func TestLinkingRules(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	conn := f.put(connectionInput(true))

	t.Run("refuses an existing account when the domain is not verified", func(t *testing.T) {
		_, err := models.CreateAccountInTransaction(database.Conn(), "Other", "other@other.example")
		require.NoError(t, err)
		err = f.signIn(conn, "subject-other", "other@other.example")
		require.Equal(t, core.CodeAccountConflict, core.CodeOf(err))
	})

	t.Run("creates an account when the domain is not verified and nobody owns the email", func(t *testing.T) {
		err := f.signIn(conn, "subject-new-domain", "stranger@stranger.example")
		require.NoError(t, err)
		account, err := models.FindAccountByEmail("stranger@stranger.example")
		require.NoError(t, err)
		require.NotEqual(t, f.r.Account.ID, account.ID)
	})

	f.verifyExample()
	reloaded, err := models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	t.Run("links the account when this organization verified the domain", func(t *testing.T) {
		err := f.signIn(reloaded, "subject-1", "owner@example.com")
		require.NoError(t, err)
		identity, err := models.FindSSOIdentityBySubject(database.Conn(), reloaded.ID, "subject-1")
		require.NoError(t, err)
		require.Equal(t, f.r.Account.ID, identity.AccountID)
		member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "owner@example.com")
		require.NoError(t, err)
		require.Nil(t, member.ProvisionedBy)
	})

	t.Run("updates the identity and keeps the account email", func(t *testing.T) {
		err := f.signIn(reloaded, "subject-1", "renamed@example.com")
		require.NoError(t, err)
		identity, err := models.FindSSOIdentityBySubject(database.Conn(), reloaded.ID, "subject-1")
		require.NoError(t, err)
		require.Equal(t, "renamed@example.com", identity.Email)
		account, err := models.FindAccountByID(f.r.Account.ID.String())
		require.NoError(t, err)
		require.Equal(t, "owner@example.com", account.Email)
	})

	t.Run("creates a member on the first sign-in", func(t *testing.T) {
		err := f.signIn(reloaded, "subject-new", "new.person@example.com")
		require.NoError(t, err)
		member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "new.person@example.com")
		require.NoError(t, err)
		require.NotNil(t, member.ProvisionedBy)
		require.Equal(t, models.SSOProvisionedByJIT, *member.ProvisionedBy)
	})

	t.Run("refuses a blocked account", func(t *testing.T) {
		require.NoError(t, f.r.Account.Block(database.Conn(), time.Now()))
		err := f.signIn(reloaded, "subject-1", "owner@example.com")
		require.Equal(t, core.CodeAccountBlocked, core.CodeOf(err))
	})
}

func TestJITRefusals(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	input := connectionInput(false)
	conn := f.put(input)

	t.Run("refuses a new member when just-in-time is off", func(t *testing.T) {
		err := f.signIn(conn, "jit-off", "jit.off@example.com")
		require.Equal(t, core.CodeAccessDenied, core.CodeOf(err))
	})

	t.Run("refuses a new member when the license is absent", func(t *testing.T) {
		f.service.Entitlements = licensing.Community
		enabled := connectionInput(true)
		_, err := f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, enabled)
		require.Error(t, err)
		saved, err := models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
		require.NoError(t, err)
		saved.JITEnabled = true
		require.NoError(t, saved.Save(database.Conn()))
		err = f.signIn(saved, "unlicensed", "unlicensed@example.com")
		require.Equal(t, core.CodeNotLicensed, core.CodeOf(err))
	})
}

func TestJITSeatLimit(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	limit := 1
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, &limit))
	err := f.signIn(conn, "seat", "seat.limit@example.com")
	require.Equal(t, core.CodeSeatLimit, core.CodeOf(err))
}

func TestJITRestoresARemovedMember(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "restore-me", "restore.me@example.com"))
	member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "restore.me@example.com")
	require.NoError(t, err)
	require.NoError(t, member.SoftDelete(database.Conn(), time.Now(), ""))

	require.NoError(t, f.signIn(conn, "restore-me", "restore.me@example.com"))
	restored, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "restore.me@example.com")
	require.NoError(t, err)
	require.Equal(t, member.ID, restored.ID)
}

func TestDeactivatedIdentityCannotSignIn(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "gone", "gone@example.com"))
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "gone")
	require.NoError(t, err)
	identity.Active = false
	require.NoError(t, identity.Save(database.Conn()))
	member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "gone@example.com")
	require.NoError(t, err)
	require.NoError(t, member.SoftDelete(database.Conn(), time.Now(), ""))

	err = f.signIn(conn, "gone", "gone@example.com")
	require.Equal(t, core.CodeAccessDenied, core.CodeOf(err))
}

func TestInactiveIdentityWithoutMembershipCannotSignIn(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	account, err := models.CreateAccount("Inactive", "inactive.member@example.com")
	require.NoError(t, err)
	now := time.Now()
	subject := "pending"
	require.NoError(t, database.Conn().Create(&models.SSOIdentity{
		ID: uuid.New(), ConnectionID: conn.ID, OrganizationID: conn.OrganizationID, AccountID: account.ID,
		Subject: &subject, Email: account.Email, Active: false, ProvisionedBy: models.SSOProvisionedBySCIM,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	err = f.signIn(conn, "pending", account.Email)
	require.Equal(t, core.CodeAccessDenied, core.CodeOf(err))
	_, err = models.FindActiveUserByEmail(f.r.Organization.ID.String(), account.Email)
	require.Error(t, err)
}

func TestEnforcement(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	input := connectionInput(true)
	input.Enforcement = models.SSOEnforcementRequired
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	owner, err := models.FindActiveUserByID(f.r.Organization.ID.String(), f.r.User.String())
	require.NoError(t, err)
	accountID := f.r.Account.ID

	t.Run("requires SSO for a member with no session entry", func(t *testing.T) {
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{AccountID: accountID})
		require.NoError(t, err)
		require.NotNil(t, required)
		require.Contains(t, required.StartURL, conn.ID.String())
	})

	t.Run("allows the owner when bypass is on", func(t *testing.T) {
		required, err := f.service.CheckOrganizationAccess(f.ctx(), owner, enterprise.SSOAccessSession{AccountID: accountID})
		require.NoError(t, err)
		require.Nil(t, required)
	})

	t.Run("requires SSO for the owner when bypass is off", func(t *testing.T) {
		conn.AllowOwnerBypass = false
		require.NoError(t, conn.Save(database.Conn()))
		required, err := f.service.CheckOrganizationAccess(f.ctx(), owner, enterprise.SSOAccessSession{AccountID: accountID})
		require.NoError(t, err)
		require.NotNil(t, required)
		conn.AllowOwnerBypass = true
		require.NoError(t, conn.Save(database.Conn()))
	})

	t.Run("allows a fresh SSO session", func(t *testing.T) {
		f.setOwnerEmail("owner@example.com")
		require.NoError(t, f.signIn(conn, "owner-subject", "owner@example.com"))
		claim := conn.ID.String() + ":" + strconv.FormatInt(time.Now().Unix(), 10)
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{
			AccountID: accountID,
			SSO:       claim,
		})
		require.NoError(t, err)
		require.Nil(t, required)
	})

	t.Run("requires SSO when the session is older than the maximum age", func(t *testing.T) {
		claim := conn.ID.String() + ":" + strconv.FormatInt(time.Now().Add(-48*time.Hour).Unix(), 10)
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{
			AccountID: accountID,
			SSO:       claim,
		})
		require.NoError(t, err)
		require.NotNil(t, required)
	})

	t.Run("requires SSO when the identity session was revoked", func(t *testing.T) {
		identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "owner-subject")
		require.NoError(t, err)
		revoked := time.Now()
		identity.SessionsRevokedAt = &revoked
		require.NoError(t, identity.Save(database.Conn()))
		claim := conn.ID.String() + ":" + strconv.FormatInt(revoked.Add(-time.Minute).Unix(), 10)
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{
			AccountID: accountID,
			SSO:       claim,
		})
		require.NoError(t, err)
		require.NotNil(t, required)
		identity.SessionsRevokedAt = nil
		require.NoError(t, identity.Save(database.Conn()))
	})

	t.Run("requires SSO when the session entry has no identity", func(t *testing.T) {
		claim := conn.ID.String() + ":" + strconv.FormatInt(time.Now().Unix(), 10)
		member := *owner
		member.IsOwner = false
		missing := uuid.New()
		member.AccountID = &missing
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{
			AccountID: missing,
			SSO:       claim,
		})
		require.NoError(t, err)
		require.NotNil(t, required)
	})

	t.Run("requires SSO when the identity is inactive", func(t *testing.T) {
		identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "owner-subject")
		require.NoError(t, err)
		identity.Active = false
		require.NoError(t, identity.Save(database.Conn()))
		claim := conn.ID.String() + ":" + strconv.FormatInt(time.Now().Unix(), 10)
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{
			AccountID: accountID,
			SSO:       claim,
		})
		require.NoError(t, err)
		require.NotNil(t, required)
	})

	t.Run("allows access when enforcement is optional", func(t *testing.T) {
		conn.Enforcement = models.SSOEnforcementOptional
		require.NoError(t, conn.Save(database.Conn()))
		member := *owner
		member.IsOwner = false
		required, err := f.service.CheckOrganizationAccess(f.ctx(), &member, enterprise.SSOAccessSession{AccountID: accountID})
		require.NoError(t, err)
		require.Nil(t, required)
	})
}

func TestSignInDoesNotOpenAnotherOrganization(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	conn := f.put(connectionInput(true))
	require.NoError(t, f.signIn(conn, "isolated", "isolated@example.com"))

	other, err := models.CreateOrganization(support.RandomName("org"), "Other")
	require.NoError(t, err)
	_, err = models.FindActiveUserByEmailInTransaction(database.Conn(), other.ID.String(), "isolated@example.com")
	require.Error(t, err)
}

func TestRecordTestDoesNotRestoreADisabledConnection(t *testing.T) {
	f := newFlow(t)
	f.verifyExample()
	f.put(connectionInput(true))
	loaded, err := models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	_, err = f.service.DisableConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)

	require.NoError(t, f.service.recordTest(f.ctx(), loaded, true, "", "subject", "ada@example.com", "Ada"))

	saved, err := models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.SSOStatusDisabled, saved.Status)
	require.NotNil(t, saved.LastTestedAt)
	require.True(t, saved.UpdatedAt.After(loaded.UpdatedAt))
	result, err := saved.TestResult()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.OK)
}

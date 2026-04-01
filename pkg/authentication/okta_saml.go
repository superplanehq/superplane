package authentication

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

// handleOktaSAMLLogin starts the SP-initiated SAML flow.
// GET /auth/okta/{org_id}/saml/login
func (a *Handler) handleOktaSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if a.publicAppBaseURL == "" {
		log.Error("Okta SAML: public base URL is not configured")
		http.Error(w, "authentication is not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orgID := strings.TrimSpace(vars["org_id"])
	if _, err := uuid.Parse(orgID); err != nil {
		http.Error(w, "invalid organization", http.StatusBadRequest)
		return
	}

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !org.IsProviderAllowed(models.ProviderOkta) {
		http.Error(w, "Okta sign-in is not enabled for this organization", http.StatusForbidden)
		return
	}

	idp, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Okta is not configured", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !idp.SamlEnabled {
		http.Error(w, "Okta SAML is disabled", http.StatusForbidden)
		return
	}
	if idp.SamlIdpCertificatePEM == "" || idp.SamlIdpSSOURL == "" {
		http.Error(w, "Okta SAML is not fully configured", http.StatusServiceUnavailable)
		return
	}

	redirect := strings.TrimSpace(r.URL.Query().Get("redirect"))
	if redirect != "" && !isValidRedirectURL(redirect) {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}

	sp, err := buildSAMLServiceProvider(idp, orgID, a.publicAppBaseURL)
	if err != nil {
		log.Errorf("Okta SAML: build service provider: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	authnReq, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		log.Errorf("Okta SAML: make authn request: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Force Okta to re-authenticate the user even if they have an active IdP
	// session. Without this, logging out of the app and logging back in would
	// silently reuse the Okta session without any credential or MFA prompt.
	forceAuthn := true
	authnReq.ForceAuthn = &forceAuthn

	relayState, err := a.jwtSigner.SignOktaSAMLState(orgID, authnReq.ID, redirect, 15*time.Minute)
	if err != nil {
		log.Errorf("Okta SAML: sign relay state: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	redirectURL, err := authnReq.Redirect(relayState, &sp)
	if err != nil {
		log.Errorf("Okta SAML: build redirect URL: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}

// handleOktaSAMLACS handles the SAML Assertion Consumer Service (ACS) endpoint.
// POST /auth/okta/{org_id}/saml/acs
func (a *Handler) handleOktaSAMLACS(w http.ResponseWriter, r *http.Request) {
	if a.publicAppBaseURL == "" {
		http.Error(w, "authentication is not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orgID := strings.TrimSpace(vars["org_id"])
	if _, err := uuid.Parse(orgID); err != nil {
		http.Error(w, "invalid organization", http.StatusBadRequest)
		return
	}

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !org.IsProviderAllowed(models.ProviderOkta) {
		http.Error(w, "Okta sign-in is not enabled for this organization", http.StatusForbidden)
		return
	}

	idp, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Okta is not configured", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !idp.SamlEnabled {
		http.Error(w, "Okta SAML is disabled", http.StatusForbidden)
		return
	}

	relayStateRaw := r.FormValue("RelayState")
	stateOrg, requestID, redirect, err := a.jwtSigner.ParseOktaSAMLState(relayStateRaw)
	if err != nil || stateOrg != orgID {
		http.Error(w, "invalid or expired relay state", http.StatusBadRequest)
		return
	}
	if redirect != "" && !isValidRedirectURL(redirect) {
		redirect = "/"
	}
	if redirect == "" {
		redirect = "/"
	}

	sp, err := buildSAMLServiceProvider(idp, orgID, a.publicAppBaseURL)
	if err != nil {
		log.Errorf("Okta SAML: build service provider: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	assertion, err := sp.ParseResponse(r, []string{requestID})
	if err != nil {
		log.Warnf("Okta SAML [%s]: ParseResponse failed: %v", orgID, err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	if assertion.Subject == nil || assertion.Subject.NameID == nil {
		http.Error(w, "missing NameID in SAML assertion", http.StatusUnauthorized)
		return
	}

	email := utils.NormalizeEmail(strings.TrimSpace(assertion.Subject.NameID.Value))
	if email == "" {
		http.Error(w, "email claim is required", http.StatusUnauthorized)
		return
	}

	user, err := models.FindActiveUserByEmail(orgID, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "no SuperPlane user exists for this email in the organization", http.StatusForbidden)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user.IsServiceAccount() {
		http.Error(w, "unsupported account type", http.StatusForbidden)
		return
	}
	if user.AccountID == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, user.ID.String()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "sign-in requires an account provisioned for this organization", http.StatusForbidden)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	account, err := models.FindAccountByID(user.AccountID.String())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	displayName := samlDisplayName(assertion, user.Name)

	if err := upsertOktaSAMLAccountProvider(a.encryptor, account, idp.SamlIdpIssuer, email, displayName); err != nil {
		log.Errorf("Okta SAML: account provider: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.acceptPendingInvitations(account); err != nil {
		log.Errorf("Okta SAML: invitations: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.issueAccountSessionForSAMLOrg(w, r, account.ID.String(), orgID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// buildSAMLServiceProvider constructs a crewjam/saml ServiceProvider from stored IdP config.
func buildSAMLServiceProvider(idp *models.OrganizationOktaIDP, orgID, baseURL string) (saml.ServiceProvider, error) {
	acsURLStr := baseURL + "/auth/okta/" + orgID + "/saml/acs"
	acsURL, err := url.Parse(acsURLStr)
	if err != nil {
		return saml.ServiceProvider{}, fmt.Errorf("invalid ACS URL: %w", err)
	}

	certData, err := pemToBase64DER(idp.SamlIdpCertificatePEM)
	if err != nil {
		return saml.ServiceProvider{}, fmt.Errorf("invalid IdP certificate: %w", err)
	}

	entityID := baseURL + "/auth/okta/" + orgID

	sp := saml.ServiceProvider{
		EntityID: entityID,
		AcsURL:   *acsURL,
		IDPMetadata: &saml.EntityDescriptor{
			EntityID: idp.SamlIdpIssuer,
			IDPSSODescriptors: []saml.IDPSSODescriptor{
				{
					SSODescriptor: saml.SSODescriptor{
						RoleDescriptor: saml.RoleDescriptor{
							KeyDescriptors: []saml.KeyDescriptor{
								{
									Use: "signing",
									KeyInfo: saml.KeyInfo{
										X509Data: saml.X509Data{
											X509Certificates: []saml.X509Certificate{
												{Data: certData},
											},
										},
									},
								},
							},
						},
					},
					SingleSignOnServices: []saml.Endpoint{
						{
							Binding:  saml.HTTPRedirectBinding,
							Location: idp.SamlIdpSSOURL,
						},
					},
				},
			},
		},
	}
	return sp, nil
}

// pemToBase64DER strips PEM headers and returns the raw base64-encoded DER data,
// which is what crewjam/saml expects in X509Certificate.Data.
func pemToBase64DER(pemStr string) (string, error) {
	pemStr = strings.TrimSpace(pemStr)

	// If the cert was stored without PEM headers (raw base64), return as-is after
	// verifying it decodes as valid DER.
	if !strings.HasPrefix(pemStr, "-----") {
		if _, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(pemStr, "\n", "")); err != nil {
			return "", fmt.Errorf("not a valid PEM or base64 DER certificate")
		}
		return strings.ReplaceAll(pemStr, "\n", ""), nil
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return "", fmt.Errorf("invalid X.509 certificate: %w", err)
	}
	return base64.StdEncoding.EncodeToString(block.Bytes), nil
}

// samlDisplayName extracts a display name from SAML assertion attributes.
// Falls back to the existing user name.
func samlDisplayName(assertion *saml.Assertion, fallback string) string {
	if assertion.AttributeStatements == nil {
		return fallback
	}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			switch attr.Name {
			case "displayName", "urn:oid:2.16.840.1.113730.3.1.241":
				if len(attr.Values) > 0 && attr.Values[0].Value != "" {
					return attr.Values[0].Value
				}
			}
		}
	}
	// Try combining firstName + lastName
	var first, last string
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			switch attr.Name {
			case "firstName", "givenName", "urn:oid:2.5.4.42":
				if len(attr.Values) > 0 {
					first = attr.Values[0].Value
				}
			case "lastName", "sn", "urn:oid:2.5.4.4":
				if len(attr.Values) > 0 {
					last = attr.Values[0].Value
				}
			}
		}
	}
	if combined := strings.TrimSpace(first + " " + last); combined != "" {
		return combined
	}
	return fallback
}

// upsertOktaSAMLAccountProvider stores or updates the Okta SAML account provider row.
func upsertOktaSAMLAccountProvider(
	encryptor crypto.Encryptor,
	account *models.Account,
	samlIssuer, email, name string,
) error {
	// Derive a stable provider ID from issuer + email (SAML NameID is email).
	providerUserID := crypto.HashToken(strings.TrimSuffix(samlIssuer, "/") + "\x00" + email)
	// Store a placeholder token (SAML does not issue access tokens).
	placeholderTok, err := encryptor.Encrypt(context.Background(), []byte("-"), []byte(email))
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(placeholderTok)

	var existing models.AccountProvider
	q := database.Conn().Where("account_id = ? AND provider = ?", account.ID, models.ProviderOkta).First(&existing)
	if q.Error == nil {
		existing.ProviderID = providerUserID
		existing.Username = email
		existing.Email = email
		existing.Name = name
		existing.AvatarURL = ""
		existing.AccessToken = enc
		return database.Conn().Save(&existing).Error
	}
	if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return q.Error
	}

	ap := &models.AccountProvider{
		AccountID:   account.ID,
		Provider:    models.ProviderOkta,
		ProviderID:  providerUserID,
		Username:    email,
		Email:       email,
		Name:        name,
		AvatarURL:   "",
		AccessToken: enc,
	}
	return database.Conn().Create(ap).Error
}

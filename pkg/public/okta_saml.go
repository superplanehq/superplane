package public

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
)

func (s *Server) registerOktaRoutes(r *mux.Router) {
	publicRoute := r.Methods(http.MethodPost).Subrouter()

	publicRoute.HandleFunc("/orgs/{orgId}/okta/auth", s.handleOktaSAML).Methods(http.MethodPost)
}

func (s *Server) handleOktaSAML(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	orgID := vars["orgId"]

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		log.Errorf("Okta SAML: organization %s not found: %v", orgID, err)
		http.Error(w, "organization not found", http.StatusNotFound)
		return
	}

	config, err := models.FindOrganizationOktaConfig(org.ID)
	if err != nil {
		log.Errorf("Okta SAML: Okta config for org %s not found: %v", org.ID, err)
		http.Error(w, "Okta not configured for organization", http.StatusForbidden)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		http.Error(w, "missing SAMLResponse", http.StatusBadRequest)
		return
	}

	certStore, err := buildCertificateStore(config.SamlCertificate)
	if err != nil {
		log.Errorf("Okta SAML: invalid certificate for org %s: %v", org.ID, err)
		http.Error(w, "invalid configuration", http.StatusInternalServerError)
		return
	}

	sp := &saml2.SAMLServiceProvider{
		IdentityProviderIssuer: config.SamlIssuer,
		ServiceProviderIssuer:  s.BaseURL + "/orgs/" + org.ID.String() + "/okta/auth",
		AudienceURI:            s.BaseURL + "/orgs/" + org.ID.String() + "/okta/auth",
		IDPCertificateStore:    certStore,
	}

	assertionInfo, err := sp.RetrieveAssertionInfo(samlResponse)
	if err != nil {
		log.Errorf("Okta SAML: error retrieving assertion for org %s: %v", org.ID, err)
		http.Error(w, "invalid SAML assertion", http.StatusForbidden)
		return
	}

	if assertionInfo.WarningInfo.InvalidTime || assertionInfo.WarningInfo.NotInAudience {
		log.Errorf("Okta SAML: assertion warnings for org %s: %+v", org.ID, assertionInfo.WarningInfo)
		http.Error(w, "invalid SAML assertion", http.StatusForbidden)
		return
	}

	email := assertionInfo.Values.Get("email")
	if email == "" {
		http.Error(w, "email claim missing", http.StatusForbidden)
		return
	}

	account, err := models.FindAccountByEmail(email)
	if err != nil {
		log.Errorf("Okta SAML: account for email %s not found: %v", email, err)
		http.Error(w, "account not found", http.StatusForbidden)
		return
	}

	user, err := models.FindActiveUserByEmail(org.ID.String(), account.Email)
	if err != nil {
		log.Errorf("Okta SAML: user for email %s and org %s not found: %v", account.Email, org.ID, err)
		http.Error(w, "user not found in organization", http.StatusForbidden)
		return
	}

	token, err := s.jwt.Generate(account.ID.String(), 24*60*60*1e9)
	if err != nil {
		log.Errorf("Okta SAML: error generating token for account %s: %v", account.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/"+user.OrganizationID.String(), http.StatusTemporaryRedirect)
}

func buildCertificateStore(pemCert string) (*dsig.MemoryX509CertificateStore, error) {
	block, _ := pem.Decode([]byte(pemCert))
	if block == nil {
		// try raw base64 DER
		certData, err := base64.StdEncoding.DecodeString(pemCert)
		if err != nil {
			return nil, err
		}
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return nil, err
		}

		return &dsig.MemoryX509CertificateStore{
			Roots: []*x509.Certificate{cert},
		}, nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{cert},
	}, nil
}

func hashSCIMToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(sum[:])
}

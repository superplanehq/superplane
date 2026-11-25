package sso

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ssooidc "github.com/superplanehq/superplane/ee/sso/oidc"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

var errGlobalReplay = errors.New("global token revocation replay")

func (s *Service) globalTokenRevocation(w http.ResponseWriter, r *http.Request) {
	if !s.limits.allow(middleware.ClientIP(r)) {
		http.Error(w, "Too many requests.", http.StatusTooManyRequests)
		return
	}
	conn, status, reason := s.globalRevocationConnection(r)
	if status != 0 {
		s.finishGlobalRevocation(w, r, conn, status, reason, "", false)
		return
	}
	raw, ok := bearerJWT(r.Header.Get("Authorization"))
	if !ok {
		s.finishGlobalRevocation(w, r, conn, http.StatusUnauthorized, "auth", "", false)
		return
	}
	audience := s.BaseURL + "/auth/sso/" + conn.ID.String() + "/global-token-revocation"
	jti, expiresAt, err := s.adapter.VerifyGlobalRevocation(r.Context(), conn, raw, audience)
	if err != nil {
		s.finishGlobalRevocation(w, r, conn, http.StatusUnauthorized, "auth", revocationDetail(err), false)
		return
	}
	identities, status, reason := globalRevocationIdentities(w, r, conn)
	if status != 0 {
		s.finishGlobalRevocation(w, r, conn, status, reason, "", false)
		return
	}
	if len(identities) == 0 {
		s.finishGlobalRevocation(w, r, conn, http.StatusNotFound, "", "", false)
		return
	}
	now := s.now()
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		fresh, err := models.ConsumeSSOLogoutReplay(tx, conn.ID, jti, expiresAt.Add(2*time.Minute))
		if err != nil {
			return err
		}
		if !fresh {
			return errGlobalReplay
		}
		for i := range identities {
			if err := identities[i].RevokeSessions(tx, now); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errGlobalReplay) {
		s.finishGlobalRevocation(w, r, conn, http.StatusUnauthorized, "replay", "", false)
		return
	}
	if err != nil {
		s.finishGlobalRevocation(w, r, conn, http.StatusUnprocessableEntity, "", "", false)
		return
	}
	s.finishGlobalRevocation(w, r, conn, http.StatusNoContent, "", "", true)
}

func revocationDetail(err error) string {
	var failure *ssooidc.RevocationError
	if errors.As(err, &failure) {
		return failure.Detail
	}
	return ""
}

func (s *Service) globalRevocationConnection(r *http.Request) (*models.SSOConnection, int, string) {
	connectionID, err := uuid.Parse(mux.Vars(r)["connectionID"])
	if err != nil {
		return nil, http.StatusUnauthorized, "auth"
	}
	conn, err := models.FindSSOConnectionByID(database.DB(r.Context()), connectionID)
	if err != nil || conn.Protocol != models.SSOProtocolOIDC {
		return conn, http.StatusUnauthorized, "auth"
	}
	cfg, err := conn.OIDCConfig()
	if err != nil || strings.TrimSpace(cfg.Issuer) == "" {
		return conn, http.StatusUnauthorized, "auth"
	}
	return conn, 0, ""
}

func globalRevocationIdentities(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) ([]models.SSOIdentity, int, string) {
	if !jsonContent(r) {
		return nil, http.StatusBadRequest, "body"
	}
	var body struct {
		SubID struct {
			Format string `json:"format"`
			Iss    string `json:"iss"`
			Sub    string `json:"sub"`
			Email  string `json:"email"`
		} `json:"sub_id"`
	}
	reader := http.MaxBytesReader(w, r.Body, 1<<16)
	if err := json.NewDecoder(reader).Decode(&body); err != nil {
		return nil, http.StatusBadRequest, "body"
	}
	switch body.SubID.Format {
	case "iss_sub":
		return identitiesByIssuerSubject(r, conn, body.SubID.Iss, body.SubID.Sub)
	case "email":
		return identitiesByEmail(r, conn, body.SubID.Email)
	default:
		return nil, http.StatusBadRequest, "subject_format"
	}
}

func identitiesByIssuerSubject(r *http.Request, conn *models.SSOConnection, issuer, subject string) ([]models.SSOIdentity, int, string) {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(subject) == "" {
		return nil, http.StatusBadRequest, "body"
	}
	cfg, err := conn.OIDCConfig()
	if err != nil || !sameIssuer(cfg.Issuer, issuer) {
		return nil, http.StatusBadRequest, "issuer"
	}
	identity, err := models.FindSSOIdentityByIDPSubject(database.DB(r.Context()), conn.ID, subject)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, ""
	}
	if err != nil {
		return nil, http.StatusUnprocessableEntity, ""
	}
	return []models.SSOIdentity{*identity}, 0, ""
}

func identitiesByEmail(r *http.Request, conn *models.SSOConnection, email string) ([]models.SSOIdentity, int, string) {
	if strings.TrimSpace(email) == "" {
		return nil, http.StatusBadRequest, "body"
	}
	identities, err := models.ListSSOIdentitiesByEmail(database.DB(r.Context()), conn.ID, email)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, ""
	}
	return identities, 0, ""
}

func (s *Service) finishGlobalRevocation(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection, status int, reason, detail string, found bool) {
	outcome := "failure"
	switch status {
	case http.StatusNoContent:
		outcome = "success"
	case http.StatusNotFound:
		outcome = "not_found"
	}
	fields := map[string]any{
		"action":                    "sso_global_token_revocation",
		"outcome":                   outcome,
		"sso_logout_identity_found": found,
	}
	if reason != "" {
		fields["sso_failure_reason"] = reason
	}
	if detail != "" {
		fields["sso_failure_detail"] = detail
	}
	if id := mux.Vars(r)["connectionID"]; id != "" {
		if _, err := uuid.Parse(id); err == nil {
			fields["sso_connection_id"] = id
		}
	}
	if conn != nil && conn.OrganizationID != uuid.Nil {
		fields["organization_id"] = conn.OrganizationID.String()
	}
	logging.AddRequestLogFields(r.Context(), fields)
	w.WriteHeader(status)
}

func bearerJWT(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.Contains(token, " ") {
		return "", false
	}
	return token, true
}

func jsonContent(r *http.Request) bool {
	value := r.Header.Get("Content-Type")
	if value == "" {
		return false
	}
	media, _, err := mime.ParseMediaType(value)
	return err == nil && media == "application/json"
}

func sameIssuer(left, right string) bool {
	return strings.TrimRight(strings.TrimSpace(left), "/") == strings.TrimRight(strings.TrimSpace(right), "/")
}

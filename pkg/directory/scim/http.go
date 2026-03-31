package scim

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

const scimURLPrefix = "/api/v1/scim/"

// RegisterRoutes mounts SCIM 2.0 under /api/v1/scim/{organization_id}/v2/... .
func RegisterRoutes(r *mux.Router, auth authorization.Authorization) {
	userSchema := schema.CoreUserSchema()
	srv, err := scim.NewServer(&scim.ServerArgs{
		ServiceProviderConfig: &scim.ServiceProviderConfig{
			MaxResults:       100,
			SupportFiltering: false,
			SupportPatch:     true,
		},
		ResourceTypes: []scim.ResourceType{
			{
				ID:          optional.NewString("User"),
				Name:        "User",
				Endpoint:    "/Users",
				Description: optional.NewString("User Account"),
				Schema:      userSchema,
				Handler:     &UserHandler{Auth: auth},
			},
		},
	})
	if err != nil {
		log.Fatalf("scim server: %v", err)
	}

	r.PathPrefix(scimURLPrefix).Handler(scimBearerAndOrgMiddleware(&srv))
}

func scimError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	body := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"status":  fmt.Sprintf("%d", status),
		"detail":  detail,
	}
	_ = json.NewEncoder(w).Encode(body)
}

func scimBearerAndOrgMiddleware(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, scimURLPrefix) {
			http.NotFound(w, r)
			return
		}

		rest := strings.TrimPrefix(r.URL.Path, scimURLPrefix)
		orgID, suffix, ok := strings.Cut(rest, "/")
		if !ok || orgID == "" || suffix == "" {
			scimError(w, http.StatusBadRequest, "invalid SCIM path")
			return
		}

		row, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
		if err != nil {
			log.Warnf("SCIM [%s] %s %s: org not found or SCIM not configured", orgID, r.Method, r.URL.Path)
			scimError(w, http.StatusNotFound, "SCIM not configured")
			return
		}
		if !row.ScimEnabled || row.ScimBearerTokenHash == nil || *row.ScimBearerTokenHash == "" {
			log.Warnf("SCIM [%s] %s %s: SCIM disabled or token not set", orgID, r.Method, r.URL.Path)
			scimError(w, http.StatusForbidden, "SCIM disabled")
			return
		}

		authz := r.Header.Get("Authorization")
		const bearer = "Bearer "
		if len(authz) < len(bearer) || !strings.EqualFold(authz[:len(bearer)], bearer) {
			log.Warnf("SCIM [%s] %s %s: missing or malformed Authorization header", orgID, r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			scimError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		raw := strings.TrimSpace(authz[len(bearer):])
		if raw == "" {
			log.Warnf("SCIM [%s] %s %s: empty bearer token", orgID, r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			scimError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		want := []byte(*row.ScimBearerTokenHash)
		got := []byte(crypto.HashToken(raw))
		if len(want) != len(got) || subtle.ConstantTimeCompare(want, got) != 1 {
			log.Warnf("SCIM [%s] %s %s: bearer token mismatch", orgID, r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			scimError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		log.Infof("SCIM [%s] %s %s: authenticated", orgID, r.Method, r.URL.Path)

		r = r.Clone(WithOrganizationID(r.Context(), orgID))
		r2 := *r
		u := *r.URL
		u.Path = "/" + suffix
		r2.URL = &u
		inner.ServeHTTP(w, &r2)
	})
}

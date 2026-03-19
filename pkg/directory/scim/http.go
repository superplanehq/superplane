package scim

import (
	"crypto/subtle"
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

func scimBearerAndOrgMiddleware(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, scimURLPrefix) {
			http.NotFound(w, r)
			return
		}

		rest := strings.TrimPrefix(r.URL.Path, scimURLPrefix)
		orgID, suffix, ok := strings.Cut(rest, "/")
		if !ok || orgID == "" || suffix == "" {
			http.Error(w, "invalid SCIM path", http.StatusBadRequest)
			return
		}

		row, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
		if err != nil {
			http.Error(w, "SCIM not configured", http.StatusNotFound)
			return
		}
		if !row.ScimEnabled || row.ScimBearerTokenHash == nil || *row.ScimBearerTokenHash == "" {
			http.Error(w, "SCIM disabled", http.StatusForbidden)
			return
		}

		authz := r.Header.Get("Authorization")
		const bearer = "Bearer "
		if len(authz) < len(bearer) || !strings.EqualFold(authz[:len(bearer)], bearer) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		raw := strings.TrimSpace(authz[len(bearer):])
		if raw == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		want := []byte(*row.ScimBearerTokenHash)
		got := []byte(crypto.HashToken(raw))
		if len(want) != len(got) || subtle.ConstantTimeCompare(want, got) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="SCIM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r = r.Clone(WithOrganizationID(r.Context(), orgID))
		r2 := *r
		u := *r.URL
		u.Path = "/" + suffix
		r2.URL = &u
		inner.ServeHTTP(w, &r2)
	})
}

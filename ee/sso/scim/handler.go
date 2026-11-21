package scim

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	schemaUser  = "urn:ietf:params:scim:schemas:core:2.0:User"
	schemaError = "urn:ietf:params:scim:api:messages:2.0:Error"
	schemaList  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	notLicensed = "SSO is not licensed for this organization"
)

// Handler serves SCIM 2.0 Users for one connection path.
type Handler struct {
	pipeline *core.Pipeline
	baseURL  string
}

// NewHandler returns the /scim/v2 router.
func NewHandler(pipeline *core.Pipeline, baseURL string) http.Handler {
	handler := &Handler{pipeline: pipeline, baseURL: strings.TrimRight(baseURL, "/")}
	router := mux.NewRouter()
	router.PathPrefix("/scim/v2/{connectionID}/Groups").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "", "Groups are not supported.")
	})
	router.HandleFunc("/scim/v2/{connectionID}/ServiceProviderConfig", handler.serviceProviderConfig).Methods(http.MethodGet)
	router.HandleFunc("/scim/v2/{connectionID}/ResourceTypes", handler.resourceTypes).Methods(http.MethodGet)
	router.HandleFunc("/scim/v2/{connectionID}/Schemas", handler.schemas).Methods(http.MethodGet)
	router.HandleFunc("/scim/v2/{connectionID}/Users", handler.users).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/scim/v2/{connectionID}/Users/{userID}", handler.user).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "", "This resource does not exist.")
	})
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusMethodNotAllowed, "", "This method is not supported.")
	})
	return router
}

func (h *Handler) now() time.Time {
	if h.pipeline != nil && h.pipeline.Now != nil {
		return h.pipeline.Now()
	}
	return time.Now()
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (*models.SSOConnection, bool) {
	connectionID, err := uuid.Parse(mux.Vars(r)["connectionID"])
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "", "The SCIM token is not valid.")
		return nil, false
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, r, http.StatusUnauthorized, "", "The SCIM token is not valid.")
		return nil, false
	}
	stored, err := models.FindActiveSCIMTokenByHash(database.DB(r.Context()), crypto.HashToken(token))
	if err != nil || stored.ConnectionID != connectionID {
		writeError(w, r, http.StatusUnauthorized, "", "The SCIM token is not valid.")
		return nil, false
	}
	conn, err := models.FindSSOConnectionByID(database.DB(r.Context()), connectionID)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "", "The SCIM token is not valid.")
		return nil, false
	}
	if conn.Status != models.SSOStatusActive && conn.Status != models.SSOStatusDraft {
		writeError(w, r, http.StatusForbidden, "", "Single Sign-On is disabled for this organization.")
		return nil, false
	}
	_ = models.TouchSCIMTokenLastUsed(database.DB(r.Context()), stored.ID, h.now())
	return conn, true
}

func (h *Handler) serviceProviderConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":               []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":                 map[string]any{"supported": true},
		"bulk":                  map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":                map[string]any{"supported": true, "maxResults": 200},
		"changePassword":        map[string]any{"supported": false},
		"sort":                  map[string]any{"supported": false},
		"etag":                  map[string]any{"supported": false},
		"authenticationSchemes": []any{},
	})
}

func (h *Handler) resourceTypes(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{schemaList},
		"totalResults": 1,
		"Resources": []any{map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":          "User",
			"name":        "User",
			"endpoint":    "/Users",
			"schema":      schemaUser,
			"description": "User Account",
		}},
	})
}

func (h *Handler) schemas(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{schemaList},
		"totalResults": 1,
		"Resources": []any{map[string]any{
			"id":   schemaUser,
			"name": "User",
			"attributes": []any{
				map[string]any{"name": "userName", "type": "string", "required": true},
				map[string]any{"name": "name", "type": "complex"},
				map[string]any{"name": "displayName", "type": "string"},
				map[string]any{"name": "emails", "type": "complex", "multiValued": true},
				map[string]any{"name": "active", "type": "boolean"},
				map[string]any{"name": "externalId", "type": "string"},
			},
		}},
	})
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		h.listUsers(w, r, conn)
		return
	}
	h.createUser(w, r, conn)
}

func (h *Handler) user(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getUser(w, r, conn)
	case http.MethodPut:
		h.replaceUser(w, r, conn)
	case http.MethodPatch:
		h.patchUser(w, r, conn)
	case http.MethodDelete:
		h.deleteUser(w, r, conn)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "", "This method is not supported.")
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, scimType, detail string) {
	if r != nil {
		logSCIM(r, "failure", detail)
	}
	body := map[string]any{
		"schemas": []string{schemaError},
		"status":  strconv.Itoa(status),
		"detail":  detail,
	}
	if scimType != "" {
		body["scimType"] = scimType
	}
	writeJSON(w, status, body)
}

func logSCIM(r *http.Request, outcome, reason string) {
	fields := map[string]any{
		"action":  scimAction(r),
		"object":  "sso_identity",
		"outcome": outcome,
	}
	if outcome == "failure" && reason != "" {
		fields["sso_failure_reason"] = reason
	}
	if id := mux.Vars(r)["connectionID"]; id != "" {
		fields["sso_connection_id"] = id
		if parsed, err := uuid.Parse(id); err == nil {
			if conn, err := models.FindSSOConnectionByID(database.DB(r.Context()), parsed); err == nil {
				fields["organization_id"] = conn.OrganizationID.String()
			}
		}
	}
	logging.AddRequestLogFields(r.Context(), fields)
}

func scimAction(r *http.Request) string {
	if !strings.Contains(r.URL.Path, "/Users") {
		return "scim_request"
	}
	switch r.Method {
	case http.MethodPost:
		return "scim_user_create"
	case http.MethodDelete:
		return "scim_user_deactivate"
	case http.MethodGet:
		return "scim_user_read"
	default:
		return "scim_user_update"
	}
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, 1<<20))
}

func acceptJSON(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return r.Method == http.MethodGet || r.Method == http.MethodDelete
	}
	media := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return media == "application/scim+json" || media == "application/json"
}

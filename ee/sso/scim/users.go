package scim

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

var filterPattern = regexp.MustCompile(`(?i)^\s*(userName|externalId)\s+eq\s+(?:"([^"]*)"|'([^']*)')\s*$`)

type scimName struct {
	Formatted  string `json:"formatted"`
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type scimEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type scimUserBody struct {
	UserName    string      `json:"userName"`
	ExternalID  string      `json:"externalId"`
	DisplayName string      `json:"displayName"`
	Name        *scimName   `json:"name"`
	Emails      []scimEmail `json:"emails"`
	Active      any         `json:"active"`
}

type profileChange struct {
	UserName   *string
	ExternalID *string
	Name       *string
	Email      *string
	Active     *bool
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	attr, value, ok := parseFilter(r.URL.Query().Get("filter"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalidFilter", "This filter is not supported.")
		return
	}
	start, count, ok := page(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalidValue", "The page values are not valid.")
		return
	}
	rows, total, err := models.ListSSOIdentities(database.DB(r.Context()), conn.ID, attr, value, start-1, count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "", "The request failed.")
		return
	}
	resources := make([]any, 0, len(rows))
	for _, row := range rows {
		resources = append(resources, h.userResource(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{schemaList},
		"totalResults": total,
		"startIndex":   start,
		"itemsPerPage": len(resources),
		"Resources":    resources,
	})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	identity, ok := h.identityFromPath(w, r, conn)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, h.userResource(*identity))
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	if !acceptJSON(r) {
		writeError(w, http.StatusBadRequest, "invalidValue", "The content type is not supported.")
		return
	}
	body, raw, err := decodeUser(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalidValue", "The user document is not valid.")
		return
	}
	email, name, err := identityFields(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
		return
	}
	active := true
	if _, present := raw["active"]; present {
		parsed, ok := parseActive(body.Active)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalidValue", "The active value is not valid.")
			return
		}
		active = parsed
	}
	if err := licensing.Require(r.Context(), h.pipeline.Entitlements, conn.OrganizationID, licensing.FeatureSSO); err != nil {
		writeError(w, http.StatusForbidden, "", notLicensed)
		return
	}
	if err := h.ensureUnique(r, conn.ID, uuid.Nil, body.UserName, body.ExternalID); err != nil {
		writeMapped(w, err)
		return
	}
	if active {
		if err := billing.PrepareAdmission(r.Context(), conn.OrganizationID); err != nil {
			writeMapped(w, err)
			return
		}
	}
	var created *models.SSOIdentity
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		account, err := h.pipeline.AccountForEmail(tx, conn, email, name)
		if err != nil {
			return err
		}
		now := h.now()
		identity := &models.SSOIdentity{
			ID:             uuid.New(),
			ConnectionID:   conn.ID,
			OrganizationID: conn.OrganizationID,
			AccountID:      account.ID,
			Email:          utils.NormalizeEmail(email),
			Name:           name,
			Active:         active,
			ProvisionedBy:  models.SSOProvisionedBySCIM,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		userName := body.UserName
		identity.SCIMUserName = &userName
		if body.ExternalID != "" {
			externalID := body.ExternalID
			identity.ExternalID = &externalID
		}
		if active {
			if err := h.pipeline.EnsureMembership(r.Context(), tx, conn, account, identity, models.SSOProvisionedBySCIM, models.SSOProvisionedBySCIM, false); err != nil {
				return err
			}
		}
		if err := identity.Save(tx); err != nil {
			return err
		}
		created = identity
		return nil
	})
	if err != nil {
		writeMapped(w, err)
		return
	}
	logging.AddRequestLogFields(r.Context(), map[string]any{
		"action": "scim_user_create", "object": "sso_identity", "outcome": "success",
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	})
	resource := h.userResource(*created)
	w.Header().Set("Location", resource["meta"].(map[string]any)["location"].(string))
	writeJSON(w, http.StatusCreated, resource)
}

func (h *Handler) replaceUser(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	if !acceptJSON(r) {
		writeError(w, http.StatusBadRequest, "invalidValue", "The content type is not supported.")
		return
	}
	identity, ok := h.identityFromPath(w, r, conn)
	if !ok {
		return
	}
	body, raw, err := decodeUser(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalidValue", "The user document is not valid.")
		return
	}
	change := profileChange{}
	if body.UserName != "" {
		change.UserName = &body.UserName
	}
	if body.ExternalID != "" {
		change.ExternalID = &body.ExternalID
	}
	email, name, err := identityFields(body)
	if err == nil {
		change.Email = &email
		change.Name = &name
	}
	if _, present := raw["active"]; present {
		parsed, ok := parseActive(body.Active)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalidValue", "The active value is not valid.")
			return
		}
		change.Active = &parsed
	}
	h.applyChange(w, r, conn, identity, change)
}

func (h *Handler) patchUser(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	if !acceptJSON(r) {
		writeError(w, http.StatusBadRequest, "invalidValue", "The content type is not supported.")
		return
	}
	identity, ok := h.identityFromPath(w, r, conn)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalidValue", "The patch document is not valid.")
		return
	}
	change, err := parsePatch(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
		return
	}
	h.applyChange(w, r, conn, identity, change)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) {
	identity, ok := h.identityFromPath(w, r, conn)
	if !ok {
		return
	}
	if err := h.deactivate(r, conn, identity); err != nil {
		writeMapped(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) applyChange(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection, identity *models.SSOIdentity, change profileChange) {
	userName := ""
	if change.UserName != nil {
		userName = *change.UserName
	}
	externalID := ""
	if change.ExternalID != nil {
		externalID = *change.ExternalID
	}
	if err := h.ensureUnique(r, conn.ID, identity.ID, userName, externalID); err != nil {
		writeMapped(w, err)
		return
	}
	if change.UserName != nil {
		identity.SCIMUserName = change.UserName
	}
	if change.ExternalID != nil {
		identity.ExternalID = change.ExternalID
	}
	if change.Email != nil {
		identity.Email = utils.NormalizeEmail(*change.Email)
	}
	if change.Name != nil {
		identity.Name = *change.Name
	}
	if err := identity.Save(database.DB(r.Context())); err != nil {
		writeMapped(w, err)
		return
	}
	if change.Active != nil && *change.Active != identity.Active {
		var err error
		if *change.Active {
			err = h.reactivate(r, conn, identity)
		} else {
			err = h.deactivate(r, conn, identity)
		}
		if err != nil {
			writeMapped(w, err)
			return
		}
	}
	logging.AddRequestLogFields(r.Context(), map[string]any{
		"action": "scim_user_update", "object": "sso_identity", "outcome": "success",
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	})
	updated, err := models.FindSSOIdentityByID(database.DB(r.Context()), conn.ID, identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "", "The request failed.")
		return
	}
	writeJSON(w, http.StatusOK, h.userResource(*updated))
}

func (h *Handler) deactivate(r *http.Request, conn *models.SSOConnection, identity *models.SSOIdentity) error {
	now := h.now()
	identity.Active = false
	identity.SessionsRevokedAt = &now
	if err := identity.Save(database.DB(r.Context())); err != nil {
		return err
	}
	logging.AddRequestLogFields(r.Context(), map[string]any{
		"action": "scim_user_deactivate", "object": "sso_identity", "outcome": "success",
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	})
	if identity.UserID == nil {
		return nil
	}
	_, err := organizations.RemoveUser(r.Context(), h.pipeline.Auth, conn.OrganizationID.String(), identity.UserID.String())
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if errors.Is(err, models.ErrLastOrganizationOwner) {
		logging.AddRequestLogFields(r.Context(), map[string]any{"scim_last_owner_kept": true})
		log.WithField("organization_id", conn.OrganizationID.String()).Warn("SCIM deactivation kept the last organization owner")
		return nil
	}
	code, _, ok := grpcerrors.HandlerStatus(err)
	if ok && code == codes.NotFound {
		return nil
	}
	return err
}

func (h *Handler) reactivate(r *http.Request, conn *models.SSOConnection, identity *models.SSOIdentity) error {
	if err := billing.PrepareAdmission(r.Context(), conn.OrganizationID); err != nil {
		return err
	}
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		account, err := models.LockAccount(tx, identity.AccountID)
		if err != nil {
			return err
		}
		identity.Active = true
		return h.pipeline.EnsureMembership(r.Context(), tx, conn, account, identity, models.SSOProvisionedBySCIM, models.SSOProvisionedBySCIM, false)
	})
	if err != nil {
		return err
	}
	identity.Active = true
	return identity.Save(database.DB(r.Context()))
}

func (h *Handler) ensureUnique(r *http.Request, connectionID, self uuid.UUID, userName, externalID string) error {
	if userName != "" {
		existing, err := models.FindSSOIdentityByUserName(database.DB(r.Context()), connectionID, userName)
		if err == nil && existing.ID != self {
			return uniqueError("A user with this userName already exists.")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if externalID != "" {
		existing, err := models.FindSSOIdentityByExternalID(database.DB(r.Context()), connectionID, externalID)
		if err == nil && existing.ID != self {
			return uniqueError("A user with this externalId already exists.")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (h *Handler) identityFromPath(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) (*models.SSOIdentity, bool) {
	id, err := uuid.Parse(mux.Vars(r)["userID"])
	if err != nil {
		writeError(w, http.StatusNotFound, "", "This user does not exist.")
		return nil, false
	}
	identity, err := models.FindSSOIdentityByID(database.DB(r.Context()), conn.ID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusNotFound, "", "This user does not exist.")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "", "The request failed.")
		return nil, false
	}
	return identity, true
}

func (h *Handler) userResource(identity models.SSOIdentity) map[string]any {
	userName := ""
	if identity.SCIMUserName != nil {
		userName = *identity.SCIMUserName
	}
	location := h.baseURL + "/scim/v2/" + identity.ConnectionID.String() + "/Users/" + identity.ID.String()
	resource := map[string]any{
		"schemas":     []string{schemaUser},
		"id":          identity.ID.String(),
		"userName":    userName,
		"displayName": identity.Name,
		"name":        map[string]any{"formatted": identity.Name},
		"emails":      []any{map[string]any{"value": identity.Email, "primary": true}},
		"active":      identity.Active,
		"meta": map[string]any{
			"resourceType": "User",
			"created":      identity.CreatedAt.UTC().Format(time.RFC3339),
			"lastModified": identity.UpdatedAt.UTC().Format(time.RFC3339),
			"location":     location,
		},
	}
	if identity.ExternalID != nil {
		resource["externalId"] = *identity.ExternalID
	}
	return resource
}

func decodeUser(r *http.Request) (scimUserBody, map[string]json.RawMessage, error) {
	body, err := readBody(r)
	if err != nil {
		return scimUserBody{}, nil, err
	}
	var user scimUserBody
	if err := json.Unmarshal(body, &user); err != nil {
		return scimUserBody{}, nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return scimUserBody{}, nil, err
	}
	return user, raw, nil
}

func identityFields(body scimUserBody) (string, string, error) {
	email := ""
	for _, item := range body.Emails {
		if item.Primary && strings.TrimSpace(item.Value) != "" {
			email = item.Value
			break
		}
	}
	if email == "" {
		for _, item := range body.Emails {
			if strings.TrimSpace(item.Value) != "" {
				email = item.Value
				break
			}
		}
	}
	if email == "" && strings.Contains(body.UserName, "@") {
		email = body.UserName
	}
	if strings.TrimSpace(email) == "" || !strings.Contains(email, "@") {
		return "", "", errors.New("The user has no email address.")
	}
	name := strings.TrimSpace(body.DisplayName)
	if name == "" && body.Name != nil {
		name = strings.TrimSpace(body.Name.Formatted)
		if name == "" {
			name = strings.TrimSpace(strings.TrimSpace(body.Name.GivenName) + " " + strings.TrimSpace(body.Name.FamilyName))
		}
	}
	return email, name, nil
}

func parseActive(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(typed) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func parseFilter(filter string) (string, string, bool) {
	if strings.TrimSpace(filter) == "" {
		return "", "", true
	}
	match := filterPattern.FindStringSubmatch(filter)
	if match == nil {
		return "", "", false
	}
	value := match[2]
	if value == "" {
		value = match[3]
	}
	switch strings.ToLower(match[1]) {
	case "username":
		return "userName", value, true
	case "externalid":
		return "externalId", value, true
	default:
		return "", "", false
	}
}

func page(r *http.Request) (int, int, bool) {
	start := 1
	count := 100
	if raw := r.URL.Query().Get("startIndex"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, false
		}
		if parsed < 1 {
			parsed = 1
		}
		start = parsed
	}
	if raw := r.URL.Query().Get("count"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return 0, 0, false
		}
		count = parsed
	}
	if count > 200 {
		count = 200
	}
	return start, count, true
}

func parsePatch(body []byte) (profileChange, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(body, &document); err != nil {
		return profileChange{}, errors.New("The patch document is not valid.")
	}
	var rawOps json.RawMessage
	for key, value := range document {
		if strings.EqualFold(key, "operations") {
			rawOps = value
		}
	}
	if len(rawOps) == 0 {
		return profileChange{}, errors.New("The patch document has no operations.")
	}
	var operations []struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(rawOps, &operations); err != nil {
		return profileChange{}, errors.New("The patch document is not valid.")
	}
	change := profileChange{}
	for _, operation := range operations {
		switch strings.ToLower(operation.Op) {
		case "replace", "add":
		default:
			return profileChange{}, errors.New("This patch operation is not supported.")
		}
		if err := applyPatchValue(&change, operation.Path, operation.Value); err != nil {
			return profileChange{}, err
		}
	}
	return change, nil
}

func applyPatchValue(change *profileChange, path string, value json.RawMessage) error {
	path = strings.TrimSpace(path)
	if path == "" {
		var object map[string]any
		if err := json.Unmarshal(value, &object); err != nil {
			return errors.New("The patch value is not valid.")
		}
		for key, item := range object {
			raw, err := json.Marshal(item)
			if err != nil {
				return err
			}
			if err := applyPatchValue(change, key, raw); err != nil {
				return err
			}
		}
		return nil
	}
	switch strings.ToLower(path) {
	case "username":
		text, err := jsonString(value)
		if err != nil {
			return err
		}
		change.UserName = &text
	case "externalid":
		text, err := jsonString(value)
		if err != nil {
			return err
		}
		change.ExternalID = &text
	case "displayname", "name.formatted":
		text, err := jsonString(value)
		if err != nil {
			return err
		}
		change.Name = &text
	case "emails":
		var emails []scimEmail
		if err := json.Unmarshal(value, &emails); err != nil {
			return errors.New("The email value is not valid.")
		}
		email, _, err := identityFields(scimUserBody{Emails: emails, UserName: "missing@example.com"})
		if err != nil {
			return err
		}
		change.Email = &email
	case "active":
		var parsed any
		if err := json.Unmarshal(value, &parsed); err != nil {
			return errors.New("The active value is not valid.")
		}
		active, ok := parseActive(parsed)
		if !ok {
			return errors.New("The active value is not valid.")
		}
		change.Active = &active
	default:
		return errors.New("This attribute is not supported.")
	}
	return nil
}

func jsonString(value json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return "", errors.New("The value is not text.")
	}
	return text, nil
}

type uniquenessError struct{ detail string }

func (e uniquenessError) Error() string { return e.detail }

func uniqueError(detail string) error { return uniquenessError{detail: detail} }

func writeMapped(w http.ResponseWriter, err error) {
	var unique uniquenessError
	if errors.As(err, &unique) {
		writeError(w, http.StatusConflict, "uniqueness", unique.detail)
		return
	}
	if errors.Is(err, licensing.ErrNotLicensed) || core.CodeOf(err) == core.CodeNotLicensed {
		writeError(w, http.StatusForbidden, "", notLicensed)
		return
	}
	if core.CodeOf(err) == core.CodeSeatLimit || core.IsSeatLimit(err) {
		writeError(w, http.StatusConflict, "", "No seat is available for a new member.")
		return
	}
	switch core.CodeOf(err) {
	case core.CodeAccountConflict:
		writeError(w, http.StatusConflict, "", "An account with this email already exists, and this organization has not verified the email domain.")
	case core.CodeAccountBlocked:
		writeError(w, http.StatusConflict, "", "This account is blocked.")
	default:
		writeError(w, http.StatusInternalServerError, "", "The request failed.")
	}
}

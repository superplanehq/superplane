package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	scimListResponseSchema          = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	scimUserResourceSchema          = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimGroupResourceSchema         = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimServiceProviderConfigSchema = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
)

type scimListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	Resources    []interface{} `json:"Resources"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
}

type scimUser struct {
	Schemas  []string        `json:"schemas"`
	ID       string          `json:"id,omitempty"`
	UserName string          `json:"userName"`
	Name     scimUserName    `json:"name,omitempty"`
	Active   bool            `json:"active"`
	Emails   []scimUserEmail `json:"emails,omitempty"`
}

type scimUserName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type scimUserEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary,omitempty"`
}

type scimGroup struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id,omitempty"`
	DisplayName string            `json:"displayName"`
	Members     []scimGroupMember `json:"members,omitempty"`
}

type scimGroupMember struct {
	Value string `json:"value"`
}

type scimPatchRequest struct {
	Schemas    []string      `json:"schemas"`
	Operations []scimPatchOp `json:"Operations"`
}

type scimPatchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

func (s *Server) authenticateSCIMRequest(w http.ResponseWriter, r *http.Request) (*models.Organization, *models.OrganizationOktaConfig, bool) {
	vars := mux.Vars(r)
	orgID := vars["orgId"]

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		http.Error(w, "organization not found", http.StatusNotFound)
		return nil, nil, false
	}

	config, err := models.FindOrganizationOktaConfig(org.ID)
	if err != nil {
		http.Error(w, "Okta not configured for organization", http.StatusForbidden)
		return nil, nil, false
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Okta SCIM"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}

	if models.HashSCIMToken(token) != config.ScimTokenHash {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}

	return org, config, true
}

func (s *Server) handleSCIMServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.authenticateSCIMRequest(w, r)
	if !ok {
		return
	}

	resp := map[string]interface{}{
		"schemas": []string{scimServiceProviderConfigSchema},
		"patch": map[string]bool{
			"supported": true,
		},
		"bulk": map[string]interface{}{
			"supported":      false,
			"maxOperations":  0,
			"maxPayloadSize": 0,
		},
		"filter": map[string]interface{}{
			"supported":  true,
			"maxResults": 200,
		},
		"changePassword": map[string]bool{
			"supported": false,
		},
		"sort": map[string]bool{
			"supported": false,
		},
		"authenticationSchemes": []map[string]string{
			{
				"type":        "oauthbearertoken",
				"name":        "OAuth Bearer Token",
				"description": "OAuth Bearer Token used for SCIM authentication.",
			},
		},
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSCIMUsers(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.authenticateSCIMRequest(w, r)
	if !ok {
		return
	}

	if r.Method == http.MethodPost {
		var req scimUser
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		email := req.UserName
		if email == "" && len(req.Emails) > 0 {
			email = req.Emails[0].Value
		}
		if email == "" {
			http.Error(w, "email is required", http.StatusBadRequest)
			return
		}

		name := strings.TrimSpace(req.Name.GivenName + " " + req.Name.FamilyName)
		if name == "" {
			name = email
		}

		account, err := models.FindAccountByEmail(email)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			account, err = models.CreateAccount(name, email)
			if err != nil {
				http.Error(w, "failed to create account", http.StatusInternalServerError)
				return
			}
		}

		user, err := models.FindMaybeDeletedUserByEmail(org.ID.String(), email)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			user, err = models.CreateUser(org.ID, account.ID, email, name)
			if err != nil {
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
		}

		// Ensure the user has at least a viewer role in the organization so that
		// they are visible in org member listings.
		if s.authService != nil {
			if err := s.authService.AssignRole(user.ID.String(), models.RoleOrgViewer, org.ID.String(), models.DomainTypeOrganization); err != nil {
				log.Errorf("SCIM: failed to assign default viewer role to user %s in org %s: %v", user.ID, org.ID, err)
			}
		}

		// Handle initial active state
		if !req.Active {
			if err := user.Delete(); err != nil {
				http.Error(w, "failed to deactivate user", http.StatusInternalServerError)
				return
			}
		}

		resp := scimUserFromModels(user, account)
		w.Header().Set("Content-Type", "application/scim+json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	filter := r.URL.Query().Get("filter")
	if filter == "" {
		// Okta primarily uses filtered queries; return empty list otherwise.
		resp := scimListResponse{
			Schemas:      []string{scimListResponseSchema},
			TotalResults: 0,
			Resources:    []interface{}{},
			StartIndex:   1,
			ItemsPerPage: 0,
		}
		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	attr, value, ok := parseSCIMEqFilter(filter)
	if !ok || !(attr == "userName" || attr == "email") {
		http.Error(w, "unsupported filter", http.StatusBadRequest)
		return
	}

	user, err := models.FindMaybeDeletedUserByEmail(org.ID.String(), value)
	if err != nil {
		resp := scimListResponse{
			Schemas:      []string{scimListResponseSchema},
			TotalResults: 0,
			Resources:    []interface{}{},
			StartIndex:   1,
			ItemsPerPage: 0,
		}
		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	account, err := models.FindAccountByEmail(user.Email)
	if err != nil {
		log.Errorf("SCIM: account for user %s not found: %v", user.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	u := scimUserFromModels(user, account)
	resp := scimListResponse{
		Schemas:      []string{scimListResponseSchema},
		TotalResults: 1,
		Resources:    []interface{}{u},
		StartIndex:   1,
		ItemsPerPage: 1,
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSCIMUser(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.authenticateSCIMRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	switch r.Method {
	case http.MethodGet:
		user, err := models.FindMaybeDeletedUserByID(org.ID.String(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		account, err := models.FindAccountByEmail(user.Email)
		if err != nil {
			log.Errorf("SCIM: account for user %s not found: %v", user.ID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		u := scimUserFromModels(user, account)
		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(u)
	case http.MethodPatch:
		var req scimPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid patch", http.StatusBadRequest)
			return
		}

		user, err := models.FindMaybeDeletedUserByID(org.ID.String(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		for _, op := range req.Operations {
			if strings.EqualFold(op.Path, "active") {
				var active bool
				if err := json.Unmarshal(op.Value, &active); err != nil {
					http.Error(w, "invalid active value", http.StatusBadRequest)
					return
				}
				if !active {
					if err := user.Delete(); err != nil {
						http.Error(w, "failed to deactivate user", http.StatusInternalServerError)
						return
					}
				} else {
					if err := user.Restore(); err != nil {
						http.Error(w, "failed to activate user", http.StatusInternalServerError)
						return
					}
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		user, err := models.FindMaybeDeletedUserByID(org.ID.String(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err := user.Delete(); err != nil {
			http.Error(w, "failed to delete user", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSCIMGroups(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.authenticateSCIMRequest(w, r)
	if !ok {
		return
	}

	filter := r.URL.Query().Get("filter")
	if r.Method == http.MethodGet {
		if filter == "" {
			resp := scimListResponse{
				Schemas:      []string{scimListResponseSchema},
				TotalResults: 0,
				Resources:    []interface{}{},
				StartIndex:   1,
				ItemsPerPage: 0,
			}
			w.Header().Set("Content-Type", "application/scim+json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		attr, value, ok := parseSCIMEqFilter(filter)
		if !ok || attr != "displayName" {
			http.Error(w, "unsupported filter", http.StatusBadRequest)
			return
		}

		metadata, err := models.FindGroupMetadata(value, models.DomainTypeOrganization, org.ID.String())
		if err != nil {
			resp := scimListResponse{
				Schemas:      []string{scimListResponseSchema},
				TotalResults: 0,
				Resources:    []interface{}{},
				StartIndex:   1,
				ItemsPerPage: 0,
			}
			w.Header().Set("Content-Type", "application/scim+json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		group := scimGroup{
			Schemas:     []string{scimGroupResourceSchema},
			ID:          metadata.GroupName,
			DisplayName: metadata.DisplayName,
		}

		resp := scimListResponse{
			Schemas:      []string{scimListResponseSchema},
			TotalResults: 1,
			Resources:    []interface{}{group},
			StartIndex:   1,
			ItemsPerPage: 1,
		}

		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		var req scimGroup
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		groupName := req.DisplayName
		if groupName == "" {
			http.Error(w, "displayName is required", http.StatusBadRequest)
			return
		}

		err := s.authService.CreateGroup(org.ID.String(), models.DomainTypeOrganization, groupName, models.RoleOrgViewer, groupName, "")
		if err != nil {
			log.Errorf("SCIM: failed to create group %s: %v", groupName, err)
			http.Error(w, "failed to create group", http.StatusInternalServerError)
			return
		}

		resp := scimGroup{
			Schemas:     []string{scimGroupResourceSchema},
			ID:          groupName,
			DisplayName: groupName,
		}

		w.Header().Set("Content-Type", "application/scim+json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleSCIMGroup(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.authenticateSCIMRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	groupName := vars["groupId"]

	switch r.Method {
	case http.MethodGet:
		metadata, err := models.FindGroupMetadata(groupName, models.DomainTypeOrganization, org.ID.String())
		if err != nil {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}

		userIDs, err := s.authService.GetGroupUsers(org.ID.String(), models.DomainTypeOrganization, groupName)
		if err != nil {
			http.Error(w, "failed to get group members", http.StatusInternalServerError)
			return
		}

		members := make([]scimGroupMember, 0, len(userIDs))
		for _, id := range userIDs {
			members = append(members, scimGroupMember{Value: id})
		}

		resp := scimGroup{
			Schemas:     []string{scimGroupResourceSchema},
			ID:          metadata.GroupName,
			DisplayName: metadata.DisplayName,
			Members:     members,
		}

		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(resp)
	case http.MethodPatch:
		var req scimPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid patch", http.StatusBadRequest)
			return
		}

		for _, op := range req.Operations {
			switch strings.ToLower(op.Op) {
			case "add":
				var value struct {
					Members []scimGroupMember `json:"members"`
				}
				if err := json.Unmarshal(op.Value, &value); err != nil {
					http.Error(w, "invalid members", http.StatusBadRequest)
					return
				}
				for _, m := range value.Members {
					if err := s.authService.AddUserToGroup(org.ID.String(), models.DomainTypeOrganization, m.Value, groupName); err != nil {
						log.Errorf("SCIM: failed to add user %s to group %s: %v", m.Value, groupName, err)
					}
				}
			case "remove":
				var value struct {
					Members []scimGroupMember `json:"members"`
				}
				if err := json.Unmarshal(op.Value, &value); err != nil {
					http.Error(w, "invalid members", http.StatusBadRequest)
					return
				}
				for _, m := range value.Members {
					if err := s.authService.RemoveUserFromGroup(org.ID.String(), models.DomainTypeOrganization, m.Value, groupName); err != nil {
						log.Errorf("SCIM: failed to remove user %s from group %s: %v", m.Value, groupName, err)
					}
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.authService.DeleteGroup(org.ID.String(), models.DomainTypeOrganization, groupName); err != nil {
			http.Error(w, "failed to delete group", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func scimUserFromModels(user *models.User, account *models.Account) scimUser {
	nameParts := strings.Fields(account.Name)
	var given, family string
	if len(nameParts) > 0 {
		given = nameParts[0]
	}
	if len(nameParts) > 1 {
		family = strings.Join(nameParts[1:], " ")
	}

	return scimUser{
		Schemas:  []string{scimUserResourceSchema},
		ID:       user.ID.String(),
		UserName: account.Email,
		Name: scimUserName{
			GivenName:  given,
			FamilyName: family,
		},
		Active: user.DeletedAt.Time.IsZero(),
		Emails: []scimUserEmail{
			{
				Value:   account.Email,
				Primary: true,
			},
		},
	}
}

// parseSCIMEqFilter parses filters like `userName eq "foo"` or `displayName eq "bar"`.
func parseSCIMEqFilter(filter string) (attr string, value string, ok bool) {
	parts := strings.SplitN(filter, " ", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	if !strings.EqualFold(parts[1], "eq") {
		return "", "", false
	}
	attr = parts[0]
	value = strings.TrimSpace(parts[2])
	value = strings.Trim(value, `"`)
	return attr, value, true
}

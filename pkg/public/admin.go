package public

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/impersonation"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

const defaultPageSize = 50
const maxPageSize = 200

type paginatedResponse struct {
	Items  any   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

func parsePagination(r *http.Request) (search string, limit, offset int) {
	search = r.URL.Query().Get("search")

	limit = defaultPageSize
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}

	return search, limit, offset
}

// adminListOrganizations returns paginated organizations in the installation.
// adminListAccounts returns paginated accounts in the installation.
func (s *Server) adminListAccounts(w http.ResponseWriter, r *http.Request) {
	search, limit, offset := parsePagination(r)

	accounts, total, err := models.ListAccounts(search, limit, offset)
	if err != nil {
		log.Errorf("admin: failed to list accounts: %v", err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	type accountItem struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		Email             string `json:"email"`
		InstallationAdmin bool   `json:"installation_admin"`
	}

	items := make([]accountItem, 0, len(accounts))
	for _, a := range accounts {
		items = append(items, accountItem{
			ID:                a.ID.String(),
			Name:              a.Name,
			Email:             a.Email,
			InstallationAdmin: a.IsInstallationAdmin(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *Server) adminListOrganizations(w http.ResponseWriter, r *http.Request) {
	search, limit, offset := parsePagination(r)

	organizations, total, err := models.ListAllOrganizations(search, limit, offset)
	if err != nil {
		log.Errorf("admin: failed to list organizations: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}

	orgIDs := make([]string, 0, len(organizations))
	for _, org := range organizations {
		orgIDs = append(orgIDs, org.ID.String())
	}

	canvasCounts, err := models.CountCanvasesByOrganizationIDs(orgIDs)
	if err != nil {
		log.Errorf("admin: failed to count canvases: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}

	memberCounts, err := models.CountActiveUsersByOrganizationIDs(orgIDs)
	if err != nil {
		log.Errorf("admin: failed to count members: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}

	type orgItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CanvasCount int64  `json:"canvas_count"`
		MemberCount int64  `json:"member_count"`
	}

	items := make([]orgItem, 0, len(organizations))
	for _, org := range organizations {
		id := org.ID.String()
		items = append(items, orgItem{
			ID:          id,
			Name:        org.Name,
			Description: org.Description,
			CanvasCount: canvasCounts[id],
			MemberCount: memberCounts[id],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// adminListCanvases returns paginated canvases for a given organization.
func (s *Server) adminListCanvases(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]

	if _, err := models.FindOrganizationByID(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	search, limit, offset := parsePagination(r)

	canvases, total, err := models.ListCanvasesPaginated(orgID, search, limit, offset)
	if err != nil {
		log.Errorf("admin: failed to list canvases for org %s: %v", orgID, err)
		http.Error(w, "Failed to list canvases", http.StatusInternalServerError)
		return
	}

	type canvasItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	items := make([]canvasItem, 0, len(canvases))
	for _, c := range canvases {
		items = append(items, canvasItem{
			ID:          c.ID.String(),
			Name:        c.Name,
			Description: c.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// adminListOrgUsers returns paginated human users for a given organization.
func (s *Server) adminListOrgUsers(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]

	if _, err := models.FindOrganizationByID(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	search, limit, offset := parsePagination(r)

	users, total, err := models.ListActiveUsersByOrganization(orgID, search, limit, offset)
	if err != nil {
		log.Errorf("admin: failed to list users for org %s: %v", orgID, err)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	type userItem struct {
		ID        string  `json:"id"`
		Name      string  `json:"name"`
		Email     *string `json:"email"`
		AccountID *string `json:"account_id"`
	}

	items := make([]userItem, 0, len(users))
	for _, u := range users {
		var accountID *string
		if u.AccountID != nil {
			s := u.AccountID.String()
			accountID = &s
		}
		items = append(items, userItem{
			ID:        u.ID.String(),
			Name:      u.Name,
			Email:     u.Email,
			AccountID: accountID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// startImpersonation begins an impersonation session.
func (s *Server) startImpersonation(w http.ResponseWriter, r *http.Request) {
	// Block nested impersonation only if the existing token is still valid.
	if tokenStr, err := impersonation.ReadCookie(r); err == nil {
		if _, validErr := impersonation.ValidateToken(s.jwt, tokenStr); validErr == nil {
			http.Error(w, "Already impersonating — end current session first", http.StatusBadRequest)
			return
		}
		impersonation.ClearCookie(w, r)
	}

	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		AccountID string `json:"account_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccountID == "" {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	if req.AccountID == admin.ID.String() {
		http.Error(w, "Cannot impersonate yourself", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(req.AccountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	token, err := impersonation.GenerateToken(s.jwt, admin.ID.String(), target.ID.String())
	if err != nil {
		log.Errorf("admin: failed to generate impersonation token: %v", err)
		http.Error(w, "Failed to start impersonation", http.StatusInternalServerError)
		return
	}

	impersonation.SetCookie(w, r, token)

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "impersonation_start",
		"client_ip":         r.RemoteAddr,
	}).Info("admin impersonation started")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_url": "/",
	})
}

// endImpersonation ends the current impersonation session.
func (s *Server) endImpersonation(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read the token before clearing it so we can log the target identity
	fields := log.Fields{
		"admin_account_id": account.ID.String(),
		"admin_email":      account.Email,
		"action":           "impersonation_end",
		"client_ip":        r.RemoteAddr,
	}

	if tokenStr, err := impersonation.ReadCookie(r); err == nil {
		if claims, err := impersonation.ValidateToken(s.jwt, tokenStr); err == nil {
			fields["target_account_id"] = claims.ImpersonatedAccountID
			if target, err := models.FindAccountByID(claims.ImpersonatedAccountID); err == nil {
				fields["target_email"] = target.Email
			}
		}
	}

	log.WithFields(fields).Info("admin impersonation ended")

	impersonation.ClearCookie(w, r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_url": "/admin",
	})
}

// impersonationStatus returns the current impersonation state.
func (s *Server) impersonationStatus(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := impersonation.ReadCookie(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"active": false})
		return
	}

	claims, err := impersonation.ValidateToken(s.jwt, tokenStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"active": false})
		return
	}

	// Verify the token belongs to the currently authenticated admin
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok || claims.AdminAccountID != account.ID.String() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"active": false})
		return
	}

	userName := ""
	if target, err := models.FindAccountByID(claims.ImpersonatedAccountID); err == nil {
		userName = target.Name
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"active":           true,
		"user_name":        userName,
		"admin_account_id": claims.AdminAccountID,
	})
}

// promoteAdmin promotes an account to installation admin.
func (s *Server) promoteAdmin(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := mux.Vars(r)["accountId"]

	target, err := models.FindAccountByID(targetID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := models.PromoteToInstallationAdmin(targetID); err != nil {
		log.Errorf("admin: failed to promote %s: %v", targetID, err)
		http.Error(w, "Failed to promote account", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "promote_admin",
		"client_ip":         r.RemoteAddr,
	}).Info("account promoted to installation admin")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "promoted"})
}

// demoteAdmin removes installation admin from an account.
func (s *Server) demoteAdmin(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := mux.Vars(r)["accountId"]

	// Prevent self-demotion
	if admin.ID.String() == targetID {
		http.Error(w, "Cannot demote yourself", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(targetID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := models.DemoteFromInstallationAdmin(targetID); err != nil {
		log.Errorf("admin: failed to demote %s: %v", targetID, err)
		http.Error(w, "Failed to demote account", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "demote_admin",
		"client_ip":         r.RemoteAddr,
	}).Info("account demoted from installation admin")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "demoted"})
}

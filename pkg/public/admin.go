package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/networkpolicy"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/supportsession"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const defaultPageSize = 50
const maxPageSize = 200

type paginatedResponse struct {
	Items  any   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

type installationSettingsResponse struct {
	AllowPrivateNetworkAccess  bool     `json:"allow_private_network_access"`
	SignupsEnabled             bool     `json:"signups_enabled"`
	InvitationDurationDays     int      `json:"invitation_duration_days"`
	PastDueGraceDays           int      `json:"past_due_grace_days"`
	SignupsBlockedByEnv        bool     `json:"signups_blocked_by_environment"`
	EffectiveBlockedHTTPHosts  []string `json:"effective_blocked_http_hosts"`
	EffectivePrivateIPRanges   []string `json:"effective_private_ip_ranges"`
	BlockedHTTPHostsOverridden bool     `json:"blocked_http_hosts_overridden"`
	PrivateIPRangesOverridden  bool     `json:"private_ip_ranges_overridden"`
	SMTPEnabled                bool     `json:"smtp_enabled"`
	SMTPHost                   string   `json:"smtp_host"`
	SMTPPort                   int      `json:"smtp_port"`
	SMTPUsername               string   `json:"smtp_username"`
	SMTPFromName               string   `json:"smtp_from_name"`
	SMTPFromEmail              string   `json:"smtp_from_email"`
	SMTPUseTLS                 bool     `json:"smtp_use_tls"`
	SMTPPasswordConfigured     bool     `json:"smtp_password_configured"`
}

type installationSettingsRequest struct {
	AllowPrivateNetworkAccess *bool   `json:"allow_private_network_access"`
	SignupsEnabled            *bool   `json:"signups_enabled"`
	InvitationDurationDays    *int    `json:"invitation_duration_days"`
	PastDueGraceDays          *int    `json:"past_due_grace_days"`
	SMTPEnabled               *bool   `json:"smtp_enabled"`
	SMTPHost                  *string `json:"smtp_host"`
	SMTPPort                  *int    `json:"smtp_port"`
	SMTPUsername              *string `json:"smtp_username"`
	SMTPPassword              *string `json:"smtp_password"`
	SMTPFromName              *string `json:"smtp_from_name"`
	SMTPFromEmail             *string `json:"smtp_from_email"`
	SMTPUseTLS                *bool   `json:"smtp_use_tls"`
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

func parseSorting(r *http.Request) (sortBy, sortDirection string) {
	return r.URL.Query().Get("sort_by"), r.URL.Query().Get("sort_direction")
}

func (s *Server) adminGetInstallationNetworkSettings(w http.ResponseWriter, r *http.Request) {
	response, err := s.buildInstallationSettingsResponse()
	if err != nil {
		log.Errorf("admin: failed to load installation settings: %v", err)
		http.Error(w, "Failed to load installation settings", http.StatusInternalServerError)
		return
	}

	respondJSON(w, response)
}

func (s *Server) adminUpdateInstallationNetworkSettings(w http.ResponseWriter, r *http.Request) {
	var req installationSettingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.updateInstallationSettings(r.Context(), req); err != nil {
		log.Errorf("admin: failed to update installation settings: %v", err)

		statusCode := http.StatusInternalServerError
		if errors.Is(err, errInvalidInstallationSettingsRequest) {
			statusCode = http.StatusBadRequest
		}

		http.Error(w, err.Error(), statusCode)
		return
	}

	if req.AllowPrivateNetworkAccess != nil {
		s.registry.HTTPContext().InvalidatePolicyCache()
	}

	response, err := s.buildInstallationSettingsResponse()
	if err != nil {
		log.Errorf("admin: failed to load updated installation settings: %v", err)
		http.Error(w, "Failed to update installation settings", http.StatusInternalServerError)
		return
	}

	respondJSON(w, response)
}

var errInvalidInstallationSettingsRequest = errors.New("invalid installation settings request")

func (s *Server) updateInstallationSettings(ctx context.Context, req installationSettingsRequest) error {
	if req.InvitationDurationDays != nil && (*req.InvitationDurationDays < 1 || *req.InvitationDurationDays > 365) {
		return errors.Join(errInvalidInstallationSettingsRequest, errors.New("invitation duration must be between 1 and 365 days"))
	}
	if req.PastDueGraceDays != nil {
		if err := billing.ValidatePastDueGraceDays(*req.PastDueGraceDays); err != nil {
			return errors.Join(errInvalidInstallationSettingsRequest, err)
		}
	}
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if req.AllowPrivateNetworkAccess != nil || req.SignupsEnabled != nil || req.InvitationDurationDays != nil {
			err := models.ApplyInstallationMetadata(tx, func(metadata *models.InstallationMetadata) {
				if req.AllowPrivateNetworkAccess != nil {
					metadata.AllowPrivateNetworkAccess = *req.AllowPrivateNetworkAccess
				}

				if req.SignupsEnabled != nil {
					metadata.SignupsEnabled = *req.SignupsEnabled
				}

				if req.InvitationDurationDays != nil {
					metadata.InvitationDurationDays = *req.InvitationDurationDays
				}

				metadata.UpdatedAt = time.Now()
			})
			if err != nil {
				return err
			}
		}

		if req.PastDueGraceDays != nil {
			if err := billing.UpdatePastDueGraceDays(tx, *req.PastDueGraceDays); err != nil {
				return err
			}
		}

		if !shouldUpdateSMTPSettings(req) {
			return nil
		}

		return s.updateInstallationSMTPSettingsInTransaction(ctx, tx, req)
	})
}

func (s *Server) buildInstallationSettingsResponse() (installationSettingsResponse, error) {
	metadata, err := models.GetInstallationMetadata(database.Conn())
	if err != nil {
		return installationSettingsResponse{}, err
	}

	policy, err := networkpolicy.ResolveHTTPPolicy()
	if err != nil {
		return installationSettingsResponse{}, err
	}

	graceDays, err := billing.GetPastDueGraceDays(database.Conn())
	if err != nil {
		return installationSettingsResponse{}, err
	}

	response := installationSettingsResponse{
		AllowPrivateNetworkAccess:  policy.AllowPrivateNetworkAccess,
		SignupsEnabled:             metadata.SignupsEnabled,
		InvitationDurationDays:     metadata.InvitationDurationDays,
		PastDueGraceDays:           graceDays,
		SignupsBlockedByEnv:        s.authHandler.SignupsBlockedByEnvironment(),
		EffectiveBlockedHTTPHosts:  policy.BlockedHosts,
		EffectivePrivateIPRanges:   policy.PrivateIPRanges,
		BlockedHTTPHostsOverridden: policy.BlockedHostsOverridden,
		PrivateIPRangesOverridden:  policy.PrivateIPRangesOverridden,
	}

	emailSettings, err := models.FindEmailSettings(models.EmailProviderSMTP)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return installationSettingsResponse{}, err
	}

	if emailSettings != nil {
		response.SMTPEnabled = true
		response.SMTPHost = emailSettings.SMTPHost
		response.SMTPPort = emailSettings.SMTPPort
		response.SMTPUsername = emailSettings.SMTPUsername
		response.SMTPFromName = emailSettings.SMTPFromName
		response.SMTPFromEmail = emailSettings.SMTPFromEmail
		response.SMTPUseTLS = emailSettings.SMTPUseTLS
		response.SMTPPasswordConfigured = len(emailSettings.SMTPPassword) > 0
	}

	return response, nil
}

func shouldUpdateSMTPSettings(req installationSettingsRequest) bool {
	return req.SMTPEnabled != nil ||
		req.SMTPHost != nil ||
		req.SMTPPort != nil ||
		req.SMTPUsername != nil ||
		req.SMTPPassword != nil ||
		req.SMTPFromName != nil ||
		req.SMTPFromEmail != nil ||
		req.SMTPUseTLS != nil
}

func (s *Server) updateInstallationSMTPSettingsInTransaction(ctx context.Context, tx *gorm.DB, req installationSettingsRequest) error {
	existingSettings, err := models.FindEmailSettingsInTransaction(tx, models.EmailProviderSMTP)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	smtpEnabled := existingSettings != nil
	if req.SMTPEnabled != nil {
		smtpEnabled = *req.SMTPEnabled
	}

	if !smtpEnabled {
		return models.DeleteEmailSettingsInTransaction(tx, models.EmailProviderSMTP)
	}

	settings := &models.EmailSettings{Provider: models.EmailProviderSMTP}
	if existingSettings != nil {
		settings = existingSettings
	}

	settings.SMTPHost = resolveStringField(req.SMTPHost, settings.SMTPHost)
	settings.SMTPPort = resolveIntField(req.SMTPPort, settings.SMTPPort)
	settings.SMTPUsername = resolveStringField(req.SMTPUsername, settings.SMTPUsername)
	settings.SMTPFromName = resolveStringField(req.SMTPFromName, settings.SMTPFromName)
	settings.SMTPFromEmail = resolveStringField(req.SMTPFromEmail, settings.SMTPFromEmail)
	settings.SMTPUseTLS = resolveBoolField(req.SMTPUseTLS, settings.SMTPUseTLS)

	if req.SMTPPassword != nil {
		if *req.SMTPPassword == "" {
			settings.SMTPPassword = nil
		} else {
			encryptedPassword, err := s.encryptor.Encrypt(
				ctx,
				[]byte(*req.SMTPPassword),
				[]byte("smtp_password"),
			)
			if err != nil {
				return err
			}

			settings.SMTPPassword = encryptedPassword
		}
	}

	if settings.SMTPHost == "" || settings.SMTPPort == 0 || settings.SMTPFromEmail == "" {
		return errors.Join(errInvalidInstallationSettingsRequest, errors.New("SMTP host, port, and from email are required"))
	}

	if settings.SMTPUsername != "" && len(settings.SMTPPassword) == 0 {
		return errors.Join(errInvalidInstallationSettingsRequest, errors.New("SMTP password is required when username is provided"))
	}

	if settings.SMTPUsername == "" {
		settings.SMTPPassword = nil
	}

	return models.UpsertEmailSettingsInTransaction(tx, settings)
}

func resolveStringField(value *string, fallback string) string {
	if value == nil {
		return fallback
	}

	return *value
}

func resolveIntField(value *int, fallback int) int {
	if value == nil {
		return fallback
	}

	return *value
}

func resolveBoolField(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}

// adminListOrganizations returns paginated organizations in the installation.
// adminListAccounts returns paginated accounts in the installation.
func (s *Server) adminListAccounts(w http.ResponseWriter, r *http.Request) {
	search, limit, offset := parsePagination(r)
	sortBy, sortDirection := parseSorting(r)

	accounts, total, err := models.ListAccounts(search, limit, offset, sortBy, sortDirection)
	if err != nil {
		log.Errorf("admin: failed to list accounts: %v", err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	type accountItem struct {
		ID                string  `json:"id"`
		Name              string  `json:"name"`
		Email             string  `json:"email"`
		InstallationAdmin bool    `json:"installation_admin"`
		Blocked           bool    `json:"blocked"`
		CreatedAt         *string `json:"created_at,omitempty"`
	}

	items := make([]accountItem, 0, len(accounts))
	for _, a := range accounts {
		item := accountItem{
			ID:                a.ID.String(),
			Name:              a.Name,
			Email:             a.Email,
			InstallationAdmin: a.IsInstallationAdmin(),
			Blocked:           a.IsBlocked(),
		}

		if a.CreatedAt != nil {
			formatted := a.CreatedAt.Format(time.RFC3339)
			item.CreatedAt = &formatted
		}

		items = append(items, item)
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
	ctx := r.Context()
	account, ok := middleware.GetAccountFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	search, limit, offset := parsePagination(r)
	sortBy, sortDirection := parseSorting(r)
	db := database.DB(ctx)

	pinnedOrganizations, err := models.ListPinnedOrganizationsForAccount(db, account.ID)
	if err != nil {
		log.Errorf("admin: failed to list pinned organizations: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}

	excludeIDs := make([]uuid.UUID, 0, len(pinnedOrganizations))
	for _, organization := range pinnedOrganizations {
		excludeIDs = append(excludeIDs, organization.ID)
	}

	organizations, total, matchTotal, err := listUnpinnedOrganizations(ctx, search, limit, offset, sortBy, sortDirection, excludeIDs)
	if err != nil {
		log.Errorf("admin: failed to list organizations: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}

	items, err := serializeAdminOrganizations(ctx, organizations)
	if err != nil {
		log.Errorf("admin: failed to serialize organizations: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}
	pinned, err := serializeAdminOrganizations(ctx, pinnedOrganizations)
	if err != nil {
		log.Errorf("admin: failed to serialize pinned organizations: %v", err)
		http.Error(w, "Failed to list organizations", http.StatusInternalServerError)
		return
	}
	for i := range pinned {
		pinned[i].Pinned = true
	}

	respondJSON(w, adminOrganizationListResponse{
		Items:      items,
		Pinned:     pinned,
		Total:      total,
		MatchTotal: matchTotal,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Server) adminGetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	db := database.DB(r.Context())
	organization, err := models.FindOrganizationWithCounts(db, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to load organization %s: %v", orgID, err)
		http.Error(w, "Failed to load organization", http.StatusInternalServerError)
		return
	}

	pinned, err := models.IsOrganizationPinnedForAccount(db, account.ID, orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization pin %s: %v", orgID, err)
		http.Error(w, "Failed to load organization", http.StatusInternalServerError)
		return
	}

	items, err := serializeAdminOrganizations(r.Context(), []models.OrganizationWithCounts{*organization})
	if err != nil {
		log.Errorf("admin: failed to serialize organization %s: %v", orgID, err)
		http.Error(w, "Failed to load organization", http.StatusInternalServerError)
		return
	}

	item := items[0]
	item.Pinned = pinned
	respondJSON(w, item)
}

func (s *Server) adminPinOrganization(w http.ResponseWriter, r *http.Request) {
	account, orgID, ok := adminOrganizationPinRequest(w, r)
	if !ok {
		return
	}

	if err := models.PinOrganizationForAccount(database.DB(r.Context()), account.ID, orgID); err != nil {
		log.Errorf("admin: failed to pin organization %s: %v", orgID, err)
		http.Error(w, "Failed to pin organization", http.StatusInternalServerError)
		return
	}

	respondJSON(w, adminOrganizationPinState{Pinned: true})
}

func (s *Server) adminUnpinOrganization(w http.ResponseWriter, r *http.Request) {
	account, orgID, ok := adminOrganizationPinRequest(w, r)
	if !ok {
		return
	}

	if err := models.UnpinOrganizationForAccount(database.DB(r.Context()), account.ID, orgID); err != nil {
		log.Errorf("admin: failed to unpin organization %s: %v", orgID, err)
		http.Error(w, "Failed to unpin organization", http.StatusInternalServerError)
		return
	}

	respondJSON(w, adminOrganizationPinState{Pinned: false})
}

func adminOrganizationPinRequest(w http.ResponseWriter, r *http.Request) (*models.Account, uuid.UUID, bool) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, uuid.Nil, false
	}

	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return nil, uuid.Nil, false
	}

	return account, orgID, true
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

// startSupportSession begins a support session.
func (s *Server) startSupportSession(w http.ResponseWriter, r *http.Request) {
	// Block nested support sessions only if the existing token is still valid.
	if tokenStr, err := supportsession.ReadCookie(r); err == nil {
		if _, validErr := supportsession.ValidateToken(s.jwt, tokenStr); validErr == nil {
			http.Error(w, "End the current support session first", http.StatusBadRequest)
			return
		}
		supportsession.ClearCookie(w, r)
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
		http.Error(w, "You cannot start a support session for your own account", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(req.AccountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}
	if target.IsBlocked() {
		http.Error(w, "You cannot start a support session for a blocked account", http.StatusBadRequest)
		return
	}

	token, err := supportsession.GenerateToken(s.jwt, admin.ID.String(), target.ID.String())
	if err != nil {
		log.Errorf("admin: failed to generate support session token: %v", err)
		http.Error(w, "Failed to start support session", http.StatusInternalServerError)
		return
	}

	supportsession.SetCookie(w, r, token)

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "support_session_start",
		"client_ip":         r.RemoteAddr,
	}).Info("admin support session started")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_url": "/",
	})
}

// endSupportSession ends the current support session.
func (s *Server) endSupportSession(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read the token before clearing it so we can log the target identity
	fields := log.Fields{
		"admin_account_id": account.ID.String(),
		"admin_email":      account.Email,
		"action":           "support_session_end",
		"client_ip":        r.RemoteAddr,
	}

	if tokenStr, err := supportsession.ReadCookie(r); err == nil {
		if claims, err := supportsession.ValidateToken(s.jwt, tokenStr); err == nil {
			fields["target_account_id"] = claims.TargetAccountID
			if target, err := models.FindAccountByID(claims.TargetAccountID); err == nil {
				fields["target_email"] = target.Email
			}
		}
	}

	log.WithFields(fields).Info("admin support session ended")

	supportsession.ClearCookie(w, r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_url": "/admin",
	})
}

// supportSessionStatus returns the current support session state.
func (s *Server) supportSessionStatus(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := supportsession.ReadCookie(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"active": false})
		return
	}

	claims, err := supportsession.ValidateToken(s.jwt, tokenStr)
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

	target, err := models.FindAccountByID(claims.TargetAccountID)
	if err != nil || target.IsBlocked() {
		supportsession.ClearCookie(w, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"active": false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"active":           true,
		"user_name":        target.Name,
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

// blocks an account and invalidates its sessions and credentials.
func (s *Server) blockAccount(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := mux.Vars(r)["accountId"]
	if admin.ID.String() == targetID {
		http.Error(w, "Cannot block yourself", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(targetID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		return target.Block(tx, time.Now())
	}); err != nil {
		log.Errorf("admin: failed to block %s: %v", targetID, err)
		http.Error(w, "Failed to block account", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "block_account",
		"client_ip":         r.RemoteAddr,
	}).Info("account blocked")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "blocked"})
}

func (s *Server) unblockAccount(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := mux.Vars(r)["accountId"]
	if admin.ID.String() == targetID {
		http.Error(w, "Cannot unblock yourself", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(targetID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		return target.Unblock(tx)
	}); err != nil {
		log.Errorf("admin: failed to unblock %s: %v", targetID, err)
		http.Error(w, "Failed to unblock account", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"admin_account_id":  admin.ID.String(),
		"admin_email":       admin.Email,
		"target_account_id": target.ID.String(),
		"target_email":      target.Email,
		"action":            "unblock_account",
		"client_ip":         r.RemoteAddr,
	}).Info("account unblocked")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unblocked"})
}

// adminDeleteAccount deletes an account, its sign-in methods, and every
// organization where it is the only owner, including their workspaces.
func (s *Server) adminDeleteAccount(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := mux.Vars(r)["accountId"]
	if admin.ID.String() == targetID {
		http.Error(w, "Cannot delete your own account from the admin panel", http.StatusBadRequest)
		return
	}

	target, err := models.FindAccountByID(targetID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}
	targetEmail := target.Email
	actorID := admin.ID

	deletedOrganizations, err := s.softDeleteAccount(r.Context(), target)
	if errors.Is(err, models.ErrAccountDeleteLastInstallationAdmin) {
		http.Error(w, "Promote another installation admin before you delete this account.", http.StatusConflict)
		return
	}
	if err != nil {
		log.Errorf("admin: failed to delete account %s: %v", targetID, err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	deletedOrganizationIDs := make([]string, 0, len(deletedOrganizations))
	captureAccountDeletion(r.Context(), actorID, target.ID, target.DeletedAt.Time, deletedOrganizations, "admin")
	for _, organization := range deletedOrganizations {
		deletedOrganizationIDs = append(deletedOrganizationIDs, organization.ID.String())
	}

	log.WithFields(log.Fields{
		"admin_account_id":         admin.ID.String(),
		"admin_email":              admin.Email,
		"target_account_id":        target.ID.String(),
		"target_email":             targetEmail,
		"deleted_organization_ids": deletedOrganizationIDs,
		"action":                   "delete_account",
		"client_ip":                r.RemoteAddr,
	}).Info("account deleted")

	respondJSON(w, map[string]any{
		"status":                   "deleted",
		"deleted_organization_ids": deletedOrganizationIDs,
	})
}

// adminListOrgExperimentalFeatures returns the installation feature registry
// plus the features enabled for the given organization. Installation admins
// are not always members of the org, so this must not go through the member
// DescribeOrganization API.
func (s *Server) adminListOrgExperimentalFeatures(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	enabled := []string(org.EnabledExperimentalFeatures)
	if enabled == nil {
		enabled = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"features": experimentalFeatureItems(),
		"enabled":  enabled,
	})
}

// adminEnableOrgExperimentalFeature toggles an experimental feature on for
// the given organization.
func (s *Server) adminEnableOrgExperimentalFeature(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]
	featureID := mux.Vars(r)["featureId"]

	if !features.Exists(featureID) {
		http.Error(w, "Unknown experimental feature", http.StatusBadRequest)
		return
	}

	parsedOrgID, err := uuid.Parse(orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if _, err := models.FindOrganizationByID(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}
	if err := models.EnableExperimentalFeature(parsedOrgID, featureID); err != nil {
		log.Errorf("admin: failed to enable feature %s for org %s: %v", featureID, orgID, err)
		http.Error(w, "Failed to enable feature", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "enabled"})
}

// adminDisableOrgExperimentalFeature toggles an experimental feature off for
// the given organization. Disabling an id that is not enabled is a no-op.
func (s *Server) adminDisableOrgExperimentalFeature(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]
	featureID := mux.Vars(r)["featureId"]

	parsedOrgID, err := uuid.Parse(orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if _, err := models.FindOrganizationByID(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := models.DisableExperimentalFeature(parsedOrgID, featureID); err != nil {
		log.Errorf("admin: failed to disable feature %s for org %s: %v", featureID, orgID, err)
		http.Error(w, "Failed to disable feature", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
}

type adminOrgItem struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	PlanKey              string  `json:"plan_key"`
	PlanName             string  `json:"plan_name"`
	Slug                 string  `json:"slug"`
	Description          string  `json:"description"`
	CanvasCount          int64   `json:"canvas_count"`
	TaskCount            int64   `json:"task_count"`
	DoneTaskCount        int64   `json:"done_task_count"`
	MemberCount          int64   `json:"member_count"`
	WorkspaceCount       int64   `json:"workspace_count"`
	RepositoryCount      int64   `json:"repository_count"`
	RemainingCreditCents int64   `json:"remaining_credit_cents"`
	CreatedAt            *string `json:"created_at,omitempty"`
	UpdatedAt            *string `json:"updated_at,omitempty"`
	Pinned               bool    `json:"pinned"`
}

type adminOrganizationListResponse struct {
	Items      []adminOrgItem `json:"items"`
	Pinned     []adminOrgItem `json:"pinned"`
	Total      int64          `json:"total"`
	MatchTotal int64          `json:"match_total"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
}

type adminOrganizationPinState struct {
	Pinned bool `json:"pinned"`
}

func listUnpinnedOrganizations(ctx context.Context, search string, limit, offset int, sortBy, sortDirection string, excludeIDs []uuid.UUID) (organizations []models.OrganizationWithCounts, unpinnedTotal int64, matchTotal int64, err error) {
	ctx, done := telemetry.Span(ctx, "organizations.list")
	defer done(&err)

	return models.ListUnpinnedOrganizations(database.DB(ctx), search, limit, offset, sortBy, sortDirection, excludeIDs)
}

func serializeAdminOrganizations(ctx context.Context, organizations []models.OrganizationWithCounts) (items []adminOrgItem, err error) {
	ctx, done := telemetry.Span(ctx, "organizations.serialize")
	defer done(&err)

	if err := models.ResolveOrganizationPlans(database.DB(ctx), organizations); err != nil {
		return nil, err
	}
	remaining, err := adminOrganizationRemainingCents(ctx, organizations)
	if err != nil {
		return nil, err
	}

	items = make([]adminOrgItem, 0, len(organizations))
	for _, org := range organizations {
		items = append(items, serializeAdminOrganization(org, remaining[org.ID]))
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("organizations.count", len(items)))
	}

	return items, nil
}

func adminOrganizationRemainingCents(ctx context.Context, organizations []models.OrganizationWithCounts) (map[uuid.UUID]int64, error) {
	cents := make(map[uuid.UUID]int64, len(organizations))
	missing := make([]uuid.UUID, 0)
	for _, org := range organizations {
		if org.RemainingCreditMicros == nil {
			missing = append(missing, org.ID)
			continue
		}
		cents[org.ID] = billing.MicrosToCents(*org.RemainingCreditMicros)
	}
	if len(missing) == 0 {
		return cents, nil
	}

	calculated, err := remainingCreditCentsByOrganization(ctx, missing)
	if err != nil {
		return nil, err
	}
	for id, value := range calculated {
		cents[id] = value
	}
	return cents, nil
}

func remainingCreditCentsByOrganization(ctx context.Context, organizationIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	micros, err := billing.RemainingCreditMicros(database.DB(ctx), organizationIDs)
	if err != nil {
		return nil, err
	}

	cents := make(map[uuid.UUID]int64, len(organizationIDs))
	for _, id := range organizationIDs {
		cents[id] = billing.MicrosToCents(micros[id])
	}
	return cents, nil
}

func serializeAdminOrganization(org models.OrganizationWithCounts, remainingCreditCents int64) adminOrgItem {
	item := adminOrgItem{
		ID:                   org.ID.String(),
		Name:                 org.Name,
		Slug:                 org.Slug,
		Description:          org.Description,
		CanvasCount:          org.CanvasCount,
		TaskCount:            org.TaskCount,
		DoneTaskCount:        org.DoneTaskCount,
		MemberCount:          org.MemberCount,
		WorkspaceCount:       org.WorkspaceCount,
		RepositoryCount:      org.RepositoryCount,
		RemainingCreditCents: remainingCreditCents,
	}

	if org.PlanSummary != nil {
		item.PlanKey = org.PlanSummary.Key
		item.PlanName = org.PlanSummary.Name
	}
	if org.CreatedAt != nil {
		formatted := org.CreatedAt.Format(time.RFC3339)
		item.CreatedAt = &formatted
	}

	if org.UpdatedAt != nil {
		formatted := org.UpdatedAt.Format(time.RFC3339)
		item.UpdatedAt = &formatted
	}

	return item
}

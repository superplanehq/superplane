package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// MaxLicenseRequestBytes limits the install request body. A license is much
// smaller than this limit.
const MaxLicenseRequestBytes = 64 * 1024

type installationLicenseResponse struct {
	Edition string `json:"edition"`
	State   string `json:"state"`
	Source  string `json:"source"`
	Reason  string `json:"reason,omitempty"`
	// ManagedByConfiguration is true when SUPERPLANE_LICENSE_PATH provides the
	// license. Administrators cannot change it in the UI.
	ManagedByConfiguration  bool                             `json:"managed_by_configuration"`
	License                 *installationLicenseDetails      `json:"license,omitempty"`
	TrustedKeys             *trustedKeysResponse             `json:"trusted_keys,omitempty"`
	FeatureCounts           map[string]int64                 `json:"feature_counts"`
	OrganizationCount       int64                            `json:"organization_count"`
	Event                   string                           `json:"event,omitempty"`
	RemovedGrants           []removedEnterpriseGrantResponse `json:"removed_grants,omitempty"`
	AutoGrantOrganizationID string                           `json:"auto_grant_organization_id,omitempty"`
}

type removedEnterpriseGrantResponse struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	FeatureKey       string `json:"feature_key"`
}

type licenseInstallDryRunResponse struct {
	License                 *installationLicenseDetails      `json:"license"`
	Event                   string                           `json:"event"`
	RemovedGrants           []removedEnterpriseGrantResponse `json:"removed_grants"`
	AutoGrantOrganizationID string                           `json:"auto_grant_organization_id"`
}

type licenseRemovalDryRunResponse struct {
	RemovedGrants []removedEnterpriseGrantResponse `json:"removed_grants"`
}

// trustedKeysResponse describes the key list that verifies licenses. The
// state is syncing until the first download finishes.
type trustedKeysResponse struct {
	State    string     `json:"state"`
	Version  int64      `json:"version"`
	SyncedAt *time.Time `json:"synced_at,omitempty"`
}

type installationLicenseDetails struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Features   []string  `json:"features"`
	IssuedAt   time.Time `json:"issued_at"`
	ValidFrom  time.Time `json:"valid_from"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type installLicenseRequest struct {
	License string `json:"license"`
}

type installKeyListRequest struct {
	KeyList string `json:"key_list"`
}

func (s *Server) licenseStatus() licensing.Status {
	if s.licenseService == nil {
		return licensing.Status{
			Edition: licensing.EditionCommunity,
			Source:  licensing.SourceNone,
			State:   licensing.StateNone,
		}
	}

	return s.licenseService.Status()
}

func installationLicenseResponseFrom(status licensing.Status) installationLicenseResponse {
	response := installationLicenseResponse{
		Edition:                string(status.Edition),
		State:                  string(status.State),
		Source:                 string(status.Source),
		Reason:                 string(status.Reason),
		ManagedByConfiguration: !status.Writable,
	}

	if status.License != nil {
		response.License = &installationLicenseDetails{
			ID:         status.License.ID.String(),
			CustomerID: status.License.CustomerID.String(),
			Features:   featureKeys(status.License.Features),
			IssuedAt:   status.License.IssuedAt.UTC(),
			ValidFrom:  status.License.ValidFrom.UTC(),
			ExpiresAt:  status.License.ExpiresAt.UTC(),
		}
	}

	return response
}

func featureKeys(features []licensing.Feature) []string {
	keys := make([]string, 0, len(features))
	for _, feature := range features {
		keys = append(keys, string(feature))
	}

	return keys
}

func (s *Server) installationLicenseResponse(status licensing.Status) (installationLicenseResponse, error) {
	response := installationLicenseResponseFrom(status)
	if s.licenseService != nil {
		if keys, ok := s.licenseService.TrustedKeys(); ok {
			response.TrustedKeys = &trustedKeysResponse{State: string(keys.State), Version: keys.Version, SyncedAt: keys.SyncedAt}
		}
	}

	counts, err := models.CountOrganizationsByEnterpriseFeature(database.Conn())
	if err != nil {
		return response, err
	}
	if counts == nil {
		counts = map[string]int64{}
	}
	organizationCount, err := models.CountOrganizations(database.Conn())
	if err != nil {
		return response, err
	}

	response.FeatureCounts = counts
	response.OrganizationCount = organizationCount
	return response, nil
}

func (s *Server) writeInstallationLicense(w http.ResponseWriter, status licensing.Status) {
	response, err := s.installationLicenseResponse(status)
	if err != nil {
		log.WithError(err).Error("admin: failed to read enterprise feature usage")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	respondJSON(w, response)
}

func (s *Server) adminGetInstallationLicense(w http.ResponseWriter, _ *http.Request) {
	s.writeInstallationLicense(w, s.licenseStatus())
}

func (s *Server) adminInstallInstallationLicense(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	if s.licenseService == nil {
		http.Error(w, "License management is not available", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxLicenseRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req installLicenseRequest
	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "The license is too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.License) == "" {
		http.Error(w, "Paste a license to install", http.StatusBadRequest)
		return
	}

	raw := []byte(req.License)
	if dryRunRequested(r) {
		s.adminDryRunInstallLicense(w, r, raw)
		return
	}

	before, err := models.FindOpenLicenseActivation(database.Conn())
	if err != nil {
		log.WithError(err).Error("admin: failed to read the open license activation")
		http.Error(w, "Failed to change the installation license", http.StatusInternalServerError)
		return
	}

	status, err := s.licenseService.Install(r.Context(), raw, account.ID)
	if err != nil {
		writeLicenseError(w, err)
		return
	}

	response, err := s.installationLicenseResponse(status)
	if err != nil {
		log.WithError(err).Error("admin: failed to read enterprise feature usage")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	after, err := models.FindOpenLicenseActivation(database.Conn())
	if err != nil {
		log.WithError(err).Error("admin: failed to read the open license activation")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	applied := enterprise.DescribeAppliedInstall(before, after)
	response.Event = applied.Event
	removed, err := removedGrantRecords(applied.Removed)
	if err != nil {
		log.WithError(err).Error("admin: failed to read organizations for removed grants")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}
	response.RemovedGrants = removed
	autoGrantID, err := autoGrantOrganizationID(applied.AutoGranted)
	if err != nil {
		log.WithError(err).Error("admin: failed to read the organization that received automatic grants")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}
	response.AutoGrantOrganizationID = autoGrantID
	respondJSON(w, response)
}

func (s *Server) adminDryRunInstallLicense(w http.ResponseWriter, r *http.Request, raw []byte) {
	license, err := s.licenseService.DryRunInstall(r.Context(), raw)
	if err != nil {
		writeLicenseError(w, err)
		return
	}

	plan, err := s.licenseInstallPlan(license)
	if err != nil {
		log.WithError(err).Error("admin: failed to plan the license install")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	removed, err := plannedGrantResponses(plan.Removed)
	if err != nil {
		log.WithError(err).Error("admin: failed to read organizations for the license plan")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	autoGrantID := ""
	if plan.AutoGrantOrganizationID != uuid.Nil {
		autoGrantID = plan.AutoGrantOrganizationID.String()
	}
	respondJSON(w, licenseInstallDryRunResponse{
		License:                 licenseDetails(license),
		Event:                   plan.Event,
		RemovedGrants:           removed,
		AutoGrantOrganizationID: autoGrantID,
	})
}

func (s *Server) adminRemoveInstallationLicense(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	if s.licenseService == nil {
		http.Error(w, "License management is not available", http.StatusServiceUnavailable)
		return
	}

	if dryRunRequested(r) {
		s.adminDryRunRemoveLicense(w)
		return
	}

	open, err := models.FindOpenLicenseActivation(database.Conn())
	if err != nil {
		log.WithError(err).Error("admin: failed to read the open license activation")
		http.Error(w, "Failed to change the installation license", http.StatusInternalServerError)
		return
	}

	status, err := s.licenseService.Remove(r.Context(), account.ID)
	if err != nil {
		writeLicenseError(w, err)
		return
	}

	response, err := s.installationLicenseResponse(status)
	if err != nil {
		log.WithError(err).Error("admin: failed to read enterprise feature usage")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	if open != nil {
		closed, findErr := models.FindLicenseActivationByID(database.Conn(), open.ID)
		if findErr != nil {
			log.WithError(findErr).Error("admin: failed to read the closed license activation")
			http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
			return
		}
		removed, recordErr := removedGrantRecords(enterprise.GrantsRemovedByDeactivation(closed))
		if recordErr != nil {
			log.WithError(recordErr).Error("admin: failed to read organizations for removed grants")
			http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
			return
		}
		response.RemovedGrants = removed
	}
	respondJSON(w, response)
}

func (s *Server) adminDryRunRemoveLicense(w http.ResponseWriter) {
	grants, err := models.ListAllOrganizationEnterpriseFeatures(database.Conn())
	if err != nil {
		log.WithError(err).Error("admin: failed to read enterprise feature grants")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	removed, err := plannedGrantResponses(enterprise.PlanLicenseRemoval(grants))
	if err != nil {
		log.WithError(err).Error("admin: failed to read organizations for the license plan")
		http.Error(w, "Failed to read the installation license", http.StatusInternalServerError)
		return
	}

	respondJSON(w, licenseRemovalDryRunResponse{RemovedGrants: removed})
}

func (s *Server) adminInstallLicenseKeyList(w http.ResponseWriter, r *http.Request) {
	if s.licenseService == nil {
		http.Error(w, "License management is not available", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxLicenseRequestBytes+licensing.MaxKeyListBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req installKeyListRequest
	if err := decoder.Decode(&req); err != nil || strings.TrimSpace(req.KeyList) == "" {
		http.Error(w, "Paste a key list to upload", http.StatusBadRequest)
		return
	}

	err := s.licenseService.InstallKeyList(r.Context(), []byte(req.KeyList))
	switch {
	case errors.Is(err, licensing.ErrInvalidKeyList):
		http.Error(w, "The key list is not valid.", http.StatusUnprocessableEntity)
		return
	case errors.Is(err, licensing.ErrKeyListDowngrade):
		http.Error(w, "The key list is older than the trusted keys.", http.StatusUnprocessableEntity)
		return
	case errors.Is(err, licensing.ErrKeySyncUnavailable):
		http.Error(w, "License key updates are not available", http.StatusServiceUnavailable)
		return
	case err != nil:
		log.WithError(err).Error("admin: failed to upload the license key list")
		http.Error(w, "Failed to upload the key list", http.StatusInternalServerError)
		return
	}

	s.writeInstallationLicense(w, s.licenseStatus())
}

func writeLicenseError(w http.ResponseWriter, err error) {
	if errors.Is(err, licensing.ErrManagedByFile) {
		http.Error(w, "The license is managed by the installation configuration", http.StatusConflict)
		return
	}

	var verificationErr *licensing.VerificationError
	if errors.As(err, &verificationErr) {
		http.Error(w, licenseReasonMessage(verificationErr.Reason), http.StatusUnprocessableEntity)
		return
	}

	log.WithError(err).Error("admin: failed to change the installation license")
	http.Error(w, "Failed to change the installation license", http.StatusInternalServerError)
}

func licenseReasonMessage(reason licensing.Reason) string {
	switch reason {
	case licensing.ReasonMalformed:
		return "The license is not in a valid format. Paste the complete license."
	case licensing.ReasonUnknownKey:
		return "SuperPlane does not trust the key that signed the license. Make sure that SuperPlane can download license key updates, or upload the latest key list."
	case licensing.ReasonUnsupportedAlgorithm, licensing.ReasonInvalidSignature:
		return "The license signature is not valid."
	case licensing.ReasonInvalidClaims:
		return "The license is not valid for self-hosted SuperPlane."
	case licensing.ReasonExpired:
		return "The license has expired."
	case licensing.ReasonNotYetValid:
		return "The license is not valid yet."
	case licensing.ReasonRevoked:
		return "This license was revoked."
	default:
		return "The license could not be read."
	}
}

func dryRunRequested(r *http.Request) bool {
	return r.URL.Query().Get("dry_run") == "true"
}

func (s *Server) licenseInstallPlan(license *licensing.License) (enterprise.LicenseInstallPlan, error) {
	open, err := models.FindOpenLicenseActivation(database.Conn())
	if err != nil {
		return enterprise.LicenseInstallPlan{}, err
	}

	grants, err := models.ListAllOrganizationEnterpriseFeatures(database.Conn())
	if err != nil {
		return enterprise.LicenseInstallPlan{}, err
	}

	count, err := models.CountOrganizations(database.Conn())
	if err != nil {
		return enterprise.LicenseInstallPlan{}, err
	}

	var onlyOrganizationID uuid.UUID
	if count == 1 {
		organization, findErr := models.FindOnlyOrganization(database.Conn())
		if findErr != nil {
			return enterprise.LicenseInstallPlan{}, findErr
		}
		if organization != nil {
			onlyOrganizationID = organization.ID
		}
	}

	return enterprise.PlanLicenseInstall(open, license, grants, count, onlyOrganizationID), nil
}

func licenseDetails(license *licensing.License) *installationLicenseDetails {
	if license == nil {
		return nil
	}

	return &installationLicenseDetails{
		ID:         license.ID.String(),
		CustomerID: license.CustomerID.String(),
		Features:   featureKeys(license.Features),
		IssuedAt:   license.IssuedAt.UTC(),
		ValidFrom:  license.ValidFrom.UTC(),
		ExpiresAt:  license.ExpiresAt.UTC(),
	}
}

func autoGrantOrganizationID(autoGranted bool) (string, error) {
	if !autoGranted {
		return "", nil
	}

	organization, err := models.FindOnlyOrganization(database.Conn())
	if err != nil || organization == nil {
		return "", err
	}

	return organization.ID.String(), nil
}

func plannedGrantResponses(removals []enterprise.PlannedGrantRemoval) ([]removedEnterpriseGrantResponse, error) {
	ids := make([]uuid.UUID, 0, len(removals))
	for _, removal := range removals {
		ids = append(ids, removal.OrganizationID)
	}

	names, err := organizationNames(ids)
	if err != nil {
		return nil, err
	}

	responses := make([]removedEnterpriseGrantResponse, 0, len(removals))
	for _, removal := range removals {
		responses = append(responses, removedEnterpriseGrantResponse{
			OrganizationID:   removal.OrganizationID.String(),
			OrganizationName: names[removal.OrganizationID],
			FeatureKey:       removal.FeatureKey,
		})
	}

	return responses, nil
}

func removedGrantRecords(grants []models.RemovedEnterpriseGrant) ([]removedEnterpriseGrantResponse, error) {
	ids := make([]uuid.UUID, 0, len(grants))
	for _, grant := range grants {
		ids = append(ids, grant.OrganizationID)
	}

	names, err := organizationNames(ids)
	if err != nil {
		return nil, err
	}

	responses := make([]removedEnterpriseGrantResponse, 0, len(grants))
	for _, grant := range grants {
		responses = append(responses, removedEnterpriseGrantResponse{
			OrganizationID:   grant.OrganizationID.String(),
			OrganizationName: names[grant.OrganizationID],
			FeatureKey:       grant.FeatureKey,
		})
	}

	return responses, nil
}

func organizationNames(ids []uuid.UUID) (map[uuid.UUID]string, error) {
	names := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return names, nil
	}

	var organizations []models.Organization
	if err := database.Conn().Where("id IN ?", ids).Find(&organizations).Error; err != nil {
		return nil, err
	}
	for _, organization := range organizations {
		names[organization.ID] = organization.Name
	}

	return names, nil
}

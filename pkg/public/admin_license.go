package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/licensing"
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
	ManagedByConfiguration bool                        `json:"managed_by_configuration"`
	License                *installationLicenseDetails `json:"license,omitempty"`
	TrustedKeys            *trustedKeysResponse        `json:"trusted_keys,omitempty"`
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

func (s *Server) installationLicenseResponse(status licensing.Status) installationLicenseResponse {
	response := installationLicenseResponseFrom(status)
	if s.licenseService == nil {
		return response
	}

	if keys, ok := s.licenseService.TrustedKeys(); ok {
		response.TrustedKeys = &trustedKeysResponse{State: string(keys.State), Version: keys.Version, SyncedAt: keys.SyncedAt}
	}

	return response
}

func (s *Server) adminGetInstallationLicense(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, s.installationLicenseResponse(s.licenseStatus()))
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

	status, err := s.licenseService.Install(r.Context(), []byte(req.License), account.ID)
	if err != nil {
		writeLicenseError(w, err)
		return
	}

	respondJSON(w, s.installationLicenseResponse(status))
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

	status, err := s.licenseService.Remove(r.Context(), account.ID)
	if err != nil {
		writeLicenseError(w, err)
		return
	}

	respondJSON(w, s.installationLicenseResponse(status))
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

	respondJSON(w, s.installationLicenseResponse(s.licenseStatus()))
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

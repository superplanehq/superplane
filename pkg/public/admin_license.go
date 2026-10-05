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

func (s *Server) adminGetInstallationLicense(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, installationLicenseResponseFrom(s.licenseStatus()))
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

	respondJSON(w, installationLicenseResponseFrom(status))
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

	respondJSON(w, installationLicenseResponseFrom(status))
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
		return "This SuperPlane version does not trust the key that signed the license. Upgrade SuperPlane, or contact SuperPlane support."
	case licensing.ReasonUnsupportedAlgorithm, licensing.ReasonInvalidSignature:
		return "The license signature is not valid."
	case licensing.ReasonInvalidClaims:
		return "The license is not valid for self-hosted SuperPlane."
	case licensing.ReasonExpired:
		return "The license has expired."
	case licensing.ReasonNotYetValid:
		return "The license is not valid yet."
	default:
		return "The license could not be read."
	}
}

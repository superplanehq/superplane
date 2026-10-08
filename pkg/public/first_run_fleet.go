package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/fleets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type firstRunFleetManagerResponse struct {
	FleetID string `json:"fleet_id"`
	Token   string `json:"token"`
	YAML    string `json:"yaml"`
}

func (s *Server) adminPrepareFleetManager(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	db := database.DB(r.Context())
	fleet, err := ensureFirstRunFleet(db)
	if err != nil {
		http.Error(w, "failed to create runner fleet", http.StatusInternalServerError)
		return
	}

	organizations, err := models.ListOrganizationsCreatedByAccount(db, account.ID)
	if err != nil || len(organizations) == 0 {
		http.Error(w, "failed to load owner organization", http.StatusInternalServerError)
		return
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(r.Header.Get("X-Forwarded-Host")), "/")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	if strings.TrimSpace(os.Getenv("APP_ENV")) == "development" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firstRunFleetManagerResponse{
			FleetID: fleet.Slug,
			YAML:    fleets.FirstRunManagerYAML(baseURL, ""),
		})
		return
	}

	user, err := models.FindActiveHumanUserByAccountAndOrganization(db, organizations[0].ID, account.ID)
	if err != nil {
		http.Error(w, "failed to load owner user", http.StatusInternalServerError)
		return
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		http.Error(w, "failed to create Fleet Manager token", http.StatusInternalServerError)
		return
	}
	token := models.NewUserAPIToken(user.ID, "Fleet Manager", crypto.HashToken(plainToken))
	if err := models.CreateUserAPIToken(db, token); err != nil {
		http.Error(w, "failed to create Fleet Manager token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(firstRunFleetManagerResponse{
		FleetID: fleet.Slug,
		Token:   plainToken,
		YAML:    fleets.FirstRunManagerYAML(baseURL, plainToken),
	})
}

func ensureFirstRunFleet(tx *gorm.DB) (*models.RunnerFleet, error) {
	fleet, err := models.FindInstallationRunnerFleet(tx, fleets.FirstRunFleetID())
	if err == nil {
		return fleet, nil
	}
	if !errors.Is(err, models.ErrRunnerFleetNotFound) {
		return nil, err
	}

	defaults := models.DefaultInstallationRunnerFleets(models.DefaultRunnerVersion)
	for i := range defaults {
		if defaults[i].Slug != fleets.FirstRunFleetID() {
			continue
		}
		if err := defaults[i].Create(tx); err != nil {
			if errors.Is(err, models.ErrRunnerFleetIDConflict) {
				return models.FindInstallationRunnerFleet(tx, fleets.FirstRunFleetID())
			}
			return nil, err
		}
		return &defaults[i], nil
	}
	return nil, errors.New("default e1-large-amd64 fleet is missing")
}

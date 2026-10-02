package runnerapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const maxRegistrationRequestBytes = 64 * 1024

type registerRunnerRequest struct {
	Version string `json:"version"`
}

type registerRunnerResponse struct {
	RunnerID    string `json:"runner_id"`
	FleetID     string `json:"fleet_id"`
	AccessToken string `json:"access_token"`
	Ephemeral   bool   `json:"ephemeral"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) registerRunner(w http.ResponseWriter, r *http.Request) {
	registrationToken, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "registration token is required")
		return
	}

	var request registerRunnerRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxRegistrationRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	request.Version = strings.TrimSpace(request.Version)
	if request.Version == "" {
		writeError(w, http.StatusBadRequest, "runner version is required")
		return
	}

	claims, err := ValidateRegistrationToken(s.signer, registrationToken)
	if err != nil {
		log.WithError(err).Warn("runner registration token validation failed")
		writeError(w, http.StatusUnauthorized, "registration token is invalid")
		return
	}
	runnerID, jti, taskID, err := registrationClaimIDs(claims)
	if err != nil {
		log.WithError(err).Warn("runner registration claims are invalid")
		writeError(w, http.StatusUnauthorized, "registration token is invalid")
		return
	}

	accessToken, err := crypto.Base64String(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "runner credential creation failed")
		return
	}

	var runner *models.Runner
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		runner, err = models.FindRunner(tx, runnerID)
		if err != nil {
			return err
		}
		return runner.Register(
			tx,
			jti,
			claims.FleetID,
			taskID,
			request.Version,
			crypto.HashToken(accessToken),
			time.Now(),
		)
	})
	switch {
	case errors.Is(err, models.ErrRunnerVersionMismatch):
		terminationErr := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
			current, findErr := models.FindRunner(tx, runnerID)
			if findErr != nil {
				return findErr
			}
			return current.Terminate(tx, models.RunnerTerminationVersionMismatch)
		})
		if terminationErr != nil {
			writeError(w, http.StatusInternalServerError, "runner registration failed")
			return
		}
		writeError(w, http.StatusConflict, "runner version does not match")
		return
	case errors.Is(err, models.ErrRunnerNotFound),
		errors.Is(err, models.ErrRunnerRegistrationInvalid):
		log.WithError(err).
			WithField("runner_id", runnerID).
			Warn("runner registration grant was rejected")
		writeError(w, http.StatusUnauthorized, "registration token is invalid")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "runner registration failed")
		return
	}

	writeJSON(w, http.StatusOK, registerRunnerResponse{
		RunnerID:    runner.ID.String(),
		FleetID:     claims.FleetID,
		AccessToken: accessToken,
		Ephemeral:   runner.Ephemeral,
	})
}

func registrationClaimIDs(claims *RegistrationClaims) (uuid.UUID, uuid.UUID, *uuid.UUID, error) {
	runnerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}
	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}
	if claims.TaskID == "" {
		return runnerID, jti, nil, nil
	}
	taskID, err := uuid.Parse(claims.TaskID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}
	return runnerID, jti, &taskID, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

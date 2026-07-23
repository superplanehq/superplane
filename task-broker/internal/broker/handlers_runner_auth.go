package broker

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/opaquetoken"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

const runnerRegistrationTTL = 10 * time.Minute

func (s *Server) createRunnerRegistration(w http.ResponseWriter, r *http.Request) {
	var req api.CreateRunnerRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	fleetID := strings.TrimSpace(req.FleetID)
	if fleetID == "" {
		writeError(w, http.StatusBadRequest, "fleet_id required")
		return
	}
	fleet, err := s.Store.GetFleet(r.Context(), fleetID)
	if err != nil {
		s.logErr("get fleet for runner registration", err)
		writeError(w, http.StatusInternalServerError, "could not load fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusNotFound, "fleet not found")
		return
	}

	token, err := opaquetoken.Generate()
	if err != nil {
		s.logErr("generate runner registration", err)
		writeError(w, http.StatusInternalServerError, "could not create registration")
		return
	}
	now := time.Now().UTC()
	expiresAt := now.Add(runnerRegistrationTTL)
	err = s.Store.CreateRunnerRegistration(r.Context(), &brokermodels.RunnerRegistration{
		TokenHash: opaquetoken.Hash(token),
		FleetID:   fleetID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	})
	if err != nil {
		s.logErr("persist runner registration", err)
		writeError(w, http.StatusInternalServerError, "could not create registration")
		return
	}
	writeJSON(w, http.StatusCreated, api.CreateRunnerRegistrationResponse{
		RegistrationToken: token,
		ExpiresAt:         expiresAt.Unix(),
	})
}

func (s *Server) registerRunner(w http.ResponseWriter, r *http.Request) {
	registrationToken := bearerToken(r)
	if registrationToken == "" {
		writeError(w, http.StatusUnauthorized, "registration token required")
		return
	}
	var req api.RegisterRunnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	runnerID := strings.TrimSpace(req.RunnerID)
	fleetID := strings.TrimSpace(req.FleetID)
	if runnerID == "" || fleetID == "" {
		writeError(w, http.StatusBadRequest, "runner_id and fleet_id required")
		return
	}
	accessToken, err := opaquetoken.Generate()
	if err != nil {
		s.logErr("generate runner credential", err)
		writeError(w, http.StatusInternalServerError, "could not register runner")
		return
	}
	err = s.Store.ExchangeRunnerRegistration(
		r.Context(),
		opaquetoken.Hash(registrationToken),
		runnerID,
		fleetID,
		opaquetoken.Hash(accessToken),
		time.Now().UTC(),
	)
	if errors.Is(err, taskstore.ErrInvalidRunnerRegistration) {
		writeError(w, http.StatusUnauthorized, "invalid or expired registration token")
		return
	}
	if err != nil {
		s.logErr("exchange runner registration", err)
		writeError(w, http.StatusInternalServerError, "could not register runner")
		return
	}
	writeJSON(w, http.StatusCreated, api.RegisterRunnerResponse{AccessToken: accessToken})
}

func (s *Server) revokeRunner(w http.ResponseWriter, r *http.Request) {
	identity, ok := runnerIdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := s.Store.DeleteRunnerCredentials(
		r.Context(), identity.FleetID, []string{identity.RunnerID},
	); err != nil {
		s.logErr("revoke runner credential", err)
		writeError(w, http.StatusInternalServerError, "could not revoke runner")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

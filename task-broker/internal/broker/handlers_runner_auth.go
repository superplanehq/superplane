package broker

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/opaquetoken"
	"github.com/superplane/runner/shared/runnerregistrationtoken"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

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
	claims, err := runnerregistrationtoken.Validate(registrationToken, fleetID, s.AuthToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired registration token")
		return
	}
	accessToken, err := opaquetoken.Generate()
	if err != nil {
		s.logErr("generate runner credential", err)
		writeError(w, http.StatusInternalServerError, "could not register runner")
		return
	}
	err = s.Store.RegisterRunnerWithJTI(
		r.Context(),
		claims.ID,
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
		s.logErr("register runner", err)
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

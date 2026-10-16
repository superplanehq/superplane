package public

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const maxCodexSubscriptionBodyBytes = 256*1024 + 4096

type codexCheckpointRequest struct {
	Bundle  json.RawMessage `json:"bundle"`
	Version int64           `json:"version"`
	Final   bool            `json:"final"`
	Usage   json.RawMessage `json:"usage"`
}

type codexProgressRequest struct {
	VerificationURL  string `json:"verificationUrl"`
	VerificationCode string `json:"verificationCode"`
}

func (s *Server) handleRunnerCodexSubscriptionCredentials(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	var bundle any
	var version int64
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		connection, lockErr := models.LockFactoryCodexConnection(tx, scope.OrganizationID, scope.FactoryID)
		if lockErr != nil {
			return lockErr
		}
		if err := connectionMatchesScope(connection, scope); err != nil {
			return err
		}
		if len(connection.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, decryptErr := s.encryptor.Decrypt(r.Context(), connection.BundleCiphertext, []byte(connection.ID.String()))
		if decryptErr != nil {
			return decryptErr
		}
		if unmarshalErr := json.Unmarshal(plain, &bundle); unmarshalErr != nil || bundle == nil {
			return errors.New("stored chatgpt subscription credentials are invalid")
		}
		version = connection.CredentialVersion
		return nil
	})
	if err != nil {
		writeCodexSubscriptionError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"bundle": bundle, "version": version})
}

func (s *Server) handleRunnerCodexSubscriptionCheckpoint(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexCheckpointRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	prepared, label, err := runneraction.PrepareCodexAuthBundle(request.Bundle)
	if err != nil {
		http.Error(w, "ChatGPT subscription credentials are invalid", http.StatusBadRequest)
		return
	}
	var usage struct {
		Limits []models.CodexUsageLimit `json:"limits"`
	}
	validUsage := json.Unmarshal(request.Usage, &usage) == nil && models.ValidateCodexUsageLimits(usage.Limits) == nil
	var version int64
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if eligibleErr := codexSubscriptionStillEligible(tx, scope.OrganizationID); eligibleErr != nil {
			return eligibleErr
		}
		connection, lockErr := models.LockFactoryCodexConnection(tx, scope.OrganizationID, scope.FactoryID)
		if lockErr != nil {
			return lockErr
		}
		if err := connectionMatchesScope(connection, scope); err != nil {
			return err
		}
		next := request.Version
		if next == 0 {
			next = connection.CredentialVersion + 1
		}
		ciphertext, encryptErr := s.encryptor.Encrypt(context.Background(), prepared, []byte(connection.ID.String()))
		if encryptErr != nil {
			return encryptErr
		}
		if storeErr := connection.StoreBundle(tx, scope.TaskID, scope.Generation, next, ciphertext, label, request.Final, time.Now()); storeErr != nil {
			return storeErr
		}
		// Usage is optional. Invalid quota data must not prevent a credential save.
		if validUsage {
			if err := connection.StoreUsage(tx, usage.Limits, time.Now()); err != nil {
				return err
			}
		}
		version = connection.CredentialVersion
		return nil
	})
	if err != nil {
		writeCodexSubscriptionError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "version": version})
}

func (s *Server) handleRunnerCodexSubscriptionProgress(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexProgressRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	verificationURL := strings.TrimSpace(request.VerificationURL)
	verificationCode := strings.TrimSpace(request.VerificationCode)
	if !strings.HasPrefix(verificationURL, "https://") || len(verificationURL) > 2048 || verificationCode == "" || len(verificationCode) > 64 {
		http.Error(w, "Invalid device sign-in details", http.StatusBadRequest)
		return
	}
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		connection, lockErr := models.LockFactoryCodexConnection(tx, scope.OrganizationID, scope.FactoryID)
		if lockErr != nil {
			return lockErr
		}
		return connection.RecordDeviceCode(tx, scope.TaskID, scope.Generation, verificationURL, verificationCode, time.Now())
	})
	if err != nil {
		writeCodexSubscriptionError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
}

func (s *Server) authenticateCodexSubscriptionRunner(w http.ResponseWriter, r *http.Request) (*runneraction.CodexSubscriptionScope, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParseCodexSubscriptionToken(s.jwt, token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func readCodexSubscriptionBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCodexSubscriptionBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil, err
	}
	return body, nil
}

func connectionMatchesScope(connection *models.FactoryCodexConnection, scope *runneraction.CodexSubscriptionScope) error {
	if connection.ID != scope.ConnectionID {
		return models.ErrCodexConnectionStale
	}
	return connectionRequire(connection, scope)
}

func connectionRequire(connection *models.FactoryCodexConnection, scope *runneraction.CodexSubscriptionScope) error {
	if connection.ReservationTaskID == nil || *connection.ReservationTaskID != scope.TaskID || connection.ReservationGeneration != scope.Generation {
		return models.ErrCodexConnectionStale
	}
	return nil
}

func codexSubscriptionStillEligible(tx *gorm.DB, organizationID uuid.UUID) error {
	_, err := models.SoleHumanOrganizationMember(tx, organizationID)
	return err
}

func writeCodexSubscriptionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrCodexConnectionNotFound):
		http.Error(w, "ChatGPT subscription connection not found", http.StatusNotFound)
	case errors.Is(err, models.ErrCodexConnectionStale):
		http.Error(w, "The ChatGPT sign-in update is out of date", http.StatusConflict)
	case errors.Is(err, models.ErrCodexConnectionNotReady):
		http.Error(w, "Connect ChatGPT before this task can run", http.StatusConflict)
	case errors.Is(err, models.ErrCodexConnectionNotEligible), errors.Is(err, models.ErrCodexConnectionUnavailable):
		http.Error(w, "This organization must have one member to use a ChatGPT subscription", http.StatusForbidden)
	default:
		http.Error(w, "ChatGPT subscription request failed", http.StatusInternalServerError)
	}
}

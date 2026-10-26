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

var errInvalidCodexBundle = errors.New("chatgpt subscription credentials are invalid")

type codexCheckpointRequest struct {
	Bundle  json.RawMessage `json:"bundle"`
	Version int64           `json:"version"`
	Final   bool            `json:"final"`
}

type codexProgressRequest struct {
	VerificationURL  string `json:"verificationUrl"`
	VerificationCode string `json:"verificationCode"`
}

type codexRefreshProxyRequest struct {
	RefreshToken string `json:"refresh_token"`
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
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if len(connection.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, decryptErr := s.encryptor.Decrypt(r.Context(), connection.BundleCiphertext, []byte(connection.ID.String()))
		if decryptErr != nil {
			return decryptErr
		}
		if lease.Purpose == models.CodexLeasePurposeRun {
			secret, secretErr := s.encryptor.Decrypt(r.Context(), lease.SecretCiphertext, []byte(connection.ID.String()))
			if secretErr != nil {
				return secretErr
			}
			plain, decryptErr = runneraction.CodexRunnerAuthBundle(plain, string(secret), time.Now())
			if decryptErr != nil {
				return decryptErr
			}
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
	var version int64
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if eligibleErr := codexSubscriptionStillEligible(tx, scope.OrganizationID); eligibleErr != nil {
			return eligibleErr
		}
		connection, lockErr := models.LockFactoryCodexConnection(tx, scope.OrganizationID, scope.FactoryID)
		if lockErr != nil {
			return lockErr
		}
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if lease.Purpose == models.CodexLeasePurposeRun {
			version = connection.CredentialVersion
			return nil
		}
		prepared, label, prepareErr := runneraction.PrepareCodexAuthBundle(request.Bundle)
		if prepareErr != nil {
			return errInvalidCodexBundle
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
		version = connection.CredentialVersion
		return nil
	})
	if errors.Is(err, errInvalidCodexBundle) {
		http.Error(w, "ChatGPT subscription credentials are invalid", http.StatusBadRequest)
		return
	}
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

func (s *Server) handleRunnerCodexSubscriptionRefresh(w http.ResponseWriter, r *http.Request) {
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexRefreshProxyRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	secret := strings.TrimSpace(request.RefreshToken)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var accessToken string
	var idToken string
	var permanent *runneraction.CodexTokenRefreshError
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		lease, err := models.FindCodexRunLeaseBySecret(tx, secret)
		if err != nil {
			return err
		}
		var connection models.FactoryCodexConnection
		if err := tx.Where("id = ?", lease.ConnectionID).First(&connection).Error; err != nil {
			return err
		}
		locked, err := models.LockFactoryCodexConnection(tx, connection.OrganizationID, connection.FactoryID)
		if err != nil {
			return err
		}
		if locked.Status == models.CodexConnectionStatusReconnectRequired {
			permanent = &runneraction.CodexTokenRefreshError{Permanent: true, Code: "refresh_token_invalidated"}
			return nil
		}
		if len(locked.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, err := s.encryptor.Decrypt(r.Context(), locked.BundleCiphertext, []byte(locked.ID.String()))
		if err != nil {
			return err
		}
		accessToken = runneraction.CodexAuthToken(plain, "access_token")
		idToken = runneraction.CodexAuthToken(plain, "id_token")
		if !runneraction.CodexAccessTokenNeedsRefresh(accessToken, time.Now()) {
			return nil
		}
		refreshed, refreshErr := s.refreshCodexToken(r.Context(), runneraction.CodexAuthToken(plain, "refresh_token"))
		var tokenErr *runneraction.CodexTokenRefreshError
		if errors.As(refreshErr, &tokenErr) && tokenErr.Permanent {
			if err := locked.MarkReconnectRequired(tx, time.Now()); err != nil {
				return err
			}
			permanent = tokenErr
			return nil
		}
		if refreshErr != nil {
			return refreshErr
		}
		if refreshed.RefreshToken == "" {
			refreshed.RefreshToken = runneraction.CodexAuthToken(plain, "refresh_token")
		}
		updated, err := runneraction.ReplaceCodexAuthTokens(plain, refreshed.AccessToken, refreshed.RefreshToken, refreshed.IDToken)
		if err != nil {
			return err
		}
		prepared, label, err := runneraction.PrepareCodexAuthBundle(updated)
		if err != nil {
			return err
		}
		ciphertext, err := s.encryptor.Encrypt(context.Background(), prepared, []byte(locked.ID.String()))
		if err != nil {
			return err
		}
		if err := locked.StoreRefreshedBundle(tx, ciphertext, label, time.Now()); err != nil {
			return err
		}
		accessToken = refreshed.AccessToken
		if refreshed.IDToken != "" {
			idToken = refreshed.IDToken
		}
		return nil
	})
	if permanent != nil {
		code := permanent.Code
		if code == "" {
			code = "refresh_token_invalidated"
		}
		_ = writeJSON(w, http.StatusUnauthorized, map[string]any{"error": code})
		return
	}
	if err != nil {
		var tokenErr *runneraction.CodexTokenRefreshError
		if errors.As(err, &tokenErr) {
			_ = writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "temporarily_unavailable"})
			return
		}
		if errors.Is(err, models.ErrCodexConnectionStale) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		writeCodexSubscriptionError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"id_token":      idToken,
		"refresh_token": secret,
	})
}

func (s *Server) refreshCodexToken(ctx context.Context, refreshToken string) (runneraction.CodexTokenSet, error) {
	if s.codexTokenRefresh != nil {
		return s.codexTokenRefresh(ctx, refreshToken)
	}
	return runneraction.RefreshCodexTokens(ctx, nil, refreshToken)
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

func scopedCodexLease(tx *gorm.DB, connection *models.FactoryCodexConnection, scope *runneraction.CodexSubscriptionScope) (*models.FactoryCodexLease, error) {
	if connection.ID != scope.ConnectionID {
		return nil, models.ErrCodexConnectionStale
	}
	return models.FindCodexLeaseForTask(tx, connection.ID, scope.TaskID, scope.Generation)
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

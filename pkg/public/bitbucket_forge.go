package public

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const forgeDeliveryBodyLimit = 1 << 20

type forgeDeliveryBody struct {
	EventType          string `json:"eventType"`
	InstallerAccountID string `json:"installerAccountId"`
}

// HandleBitbucketForgeDelivery accepts a Forge lifecycle, scheduled, or
// bootstrap call. The Forge Invocation Token is the credential. The system
// token is encrypted and cached. It is never written to the log.
func (s *Server) HandleBitbucketForgeDelivery(w http.ResponseWriter, r *http.Request) {
	s.handleBitbucketForgeDelivery(w, r, false)
}

// HandleBitbucketForgeUninstall accepts the Forge preUninstall call. Forge
// has no uninstall lifecycle event, so this route always clears the cached
// system token.
func (s *Server) HandleBitbucketForgeUninstall(w http.ResponseWriter, r *http.Request) {
	s.handleBitbucketForgeDelivery(w, r, true)
}

func (s *Server) handleBitbucketForgeDelivery(w http.ResponseWriter, r *http.Request, uninstallRoute bool) {
	cfg := config.LoadBitbucketForgeAppConfig()
	if !cfg.Enabled() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	systemToken := strings.TrimSpace(r.Header.Get(bitbucketappSystemTokenHeader()))
	keyfunc, err := bitbucketapp.VerificationKeyfunc(r.Context(), cfg.JWKSURL)
	if err != nil {
		log.WithError(err).Warn("failed to load Bitbucket Forge verification keys")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	invocation, err := bitbucketapp.ParseInvocation(bearerToken(r.Header.Get("Authorization")), systemToken, cfg.AppID, keyfunc)
	if err != nil {
		log.WithError(err).Warn("rejected Bitbucket Forge invocation")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := readForgeDeliveryBody(r)
	if err != nil {
		http.Error(w, "invalid forge delivery", http.StatusBadRequest)
		return
	}
	eventType := strings.TrimSpace(body.EventType)
	installerAccountID := strings.TrimSpace(body.InstallerAccountID)
	uninstall := uninstallRoute || strings.Contains(strings.ToLower(eventType), "uninstall")

	var ciphertext []byte
	if !uninstall {
		ciphertext, err = s.encryptor.Encrypt(r.Context(), []byte(invocation.SystemToken), []byte(invocation.InstallationID))
		if err != nil {
			log.WithError(err).WithField("installation_id", invocation.InstallationID).Error("failed to encrypt Bitbucket Forge token")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	_, err = models.SaveBitbucketForgeDelivery(database.DB(r.Context()), models.BitbucketForgeDelivery{
		InstallationID:     invocation.InstallationID,
		WorkspaceUUID:      invocation.WorkspaceUUID,
		InstallerAccountID: installerAccountID,
		APIBaseURL:         invocation.APIBaseURL,
		SystemToken:        ciphertext,
		TokenExpiresAt:     invocation.SystemTokenExpires,
		DeliveredAt:        time.Now(),
		Uninstall:          uninstall,
	})
	if err != nil {
		log.WithError(err).WithField("installation_id", invocation.InstallationID).Error("failed to store Bitbucket Forge delivery")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"installation_id":      invocation.InstallationID,
		"installer_account_id": installerAccountID,
		"workspace_uuid":       invocation.WorkspaceUUID,
		"event_type":           eventType,
		"token_expires_at":     invocation.SystemTokenExpires.UTC().Format(time.RFC3339),
		"token_valid_for":      time.Until(invocation.SystemTokenExpires).Truncate(time.Second).String(),
	}).Info("received Bitbucket Forge delivery")
	w.WriteHeader(http.StatusNoContent)
}

func bitbucketappSystemTokenHeader() string {
	return "x-forge-oauth-system"
}

func readForgeDeliveryBody(r *http.Request) (forgeDeliveryBody, error) {
	if r.Body == nil {
		return forgeDeliveryBody{}, nil
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, forgeDeliveryBodyLimit))
	if err != nil {
		return forgeDeliveryBody{}, err
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return forgeDeliveryBody{}, nil
	}
	var body forgeDeliveryBody
	if err := json.Unmarshal(payload, &body); err != nil {
		return forgeDeliveryBody{}, err
	}
	return body, nil
}

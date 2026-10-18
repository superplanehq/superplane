package public

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

const forgeDeliveryBodyLimit = 1 << 20

type forgeDeliveryBody struct {
	EventType          string `json:"eventType"`
	Timestamp          string `json:"timestamp"`
	InstallerAccountID string `json:"installerAccountId"`
	Workspace          struct {
		UUID string `json:"uuid"`
	} `json:"workspace"`
	Repository struct {
		UUID string `json:"uuid"`
	} `json:"repository"`
	PullRequest   map[string]any `json:"pullrequest"`
	BuildStatus   map[string]any `json:"buildStatus"`
	Actor         map[string]any `json:"actor"`
	Comment       map[string]any `json:"comment"`
	SelfGenerated bool           `json:"selfGenerated"`
}

// HandleBitbucketForgeDelivery accepts a Forge lifecycle, scheduled, or
// bootstrap call. The Forge Invocation Token is the credential. The system
// token is encrypted and cached. It is never written to the log.
func (s *Server) HandleBitbucketForgeDelivery(w http.ResponseWriter, r *http.Request) {
	s.handleBitbucketForgeDelivery(w, r, false, false)
}

// HandleBitbucketForgeUninstall accepts the Forge preUninstall call. Forge
// has no uninstall lifecycle event, so this route always clears the cached
// system token.
func (s *Server) HandleBitbucketForgeUninstall(w http.ResponseWriter, r *http.Request) {
	s.handleBitbucketForgeDelivery(w, r, true, false)
}

func (s *Server) HandleBitbucketForgeEvent(w http.ResponseWriter, r *http.Request) {
	s.handleBitbucketForgeDelivery(w, r, false, true)
}

func (s *Server) handleBitbucketForgeDelivery(w http.ResponseWriter, r *http.Request, uninstallRoute, eventRoute bool) {
	outcome := "rejected"
	defer func() { telemetry.RecordBitbucketForgeDelivery(r.Context(), outcome) }()
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
		log.WithError(err).WithField("installation_id", invocation.InstallationID).Warn("invalid Bitbucket Forge delivery body")
		http.Error(w, "invalid forge delivery", http.StatusBadRequest)
		return
	}
	eventType := strings.TrimSpace(body.EventType)
	if eventRoute {
		if bitbucket.ForgeEventKey(eventType) == "" ||
			invocation.WorkspaceUUID == "" || strings.Trim(body.Workspace.UUID, "{}") != invocation.WorkspaceUUID ||
			strings.Trim(body.Repository.UUID, "{}") == "" {
			http.Error(w, "invalid forge event", http.StatusBadRequest)
			return
		}
		if bitbucket.IsForgeBuildEvent(eventType) {
			if strings.TrimSpace(bitbucket.ForgeBuildKey(body.BuildStatus)) == "" ||
				!bitbucket.IsValidFullCommitSHA(bitbucket.ForgeBuildCommitSHA(body.BuildStatus)) {
				http.Error(w, "invalid forge event", http.StatusBadRequest)
				return
			}
		} else {
			var pr struct {
				ID int64 `json:"id"`
			}
			encoded, _ := json.Marshal(body.PullRequest)
			if json.Unmarshal(encoded, &pr) != nil || pr.ID <= 0 {
				http.Error(w, "invalid forge event", http.StatusBadRequest)
				return
			}
			if eventType == "avi:bitbucket:created:pullrequest-comment" {
				var comment struct {
					ID int64 `json:"id"`
				}
				encoded, _ := json.Marshal(body.Comment)
				if json.Unmarshal(encoded, &comment) != nil || comment.ID <= 0 {
					http.Error(w, "invalid forge event", http.StatusBadRequest)
					return
				}
			}
		}
	}
	installerAccountID := strings.TrimSpace(body.InstallerAccountID)
	uninstall := uninstallRoute || strings.Contains(strings.ToLower(eventType), "uninstall")

	var ciphertext []byte
	if !uninstall {
		ciphertext, err = s.encryptor.Encrypt(r.Context(), []byte(invocation.SystemToken), []byte(invocation.InstallationID))
		if err != nil {
			outcome = "failed"
			log.WithError(err).WithField("installation_id", invocation.InstallationID).Error("failed to encrypt Bitbucket Forge token")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	installation, err := models.SaveBitbucketForgeDelivery(database.DB(r.Context()), models.BitbucketForgeDelivery{
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
		outcome = "failed"
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
	if eventRoute && installation.UninstalledAt == nil && !(eventType == "avi:bitbucket:created:pullrequest-comment" && body.SelfGenerated) {
		if err := s.deliverBitbucketForgeEvent(r, invocation.InstallationID, body); err != nil {
			outcome = "failed"
			log.WithError(err).WithField("installation_id", invocation.InstallationID).Error("failed to deliver Bitbucket Forge event")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	outcome = "accepted"
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
	payload, err := io.ReadAll(io.LimitReader(r.Body, forgeDeliveryBodyLimit+1))
	if err != nil {
		return forgeDeliveryBody{}, err
	}
	if len(payload) > forgeDeliveryBodyLimit {
		return forgeDeliveryBody{}, fmt.Errorf("forge delivery exceeds size limit")
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

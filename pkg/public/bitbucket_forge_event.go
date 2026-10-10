package public

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Server) deliverBitbucketForgeEvent(r *http.Request, installationID string, event forgeDeliveryBody) error {
	repositoryUUID := strings.Trim(event.Repository.UUID, "{}")
	webhooks, err := models.ListBitbucketForgeWebhooks(database.DB(r.Context()), installationID, repositoryUUID)
	if err != nil {
		return err
	}
	var deliveryError error
	for _, webhook := range webhooks {
		var config bitbucket.WebhookConfiguration
		if err := mapstructure.Decode(webhook.Configuration.Data(), &config); err != nil {
			return err
		}
		if !slices.Contains(config.EventTypes, bitbucket.ForgeEventKey(event.EventType)) {
			continue
		}
		var metadata bitbucket.BitbucketWebhook
		if err := mapstructure.Decode(webhook.Metadata.Data(), &metadata); err != nil {
			return err
		}
		if metadata.RepositoryFullName == "" {
			return fmt.Errorf("missing repository name for webhook %s", webhook.ID)
		}
		var normalized map[string]any
		if bitbucket.IsForgeBuildEvent(event.EventType) {
			normalized = bitbucket.ForgeBuildStatusPayload(event.Timestamp, event.Repository.UUID, metadata.RepositoryFullName, event.BuildStatus, event.Actor)
		} else {
			normalized = bitbucket.ForgePullRequestPayload(event.EventType, event.Timestamp, event.Repository.UUID, metadata.RepositoryFullName, event.PullRequest, event.Actor)
			if event.Comment != nil {
				normalized["comment"] = event.Comment
			}
		}
		payload, err := json.Marshal(normalized)
		if err != nil {
			return err
		}
		secret, err := s.encryptor.Decrypt(r.Context(), webhook.Secret, []byte(webhook.ID.String()))
		if err != nil {
			return err
		}
		// Only verified Forge invocations reach this bridge; keep repository HMAC checks intact.
		mac := hmac.New(sha256.New, secret)
		_, _ = mac.Write(payload)
		request := httptest.NewRequest(http.MethodPost, r.URL.String(), bytes.NewReader(payload)).WithContext(r.Context())
		request.Header.Set("X-Event-Key", bitbucket.ForgeEventKey(event.EventType))
		request.Header.Set("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		request = mux.SetURLVars(request, map[string]string{"webhookID": webhook.ID.String()})
		response := httptest.NewRecorder()
		s.HandleWebhook(response, request)
		if response.Code >= http.StatusBadRequest && response.Code != http.StatusNotFound {
			deliveryError = fmt.Errorf("webhook %s returned status %d", webhook.ID, response.Code)
		}
	}
	return deliveryError
}

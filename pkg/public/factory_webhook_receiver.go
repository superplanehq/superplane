package public

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	webhooktrigger "github.com/superplanehq/superplane/pkg/triggers/webhook"
)

var factoryWebhookAuthenticationModes = []string{
	models.WebhookAuthenticationBearer,
	models.WebhookAuthenticationSignature,
	models.WebhookAuthenticationHeaderToken,
}

// authenticateFactoryWebhook checks a request to a webhook from the workspace
// settings once, before any listening automation sees it. It masks the token
// header in place.
func (s *Server) authenticateFactoryWebhook(ctx context.Context, webhook *models.Webhook, headers http.Header, body []byte) (int, error) {
	config := webhook.AuthenticationConfig()
	if !slices.Contains(factoryWebhookAuthenticationModes, config.Authentication) {
		return http.StatusUnauthorized, fmt.Errorf("webhook authentication is not configured")
	}

	secret, err := encryption.ForDB(database.DB(ctx), s.encryptor).Decrypt(ctx, webhook.Secret, []byte(webhook.ID.String()))
	if err != nil {
		log.WithError(err).WithField("webhook_id", webhook.ID.String()).Error("decrypt workspace webhook secret")
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	code, err := webhooktrigger.Authenticate(webhooktrigger.Configuration{
		Authentication:  config.Authentication,
		HeaderName:      config.HeaderName,
		SignatureHeader: config.HeaderName,
	}, secret, headers, body)
	if err != nil {
		return code, err
	}

	var parsed any
	if err := models.UnmarshalJSONValue(body, &parsed); err != nil {
		return http.StatusBadRequest, fmt.Errorf("send a JSON request body")
	}
	return http.StatusOK, nil
}

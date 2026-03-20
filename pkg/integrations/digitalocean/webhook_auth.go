package digitalocean

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

// rejectUnsupportedWebhook verifies webhook auth before rejecting unsupported webhook requests.
func rejectUnsupportedWebhook(ctx core.WebhookRequestContext, componentName string) (int, *core.WebhookResponseBody, error) {
	status, err := verifyWebhookBearerToken(ctx)
	if err != nil {
		return status, nil, err
	}

	return http.StatusMethodNotAllowed, nil, fmt.Errorf("%s does not support webhook requests", componentName)
}

func verifyWebhookBearerToken(ctx core.WebhookRequestContext) (int, error) {
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	authorization := ctx.Headers.Get("Authorization")
	if authorization == "" {
		return http.StatusUnauthorized, fmt.Errorf("missing Authorization header")
	}

	expected := "Bearer " + string(secret)
	if authorization != expected {
		return http.StatusUnauthorized, fmt.Errorf("invalid Bearer token")
	}

	ctx.Headers.Set("Authorization", "Bearer ********")
	return http.StatusOK, nil
}

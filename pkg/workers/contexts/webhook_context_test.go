package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestWebhookCallbackURL(t *testing.T) {
	webhookID := uuid.MustParse("282795a2-cb34-4162-896d-3b39d5a7e20b")

	cases := []struct {
		name    string
		baseURL string
	}{
		{name: "no trailing slash", baseURL: "https://sp.example.sslip.io"},
		{name: "trailing slash", baseURL: "https://sp.example.sslip.io/"},
		{name: "surrounding space", baseURL: " https://sp.example.sslip.io/ "},
	}

	want := "https://sp.example.sslip.io/api/v1/webhooks/" + webhookID.String()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, want, webhookCallbackURL(tc.baseURL, webhookID.String()))

			ctx := NewWebhookContext(nil, &models.Webhook{ID: webhookID}, nil, tc.baseURL)
			assert.Equal(t, want, ctx.GetURL())
		})
	}
}

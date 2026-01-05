package slack

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAppMention struct{}

type OnAppMentionConfiguration struct {
	Channel string `json:"channel"`
}

func (t *OnAppMention) Name() string {
	return "slack.onAppMention"
}

func (t *OnAppMention) Label() string {
	return "On App Mentioned"
}

func (t *OnAppMention) Description() string {
	return "Listen to messages mentioning the Slack App"
}

func (t *OnAppMention) Icon() string {
	return "slack"
}

func (t *OnAppMention) Color() string {
	return "gray"
}

func (t *OnAppMention) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "channel",
			Label:    "Channel",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (t *OnAppMention) Setup(ctx core.TriggerContext) error {
	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventTypes: []string{"app_mention"},
	})
}

func (t *OnAppMention) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAppMention) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAppMention) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

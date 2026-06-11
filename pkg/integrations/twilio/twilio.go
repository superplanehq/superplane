package twilio

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("twilio", &Twilio{})
}

type Twilio struct{}

func (t *Twilio) Name() string  { return "twilio" }
func (t *Twilio) Label() string { return "Twilio" }
func (t *Twilio) Icon() string  { return "twilio" }

func (t *Twilio) Description() string {
	return "Make phone calls, send SMS, and receive inbound messages with Twilio"
}

func (t *Twilio) Instructions() string {
	return `To set up the Twilio integration:

1. Log in to the **Twilio Console** (https://console.twilio.com)
2. Copy your **Account SID** and **Auth Token** from the dashboard
3. Get or buy a **Twilio phone number** with Voice and SMS capabilities
4. Paste all three values in the fields below

The phone number must be in E.164 format (e.g. +15551234567).`
}

func (t *Twilio) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "accountSid",
			Label:       "Account SID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Twilio Account SID from the Console dashboard",
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Twilio Auth Token from the Console dashboard",
		},
		{
			Name:        "fromNumber",
			Label:       "From Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "+15551234567",
			Description: "Twilio phone number for outbound calls and SMS (E.164 format)",
		},
	}
}

func (t *Twilio) Actions() []core.Action {
	return []core.Action{
		&MakeCall{},
		&SendSMS{},
	}
}

func (t *Twilio) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnInboundSMS{},
	}
}

func (t *Twilio) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.Integration)
	if err != nil {
		return err
	}

	account, err := client.GetAccount()
	if err != nil {
		return fmt.Errorf("failed to verify Twilio credentials: %v", err)
	}

	ctx.Integration.SetMetadata(map[string]string{
		"accountSid":   account.SID,
		"friendlyName": account.FriendlyName,
		"status":       account.Status,
	})

	ctx.Integration.Ready()
	return nil
}

func (t *Twilio) HandleRequest(ctx core.HTTPRequestContext) {}

func (t *Twilio) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (t *Twilio) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (t *Twilio) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *Twilio) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

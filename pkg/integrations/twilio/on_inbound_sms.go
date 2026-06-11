package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInboundSMS struct{}

func (t *OnInboundSMS) Name() string  { return "twilio.onInboundSMS" }
func (t *OnInboundSMS) Label() string { return "On Inbound SMS" }

func (t *OnInboundSMS) Description() string {
	return "Receive inbound SMS messages via Twilio webhook"
}

func (t *OnInboundSMS) Documentation() string {
	return `The **On Inbound SMS** trigger fires when an SMS is received on your Twilio phone number.

## Setup

1. Configure this trigger on your canvas
2. Copy the webhook URL shown after publishing
3. In the Twilio Console, go to your phone number settings
4. Set the **Messaging** webhook URL to the copied URL

## Event Data

Each inbound SMS includes:
- **from**: Sender phone number
- **to**: Your Twilio phone number that received the message
- **body**: The SMS message text
- **messageSid**: Twilio message ID
- **numMedia**: Number of media attachments

## Use Cases

- **Chatbot workflows**: Respond to inbound text messages
- **Acknowledgment flows**: On-call engineers can reply to alerts via SMS
- **Data collection**: Collect responses via text message`
}

func (t *OnInboundSMS) Icon() string  { return "message-square" }
func (t *OnInboundSMS) Color() string { return "#F22F46" }

func (t *OnInboundSMS) ExampleData() map[string]any {
	return getExampleData("on_inbound_sms")
}

func (t *OnInboundSMS) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnInboundSMS) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnInboundSMS) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnInboundSMS) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnInboundSMS) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnInboundSMS) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	// Parse the form-encoded body from Twilio
	params, err := url.ParseQuery(string(ctx.Body))
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	from := params.Get("From")
	to := params.Get("To")
	body := params.Get("Body")
	messageSid := params.Get("MessageSid")
	numMedia := params.Get("NumMedia")

	if messageSid == "" || from == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing required fields")
	}

	payload := map[string]any{
		"from":       from,
		"to":         to,
		"body":       body,
		"messageSid": messageSid,
		"numMedia":   numMedia,
	}

	ctx.Events.Emit("twilio.sms.received", payload)

	// Respond with empty TwiML to acknowledge
	twiml := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response></Response>"
	return http.StatusOK, &core.WebhookResponseBody{
		ContentType: "application/xml",
		Body:        []byte(twiml),
	}, nil
}

// ValidateSignature verifies the Twilio request signature.
// See: https://www.twilio.com/docs/usage/security#validating-requests
func ValidateSignature(authToken, requestURL, signature string, params map[string]string) bool {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(requestURL)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(params[k])
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(b.String()))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// Keep json import used for potential future use
var _ = json.Marshal

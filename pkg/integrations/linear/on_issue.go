package linear

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const onIssuePayloadType = "linear.issue"

type OnIssue struct{}

type OnIssueConfiguration struct {
	Team   string   `json:"team"`
	Labels []string `json:"labels"`
}

func (t *OnIssue) Name() string {
	return "linear.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Start a workflow when an issue is created, updated, or removed in Linear"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow when an issue is created, updated, or removed in Linear.

## Use Cases

- **Issue automation**: Run workflows when issues change
- **Notification workflows**: Notify channels or create tasks elsewhere
- **Filter by team/label**: Optionally restrict to a team and/or labels

## Configuration

- **Team**: Optional. Select a team to listen to, or leave empty to listen to all public teams.
- **Labels**: Optional. Only trigger when the issue has at least one of these labels.

## Event Data

The payload includes Linear webhook fields: action, type, data (issue), actor, url, createdAt, webhookTimestamp.
The action field indicates the event type: "create", "update", or "remove".`
}

func (t *OnIssue) Icon() string {
	return "linear"
}

func (t *OnIssue) Color() string {
	return "gray"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Limit to this team, or leave empty for all public teams",
			Placeholder: "Select a team (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger when the issue has at least one of these labels",
			Placeholder: "Select labels (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "label",
					Multi: true,
				},
			},
		},
	}
}

func (t *OnIssue) ExampleData() map[string]any {
	return UnmarshalExampleDataOnIssue()
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	var metadata NodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}

	if config.Team != "" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}
		team, err := client.FindTeam(config.Team)
		if err != nil {
			return err
		}
		metadata.Team = team
	}

	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID
	return ctx.Metadata.Set(metadata)
}

func (t *OnIssue) subscribe(ctx core.TriggerContext, metadata NodeMetadata) (*string, error) {
	if metadata.SubscriptionID != nil {
		logrus.Infof("using existing subscription %s", *metadata.SubscriptionID)
		return metadata.SubscriptionID, nil
	}

	logrus.Infof("creating new subscription")
	subscriptionID, err := ctx.Integration.Subscribe(struct{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnIssue) Actions() []core.Action {
	return nil
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (t *OnIssue) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	message, ok := ctx.Message.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected message type: %T", ctx.Message)
	}

	payload := LinearWebhookPayload{
		Action:    stringField(message, "action"),
		Type:      stringField(message, "type"),
		URL:       stringField(message, "url"),
		CreatedAt: stringField(message, "createdAt"),
	}

	if data, ok := message["data"].(map[string]any); ok {
		payload.Data = data
	}
	if actor, ok := message["actor"].(map[string]any); ok {
		payload.Actor = actor
	}
	if ts, ok := message["webhookTimestamp"].(float64); ok {
		payload.WebhookTimestamp = int64(ts)
	}

	if !onIssueAcceptPayload(&payload, config) {
		ctx.Logger.Infof("payload filtered out (action=%s type=%s), ignoring", payload.Action, payload.Type)
		return nil
	}

	emitPayload := map[string]any{
		"action":           payload.Action,
		"type":             payload.Type,
		"data":             payload.Data,
		"actor":            payload.Actor,
		"url":              payload.URL,
		"createdAt":        payload.CreatedAt,
		"webhookTimestamp": payload.WebhookTimestamp,
	}

	return ctx.Events.Emit(onIssuePayloadType, emitPayload)
}

// onIssueAcceptPayload returns true if the payload is an Issue event that passes team/label filters.
func onIssueAcceptPayload(payload *LinearWebhookPayload, config OnIssueConfiguration) bool {
	if payload.Type != "Issue" {
		return false
	}
	if config.Team != "" {
		teamID, _ := payload.Data["teamId"].(string)
		if teamID != config.Team {
			return false
		}
	}
	if len(config.Labels) > 0 {
		ids := payloadLabelIDs(payload.Data)
		if !slices.ContainsFunc(config.Labels, func(want string) bool { return slices.Contains(ids, want) }) {
			return false
		}
	}
	return true
}

func payloadLabelIDs(data map[string]any) []string {
	raw, _ := data["labelIds"].([]any)
	var ids []string
	for _, id := range raw {
		if s, ok := id.(string); ok {
			ids = append(ids, s)
		}
	}
	return ids
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// LinearWebhookPayload matches Linear's webhook POST body.
type LinearWebhookPayload struct {
	Action           string         `json:"action"`
	Type             string         `json:"type"`
	Data             map[string]any `json:"data"`
	Actor            map[string]any `json:"actor"`
	URL              string         `json:"url"`
	CreatedAt        string         `json:"createdAt"`
	WebhookTimestamp int64          `json:"webhookTimestamp"`
	UpdatedFrom      map[string]any `json:"updatedFrom,omitempty"`
}

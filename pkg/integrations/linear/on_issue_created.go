package linear

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

const onIssueCreatedPayloadType = "linear.issue.create"

type OnIssueCreated struct{}

type OnIssueCreatedConfiguration struct {
	Team   string   `json:"team"`
	Labels []string `json:"labels"`
}

func (t *OnIssueCreated) Name() string {
	return "linear.onIssueCreated"
}

func (t *OnIssueCreated) Label() string {
	return "On Issue Created"
}

func (t *OnIssueCreated) Description() string {
	return "Start a workflow when a new issue is created in Linear"
}

func (t *OnIssueCreated) Documentation() string {
	return `The On Issue Created trigger starts a workflow when a new issue is created in Linear.

## Use Cases

- **Issue automation**: Run workflows when new issues are created
- **Notification workflows**: Notify channels or create tasks elsewhere
- **Filter by team/label**: Optionally restrict to a team and/or labels

## Configuration

- **Team**: Optional. Select a team to listen to, or leave empty to listen to all public teams.
- **Labels**: Optional. Only trigger when the issue has at least one of these labels.

## Event Data

The payload includes Linear webhook fields: action, type, data (issue), actor, url, createdAt, webhookTimestamp.`
}

func (t *OnIssueCreated) Icon() string {
	return "linear"
}

func (t *OnIssueCreated) Color() string {
	return "gray"
}

func (t *OnIssueCreated) Configuration() []configuration.Field {
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

func (t *OnIssueCreated) ExampleData() map[string]any {
	return UnmarshalExampleDataOnIssueCreated()
}

func (t *OnIssueCreated) Setup(ctx core.TriggerContext) error {
	config := OnIssueCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if config.Team != "" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}
		teams, err := client.ListTeams()
		if err != nil {
			return fmt.Errorf("list teams: %w", err)
		}
		var team *Team
		for i := range teams {
			if teams[i].ID == config.Team {
				team = &teams[i]
				break
			}
		}
		if team == nil {
			return fmt.Errorf("team %s not found", config.Team)
		}
		if err := ctx.Metadata.Set(NodeMetadata{Team: team}); err != nil {
			return fmt.Errorf("set metadata: %w", err)
		}
	}

	webhookConfig := WebhookConfiguration{
		ResourceTypes:  []string{"Issue"},
		AllPublicTeams: config.Team == "",
	}
	if config.Team != "" {
		webhookConfig.TeamID = config.Team
	}

	return ctx.Integration.RequestWebhook(webhookConfig)
}

func (t *OnIssueCreated) Actions() []core.Action {
	return nil
}

func (t *OnIssueCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := verifyLinearSignature(ctx); err != nil {
		return errStatusCode(err), err
	}

	var payload LinearWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("parse body: %w", err)
	}

	var config OnIssueCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("decode configuration: %w", err)
	}

	if !onIssueCreatedAcceptPayload(&payload, config) {
		return http.StatusOK, nil
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
	if err := ctx.Events.Emit(onIssueCreatedPayloadType, emitPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("emit event: %w", err)
	}
	return http.StatusOK, nil
}

func verifyLinearSignature(ctx core.WebhookRequestContext) error {
	signature := ctx.Headers.Get("Linear-Signature")
	if signature == "" {
		return fmt.Errorf("missing Linear-Signature")
	}
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("get webhook secret: %w", err)
	}
	return crypto.VerifySignature(secret, ctx.Body, strings.TrimSpace(signature))
}

func errStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	// Prefer 403 for auth/signature; 500 for internal
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "signature") || strings.Contains(err.Error(), "Linear-Signature") {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

// onIssueCreatedAcceptPayload returns true if the payload should trigger (create + Issue, team/label filters pass).
func onIssueCreatedAcceptPayload(payload *LinearWebhookPayload, config OnIssueCreatedConfiguration) bool {
	if payload.Action != "create" || payload.Type != "Issue" {
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

func (t *OnIssueCreated) Cleanup(ctx core.TriggerContext) error {
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

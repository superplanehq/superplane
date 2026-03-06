package teams

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	setupInstructions = `
## Setup

1. **Create an Azure App Registration**:
   - Go to **Azure Portal** → **App registrations** → **New registration**
   - Name: "SuperPlane Bot" (or your preference)
   - Supported account types: **Accounts in any organizational directory** (multi-tenant) or single-tenant
   - Note the **Application (client) ID** — this is the **App ID** below
   - Go to **Certificates & secrets** → **New client secret** → copy the **Value** — this is the **App Password** below

2. **Add Graph API permissions** (required for channel listing):
   - Go to your **App Registration** → **API permissions** → **Add a permission**
   - Select **Microsoft Graph** → **Application permissions**
   - Add: **Team.ReadBasic.All** and **Channel.ReadBasic.All**
   - Click **Grant admin consent**

3. **Enter credentials** in the fields below and save
`

	installAppInstructionsTemplate = `
## Finish Azure Setup

Your webhook URL:
` + "`%s`" + `

### Create an Azure Bot

1. Go to **Azure Portal** → **Create a resource** → search **Azure Bot**
2. Link it to the App Registration from the previous step

### Configure the Bot

Open the Bot resource and go to **Settings** on the left sidebar:

1. **Configuration** → set the **Messaging endpoint** to the webhook URL above
2. **Channels** → click **Microsoft Teams** to enable it

### Install the Teams App

Click **Continue** to download the auto-generated Teams app manifest ZIP, then:

1. Upload it to **Teams Admin Center** → **Manage apps** → **Upload new app**, or sideload it directly
2. In the Teams app, open the target team → **Manage team** → **Apps** tab → **Get more apps** and add the newly uploaded app to the desired channel

> **Note:** Messages won't flow until all the steps above are completed.
`
)

func init() {
	registry.RegisterIntegration("teams", &Teams{})
}

// Teams implements the Microsoft Teams integration.
type Teams struct{}

// Metadata stores integration metadata after successful authentication.
type Metadata struct {
	AppID     string `json:"appId" mapstructure:"appId"`
	TenantID  string `json:"tenantId,omitempty" mapstructure:"tenantId,omitempty"`
	BotName   string `json:"botName,omitempty" mapstructure:"botName,omitempty"`
	Installed bool   `json:"installed,omitempty" mapstructure:"installed,omitempty"`
}

func (t *Teams) Name() string {
	return "teams"
}

func (t *Teams) Label() string {
	return "Microsoft Teams"
}

func (t *Teams) Icon() string {
	return "teams"
}

func (t *Teams) Description() string {
	return "Send and receive messages in Microsoft Teams channels"
}

func (t *Teams) Instructions() string {
	return setupInstructions
}

func (t *Teams) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "appId",
			Label:       "App ID",
			Type:        configuration.FieldTypeString,
			Description: "The Application (client) ID from Azure App Registration",
			Sensitive:   true,
			Required:    true,
		},
		{
			Name:        "appPassword",
			Label:       "App Password",
			Type:        configuration.FieldTypeString,
			Description: "The client secret value from Azure App Registration",
			Sensitive:   true,
			Required:    true,
		},
		{
			Name:        "tenantId",
			Label:       "Tenant ID",
			Type:        configuration.FieldTypeString,
			Description: "Azure Tenant ID (required for Graph API permissions to list channels)",
			Required:    true,
		},
		{
			Name:        "botName",
			Label:       "Bot Name",
			Type:        configuration.FieldTypeString,
			Description: "Display name for the Teams bot (used in the app manifest)",
			Required:    false,
			Placeholder: "SuperPlane Bot",
		},
	}
}

func (t *Teams) Components() []core.Component {
	return []core.Component{
		&SendTextMessage{},
	}
}

func (t *Teams) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnMention{},
		&OnMessage{},
	}
}

func (t *Teams) Sync(ctx core.SyncContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	appID, _ := ctx.Integration.GetConfig("appId")
	appPassword, _ := ctx.Integration.GetConfig("appPassword")

	if appID == nil || appPassword == nil || string(appID) == "" || string(appPassword) == "" {
		return t.createSetupPrompt(ctx)
	}

	var tenantID string
	tenantIDBytes, err := ctx.Integration.GetConfig("tenantId")
	if err == nil && tenantIDBytes != nil {
		tenantID = string(tenantIDBytes)
	}

	botName := "SuperPlane Bot"
	botNameBytes, err := ctx.Integration.GetConfig("botName")
	if err == nil && botNameBytes != nil && string(botNameBytes) != "" {
		botName = string(botNameBytes)
	}

	// Verify credentials if not already done
	if metadata.AppID == "" {
		client := NewClientFromConfig(string(appID), string(appPassword), tenantID)
		_, err = client.GetBotToken()
		if err != nil {
			return fmt.Errorf("failed to verify credentials: %v", err)
		}
	}

	// Check if the user signalled installation by setting "installed" in config.
	// We read from the raw configuration map directly because "installed" is not
	// a declared configuration field.
	installed := metadata.Installed
	if !installed {
		if cfgMap, ok := ctx.Configuration.(map[string]any); ok {
			if v, exists := cfgMap["installed"]; exists {
				installed = fmt.Sprintf("%v", v) == "true"
			}
		}
	}

	ctx.Integration.SetMetadata(Metadata{
		AppID:     string(appID),
		TenantID:  tenantID,
		BotName:   botName,
		Installed: installed,
	})

	// Always regenerate manifest ZIP so config changes (e.g. bot name) take effect
	webhookURL := fmt.Sprintf("%s/api/v1/integrations/%s/messages", ctx.WebhooksBaseURL, ctx.Integration.ID())
	instructions := fmt.Sprintf(installAppInstructionsTemplate, webhookURL)
	zipBytes, err := generateManifestZIP(string(appID), botName)
	if err != nil {
		ctx.Integration.RemoveBrowserAction()
	} else {
		dataURI := "data:application/zip;base64," + base64.StdEncoding.EncodeToString(zipBytes)
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: instructions,
			URL:         dataURI,
			Method:      "GET",
		})
	}

	// Only transition to ready once the user has downloaded the manifest
	// (signalled by setting installed=true via the Continue button).
	if installed {
		ctx.Integration.Ready()
	}

	return nil
}

func (t *Teams) createSetupPrompt(ctx core.SyncContext) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: setupInstructions,
		URL:         "https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade",
		Method:      "GET",
	})
	return nil
}

func (t *Teams) HandleRequest(ctx core.HTTPRequestContext) {
	// Read body
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("error reading request body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate JWT
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("error decoding metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		ctx.Logger.Errorf("missing Authorization header")
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		ctx.Logger.Errorf("invalid Authorization header format")
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	validator := NewJWTValidator(metadata.AppID)
	_, err = validator.ValidateToken(tokenString)
	if err != nil {
		ctx.Logger.Errorf("JWT validation failed: %v", err)
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse activity
	var activity Activity
	if err := json.Unmarshal(body, &activity); err != nil {
		ctx.Logger.Errorf("error parsing activity: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	switch activity.Type {
	case "message":
		t.handleMessage(ctx, activity)
	case "conversationUpdate":
		t.handleConversationUpdate(ctx, activity)
	default:
		ctx.Logger.Infof("ignoring activity type: %s", activity.Type)
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (t *Teams) handleMessage(ctx core.HTTPRequestContext, activity Activity) {
	ctx.Logger.Infof("handling message activity: id=%s, from=%s, conversation=%s, text=%q",
		activity.ID, activity.From.Name, activity.Conversation.ID, activity.Text)

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		return
	}

	ctx.Logger.Infof("found %d subscriptions", len(subscriptions))

	isMention := hasBotMention(activity)
	ctx.Logger.Infof("is bot mention: %v", isMention)

	eventData := map[string]any{
		"type":       activity.Type,
		"id":         activity.ID,
		"timestamp":  activity.Timestamp,
		"channelId":  activity.ChannelID,
		"serviceUrl": activity.ServiceURL,
		"from": map[string]any{
			"id":          activity.From.ID,
			"name":        activity.From.Name,
			"aadObjectId": activity.From.AADObjectID,
		},
		"conversation": map[string]any{
			"id":               activity.Conversation.ID,
			"name":             activity.Conversation.Name,
			"isGroup":          activity.Conversation.IsGroup,
			"conversationType": activity.Conversation.ConversationType,
			"tenantId":         activity.Conversation.TenantID,
		},
		"recipient": map[string]any{
			"id":          activity.Recipient.ID,
			"name":        activity.Recipient.Name,
			"aadObjectId": activity.Recipient.AADObjectID,
		},
		"text":        activity.Text,
		"entities":    activity.Entities,
		"channelData": activity.ChannelData,
	}

	for i, subscription := range subscriptions {
		c := SubscriptionConfiguration{}
		if err := mapstructure.Decode(subscription.Configuration(), &c); err != nil {
			ctx.Logger.Errorf("error decoding subscription %d configuration: %v", i, err)
			continue
		}

		ctx.Logger.Infof("subscription %d: eventTypes=%v", i, c.EventTypes)

		if isMention && slices.Contains(c.EventTypes, "mention") {
			ctx.Logger.Infof("dispatching mention event to subscription %d", i)
			if err := subscription.SendMessage(eventData); err != nil {
				ctx.Logger.Errorf("error sending mention message: %v", err)
			}
		}

		if slices.Contains(c.EventTypes, "message") {
			ctx.Logger.Infof("dispatching message event to subscription %d", i)
			if err := subscription.SendMessage(eventData); err != nil {
				ctx.Logger.Errorf("error sending message: %v", err)
			}
		}
	}
}

func (t *Teams) handleConversationUpdate(ctx core.HTTPRequestContext, activity Activity) {
	ctx.Logger.Infof("conversation update: %s (members added: %d, removed: %d)",
		activity.Conversation.ID,
		len(activity.MembersAdded),
		len(activity.MembersRemoved),
	)
}

// SubscriptionConfiguration defines which event types a trigger subscribes to.
type SubscriptionConfiguration struct {
	EventTypes []string `json:"eventTypes"`
}

func hasBotMention(activity Activity) bool {
	for _, entity := range activity.Entities {
		if entity.Type == "mention" && entity.Mentioned != nil {
			if entity.Mentioned.ID == activity.Recipient.ID {
				return true
			}
		}
	}

	return false
}

func (t *Teams) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (t *Teams) Actions() []core.Action {
	return []core.Action{}
}

func (t *Teams) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

// generateManifestZIP creates a Teams app manifest ZIP ready for upload to Teams Admin Center.
func generateManifestZIP(appID, botName string) ([]byte, error) {
	manifest := map[string]any{
		"$schema":                 "https://developer.microsoft.com/en-us/json-schemas/teams/v1.25/MicrosoftTeams.schema.json",
		"manifestVersion":         "1.25",
		"supportsChannelFeatures": "tier1",
		"version":                 "1.0.0",
		"id":                      appID,
		"developer": map[string]string{
			"name":          "SuperPlane",
			"websiteUrl":    "https://superplane.com/",
			"privacyUrl":    "https://superplane.com/privacy",
			"termsOfUseUrl": "https://superplane.com/terms",
		},
		"name": map[string]string{
			"short": botName,
		},
		"description": map[string]string{
			"short": "SuperPlane workflow automation bot",
			"full":  "Connects Microsoft Teams with SuperPlane for workflow automation, notifications, and team collaboration.",
		},
		"icons": map[string]string{
			"color":   "color.png",
			"outline": "outline.png",
		},
		"accentColor": "#FFFFFF",
		"bots": []map[string]any{
			{
				"botId": appID,
				"scopes": []string{
					"groupChat",
					"team",
					"personal",
				},
				"isNotificationOnly": false,
				"supportsFiles":      false,
				"supportsCalling":    false,
				"supportsVideo":      false,
				"commandLists": []map[string]any{
					{
						"scopes": []string{"team", "groupChat"},
						"commands": []map[string]string{
							{
								"title":       "help",
								"description": "Show available commands",
							},
						},
					},
				},
			},
		},
		"validDomains": []string{},
		"webApplicationInfo": map[string]string{
			"id": appID,
		},
		"authorization": map[string]any{
			"permissions": map[string]any{
				"resourceSpecific": []map[string]string{
					{
						"name": "ChannelMessage.Read.Group",
						"type": "Application",
					},
					{
						"name": "ChannelMessage.Send.Group",
						"type": "Application",
					},
					{
						"name": "ChatMessage.Read.Chat",
						"type": "Application",
					},
					{
						"name": "ChatMessage.Send.Chat",
						"type": "Application",
					},
				},
			},
		},
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// manifest.json
	f, err := w.Create("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest entry: %w", err)
	}
	if _, err := f.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// color.png — Teams purple placeholder icon
	colorPNG, err := solidPNG(192, color.NRGBA{R: 0x50, G: 0x59, B: 0xC9, A: 0xFF})
	if err != nil {
		return nil, fmt.Errorf("failed to generate color.png: %w", err)
	}
	f, err = w.Create("color.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create color.png entry: %w", err)
	}
	if _, err := f.Write(colorPNG); err != nil {
		return nil, fmt.Errorf("failed to write color.png: %w", err)
	}

	// outline.png — white icon on transparent background (required by Teams)
	outlinePNG, err := outlineIconPNG(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate outline.png: %w", err)
	}
	f, err = w.Create("outline.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create outline.png entry: %w", err)
	}
	if _, err := f.Write(outlinePNG); err != nil {
		return nil, fmt.Errorf("failed to write outline.png: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// solidPNG generates a solid-color PNG image of the given size.
func solidPNG(size int, c color.Color) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// outlineIconPNG generates a white rounded-rectangle on a transparent background.
// This satisfies Teams' outline icon requirement: transparent PNG with white content.
func outlineIconPNG(size int) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	white := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

	// Draw a white rounded rectangle with 2px padding and ~4px corner radius.
	pad := 2
	radius := 4
	for y := pad; y < size-pad; y++ {
		for x := pad; x < size-pad; x++ {
			// Check corners for rounding
			if inRoundedRect(x, y, pad, pad, size-pad-1, size-pad-1, radius) {
				img.Set(x, y, white)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// inRoundedRect checks if (x, y) is inside a rounded rectangle defined by
// (x0, y0)-(x1, y1) with the given corner radius.
func inRoundedRect(x, y, x0, y0, x1, y1, r int) bool {
	// Inside the main body (not in a corner zone)
	if x >= x0+r && x <= x1-r {
		return true
	}
	if y >= y0+r && y <= y1-r {
		return true
	}

	// Check each corner
	corners := [][2]int{
		{x0 + r, y0 + r}, // top-left
		{x1 - r, y0 + r}, // top-right
		{x0 + r, y1 - r}, // bottom-left
		{x1 - r, y1 - r}, // bottom-right
	}

	for _, c := range corners {
		dx := x - c[0]
		dy := y - c[1]
		if dx*dx+dy*dy <= r*r {
			return true
		}
	}

	return false
}

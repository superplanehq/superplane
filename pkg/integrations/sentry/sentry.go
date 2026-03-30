package sentry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	DefaultBaseURL = "https://sentry.io"

	ResourceTypeProject     = "project"
	ResourceTypeTeam        = "team"
	ResourceTypeIssue       = "issue"
	ResourceTypeAssignee    = "assignee"
	ResourceTypeAlert       = "alert"
	ResourceTypeAlertTarget = "alert-target"
	ResourceTypeRelease     = "release"

	SentryPersonalTokensURL = "https://sentry.io/settings/account/api/auth-tokens/"
)

const (
	appSetupDescription = `
1. Create a [personal auth token](` + SentryPersonalTokensURL + `) in Sentry. Copy the token.

   > **Token Permissions:**  
   > ` + "`Project -> Read`" + ` · ` + "`Releases -> Read & Write (project:releases)`" + ` · ` + "`Team -> Read`" + ` · ` + "`Issue & Event -> Read & Write`" + ` · ` + "`Organization -> Read & Write`" + `

2. In Sentry, go to **Settings → Integrations → Custom Integrations → Create New Integration → Internal Integration**.
3. Name it ` + "`%s`" + `, leave **Webhook URL** empty, and save. Copy the **Client Secret** shown on the bottom of the integration page.
4. Fill in **Sentry URL**, **User Token**, **Integration Name**, and **Client Secret** below, then save. SuperPlane configures the webhook and subscribes to issue events automatically.
`

	manualWebhookDescription = `
- SuperPlane connected to Sentry, but it could not automatically configure the internal integration webhook.
- Click **Continue** to open the **Custom Integrations** page in Sentry.
- Open the ` + "`%s`" + ` integration and set:
  - **Webhook URL**: ` + "`%s`" + `
  - **Webhook Subscriptions**: ` + "`issue`" + `
`

	missingIntegrationDescription = `
- SuperPlane connected to Sentry, but it could not find an internal integration named ` + "`%s`" + `.
- Click **Continue** to open the **Custom Integrations** page in Sentry.
- Create an internal integration with that exact name, or update **Integration Name** in SuperPlane to match the name already in Sentry.
- Save the integration in SuperPlane again after fixing the name so SuperPlane can configure the webhook automatically.
`

	multipleIntegrationsDescription = `
- SuperPlane connected to Sentry, but multiple internal integrations matched the name ` + "`%s`" + `.
- Click **Continue** to open the **Custom Integrations** page in Sentry.
- Rename one of the matching integrations in Sentry or choose a unique **Integration Name** in SuperPlane.
- Save the integration in SuperPlane again after fixing the name so SuperPlane can configure the webhook automatically.
`

	noIntegrationsDescription = `
- SuperPlane connected to Sentry, but there are no custom integrations in this Sentry organization yet.
- Click **Continue** to open the **Custom Integrations** page in Sentry.
- Create the internal integration first, then save the integration in SuperPlane again so SuperPlane can configure the webhook automatically.
`
)

func init() {
	registry.RegisterIntegration("sentry", &Sentry{})
}

type Sentry struct{}

type Configuration struct {
	BaseURL         string `json:"baseUrl" mapstructure:"baseUrl"`
	IntegrationName string `json:"integrationName" mapstructure:"integrationName"`
	UserToken       string `json:"userToken" mapstructure:"userToken"`
	ClientSecret    string `json:"clientSecret" mapstructure:"clientSecret"`
}

type Metadata struct {
	AppSlug      string               `json:"appSlug" mapstructure:"appSlug"`
	Organization *OrganizationSummary `json:"organization,omitempty" mapstructure:"organization,omitempty"`
	Projects     []ProjectSummary     `json:"projects" mapstructure:"projects"`
	Teams        []TeamSummary        `json:"teams" mapstructure:"teams"`
}

type OrganizationSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type ProjectSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type TeamSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type SubscriptionConfiguration struct {
	Resources []string `json:"resources" mapstructure:"resources"`
}

type WebhookInstallation struct {
	UUID string `json:"uuid" mapstructure:"uuid"`
}

type WebhookMessage struct {
	Resource     string              `json:"resource" mapstructure:"resource"`
	Action       string              `json:"action" mapstructure:"action"`
	Timestamp    string              `json:"timestamp,omitempty" mapstructure:"timestamp,omitempty"`
	Installation WebhookInstallation `json:"installation" mapstructure:"installation"`
	Data         map[string]any      `json:"data" mapstructure:"data"`
	Actor        map[string]any      `json:"actor,omitempty" mapstructure:"actor,omitempty"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "bug"
}

func (s *Sentry) Description() string {
	return "React to issue events and manage issues and metric alerts in Sentry"
}

func (s *Sentry) Instructions() string {
	return `

**Setup steps:**
1. Create a [personal auth token](` + SentryPersonalTokensURL + `) in Sentry with the permissions below. Copy the token.

   > **Token Permissions:**  
   > ` + "Project -> `Read`" + ` · ` + "Releases -> `Read & Write` (`project:releases`)" + ` · ` + "Team -> `Read`" + ` · ` + "Issue & Event -> `Read & Write`" + ` · ` + "Organization -> `Read & Write`" + `

2. In Sentry, go to **Settings → Integrations → Custom Integrations → Create New Integration → Internal Integration**.
3. Name it ` + "(e.g. `SuperPlane`)" + `, leave **Webhook URL** empty, and save. Copy the **Client Secret** shown on the bottom of the integration page.
4. Fill in **Sentry URL**, **User Token**, **Integration Name**, and **Client Secret** below, then save. SuperPlane configures the webhook and subscribes to issue events automatically.

`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Sentry Organization URL",
			Type:        configuration.FieldTypeString,
			Description: "Sentry instance URL. Use your org-specific URL (e.g., https://your-org.sentry.io) so SuperPlane can identify your organization automatically.",
			Default:     "",
			Required:    true,
		},
		{
			Name:        "userToken",
			Label:       "User Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Personal auth token from Sentry. Include `project:releases` if you use release actions.",
			Required:    true,
		},
		{
			Name:        "integrationName",
			Label:       "Integration Name",
			Type:        configuration.FieldTypeString,
			Description: "Name of the Sentry internal integration that SuperPlane should manage",
			Default:     "SuperPlane",
			Required:    false,
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Client secret from your Sentry internal integration, used to verify incoming webhooks.",
			Required:    true,
		},
	}
}

func (s *Sentry) Components() []core.Component {
	return []core.Component{
		&CreateAlert{},
		&UpdateAlert{},
		&DeleteAlert{},
		&ListAlerts{},
		&GetAlert{},
		&GetIssue{},
		&CreateRelease{},
		&CreateDeploy{},
		&UpdateIssue{},
	}
}

func (s *Sentry) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config, err := s.loadConfiguration(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if strings.TrimSpace(config.UserToken) == "" || strings.TrimSpace(config.ClientSecret) == "" {
		return s.createSetupPrompt(ctx, config)
	}

	if err := s.updateMetadata(ctx, config); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	warning, err := s.reconcileWebhook(ctx, config)
	if err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	if warning == "" {
		ctx.Integration.RemoveBrowserAction()
	} else {
		metadata := Metadata{}
		_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
		orgSlug := ""
		if metadata.Organization != nil {
			orgSlug = metadata.Organization.Slug
		}
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: webhookBrowserActionDescription(config.IntegrationName, warning, eventsURL(ctx)),
			URL:         developerSettingsURL(config.BaseURL, orgSlug),
			Method:      http.MethodGet,
		})

		if syncWarningShouldError(warning) {
			ctx.Integration.Error(warning)
			return nil
		}
	}

	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) createSetupPrompt(ctx core.SyncContext, config Configuration) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(
			appSetupDescription,
			displayIntegrationName(config.IntegrationName),
		),
		URL:    newInternalIntegrationURL(config.BaseURL),
		Method: http.MethodGet,
	})
	ctx.Integration.Error(missingCredentialsMessage(config))
	return nil
}

func (s *Sentry) reconcileWebhook(ctx core.SyncContext, config Configuration) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return "", fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Organization == nil || metadata.Organization.Slug == "" {
		return "", nil
	}

	client := NewAPIClient(ctx.HTTP, config.BaseURL, config.UserToken)

	apps, err := client.ListSentryApps(metadata.Organization.Slug)
	if err != nil {
		return "failed to list internal integrations for automatic webhook configuration", nil
	}

	app, warning := findTargetApp(apps, config.IntegrationName)
	if app == nil {
		return warning, nil
	}

	desiredWebhookURL := eventsURL(ctx)
	desiredEvents := ensureContains(app.Events, "issue")
	// issue webhooks require event:read — ensure it's in the scopes.
	desiredScopes := ensureContains(app.Scopes, "event:read")

	if strings.TrimSpace(app.WebhookURL) == desiredWebhookURL &&
		sameStringSet(app.Events, desiredEvents) &&
		sameStringSet(app.Scopes, desiredScopes) {
		metadata.AppSlug = app.Slug
		ctx.Integration.SetMetadata(metadata)
		return "", nil
	}

	updatedApp, err := client.UpdateSentryApp(app.Slug, UpdateSentryAppRequest{
		Name:           app.Name,
		Scopes:         desiredScopes,
		Events:         desiredEvents,
		WebhookURL:     desiredWebhookURL,
		RedirectURL:    app.RedirectURL,
		IsInternal:     app.IsInternal,
		IsAlertable:    app.IsAlertable,
		Overview:       app.Overview,
		VerifyInstall:  app.VerifyInstall,
		AllowedOrigins: app.AllowedOrigins,
		Author:         app.Author,
		Schema:         app.Schema,
	})
	if err != nil {
		return fmt.Sprintf("failed to update internal integration webhook automatically: %v", err), nil
	}

	// Sentry rotates the client secret when the app is updated via PUT.
	// Store the new secret so webhook signature verification keeps working
	// without requiring the user to manually update the integration.
	if updatedApp.ClientSecret != "" {
		if err := ctx.Integration.SetSecret("clientSecret", []byte(updatedApp.ClientSecret)); err != nil {
			ctx.Logger.Warnf("failed to store rotated sentry client secret: %v", err)
		}
	}

	metadata.AppSlug = app.Slug
	ctx.Integration.SetMetadata(metadata)
	return "", nil
}

func findTargetApp(apps []SentryApp, integrationName string) (*SentryApp, string) {
	name := strings.TrimSpace(integrationName)
	if name != "" {
		matches := make([]SentryApp, 0, 1)
		for _, app := range apps {
			if strings.EqualFold(strings.TrimSpace(app.Name), name) {
				matches = append(matches, app)
			}
		}

		switch len(matches) {
		case 0:
			return nil, fmt.Sprintf("could not find an internal integration named %q", name)
		case 1:
			copy := matches[0]
			return &copy, ""
		default:
			return nil, fmt.Sprintf("multiple internal integrations named %q were found; rename one of them or choose a unique Integration Name in SuperPlane", name)
		}
	}

	if len(apps) == 0 {
		return nil, "no custom integrations were found for this organization"
	}

	if len(apps) == 1 {
		copy := apps[0]
		return &copy, ""
	}

	return nil, "multiple custom integrations exist for this organization; set Integration Name in SuperPlane or configure the webhook URL manually"
}

func webhookBrowserActionDescription(integrationName, warning, webhookURL string) string {
	name := displayIntegrationName(integrationName)

	switch {
	case strings.HasPrefix(warning, "could not find an internal integration named"):
		return fmt.Sprintf(missingIntegrationDescription, name)
	case strings.HasPrefix(warning, "multiple internal integrations named"):
		return fmt.Sprintf(multipleIntegrationsDescription, name)
	case warning == "no custom integrations were found for this organization":
		return noIntegrationsDescription
	default:
		return fmt.Sprintf(manualWebhookDescription, name, webhookURL)
	}
}

func syncWarningShouldError(warning string) bool {
	switch {
	case strings.HasPrefix(warning, "could not find an internal integration named"):
		return true
	case strings.HasPrefix(warning, "multiple internal integrations named"):
		return true
	case warning == "no custom integrations were found for this organization":
		return true
	default:
		return false
	}
}

func ensureContains(values []string, needle string) []string {
	if slices.Contains(values, needle) {
		return values
	}

	result := append([]string{}, values...)
	result = append(result, needle)
	return result
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[string]int, len(a))
	for _, value := range a {
		counts[value]++
	}

	for _, value := range b {
		count, exists := counts[value]
		if !exists || count == 0 {
			return false
		}
		counts[value]--
	}

	for _, count := range counts {
		if count != 0 {
			return false
		}
	}

	return true
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Sentry) Actions() []core.Action {
	return []core.Action{}
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	s.handleWebhook(ctx)
}

func (s *Sentry) handleWebhook(ctx core.HTTPRequestContext) {
	config, err := s.loadConfiguration(ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("failed to load sentry config: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read sentry webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(config.ClientSecret) == "" {
		ctx.Logger.Warn("missing sentry client secret for webhook signature verification")
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	if err := verifyWebhookSignature(ctx.Request.Header.Get("Sentry-Hook-Signature"), body, []byte(config.ClientSecret)); err != nil {
		ctx.Logger.Warnf("invalid sentry webhook signature: %v", err)
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	resource := strings.TrimSpace(ctx.Request.Header.Get("Sentry-Hook-Resource"))
	if resource == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Action       string              `json:"action"`
		Timestamp    string              `json:"timestamp"`
		Installation WebhookInstallation `json:"installation"`
		Data         map[string]any      `json:"data"`
		Actor        map[string]any      `json:"actor"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to decode sentry webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	message := WebhookMessage{
		Resource:     resource,
		Action:       payload.Action,
		Timestamp:    payload.Timestamp,
		Installation: payload.Installation,
		Data:         payload.Data,
		Actor:        payload.Actor,
	}

	if err := s.dispatchWebhookMessage(ctx, message); err != nil {
		ctx.Logger.Errorf("failed to dispatch sentry webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (s *Sentry) dispatchWebhookMessage(ctx core.HTTPRequestContext, message WebhookMessage) error {
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to list sentry subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		config := SubscriptionConfiguration{}
		if err := mapstructure.Decode(subscription.Configuration(), &config); err != nil {
			ctx.Logger.Warnf("failed to decode sentry subscription config: %v", err)
			continue
		}

		if len(config.Resources) > 0 && !slices.Contains(config.Resources, message.Resource) {
			continue
		}

		if err := subscription.SendMessage(message); err != nil {
			ctx.Logger.Errorf("failed to send sentry message to subscription: %v", err)
		}
	}

	return nil
}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	switch resourceType {
	case ResourceTypeProject:
		resources := make([]core.IntegrationResource, 0, len(metadata.Projects))
		for _, project := range metadata.Projects {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeProject,
				ID:   project.Slug,
				Name: project.Name,
			})
		}
		return resources, nil

	case ResourceTypeTeam:
		resources := make([]core.IntegrationResource, 0, len(metadata.Teams))
		for _, team := range metadata.Teams {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTeam,
				ID:   team.Slug,
				Name: team.Name,
			})
		}
		return resources, nil

	case ResourceTypeIssue:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry client: %w", err)
		}

		issues, err := client.ListIssues()
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(issues))
		for _, issue := range issues {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeIssue,
				ID:   issue.ID,
				Name: displayIssueLabel(issue.ShortID, issue.Title),
			})
		}

		return resources, nil

	case ResourceTypeAssignee:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry client: %w", err)
		}

		issueID := strings.TrimSpace(ctx.Parameters["issueId"])
		if issueID == "" {
			return []core.IntegrationResource{}, nil
		}

		issue, err := client.GetIssue(issueID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve issue for assignee lookup: %w", err)
		}

		projectSlug := ""
		if issue.Project != nil {
			projectSlug = strings.TrimSpace(issue.Project.Slug)
		}
		if projectSlug == "" {
			return []core.IntegrationResource{}, nil
		}

		assignees, err := client.ListProjectAssignees(projectSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to list assignees: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(assignees))
		for _, assignee := range assignees {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeAssignee,
				ID:   assignee.ID,
				Name: assignee.Name,
			})
		}

		return resources, nil

	case ResourceTypeAlert:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry client: %w", err)
		}

		projectSlug := strings.TrimSpace(ctx.Parameters["project"])

		alertRules, err := client.ListAlertRules()
		if err != nil {
			return nil, fmt.Errorf("failed to list alerts: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(alertRules))
		for _, alertRule := range alertRules {
			if projectSlug != "" && !alertRuleContainsProject(alertRule, projectSlug) {
				continue
			}

			alertID := strings.TrimSpace(alertRule.ID)
			alertName := strings.TrimSpace(alertRule.Name)
			if alertID == "" || alertName == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeAlert,
				ID:   alertID,
				Name: displayAlertRuleLabel(alertRule),
			})
		}

		return resources, nil

	case ResourceTypeAlertTarget:
		targetType := strings.TrimSpace(ctx.Parameters["targetType"])
		if targetType == "" {
			return []core.IntegrationResource{}, nil
		}
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry client: %w", err)
		}

		projectSlug := strings.TrimSpace(ctx.Parameters["project"])
		if projectSlug == "" {
			alertID := strings.TrimSpace(ctx.Parameters["alertId"])
			if alertID != "" {
				alertRule, err := client.GetAlertRule(alertID)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve alert for target lookup: %w", err)
				}

				if len(alertRule.Projects) > 0 {
					projectSlug = strings.TrimSpace(alertRule.Projects[0])
				}
			}
		}

		if projectSlug == "" {
			return []core.IntegrationResource{}, nil
		}

		switch targetType {
		case alertTargetTypeUser:
			members, err := client.ListProjectMembers(projectSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to list project members: %w", err)
			}

			resources := make([]core.IntegrationResource, 0, len(members))
			for _, member := range members {
				value := ""
				label := ""

				if member.User != nil {
					value = strings.TrimSpace(member.User.ID)
					switch {
					case strings.TrimSpace(member.User.Name) != "":
						label = strings.TrimSpace(member.User.Name)
					case strings.TrimSpace(member.User.Email) != "":
						label = strings.TrimSpace(member.User.Email)
					case strings.TrimSpace(member.User.Username) != "":
						label = strings.TrimSpace(member.User.Username)
					}
				}

				if value == "" {
					value = strings.TrimSpace(member.ID)
				}
				if label == "" {
					switch {
					case strings.TrimSpace(member.Name) != "":
						label = strings.TrimSpace(member.Name)
					case strings.TrimSpace(member.Email) != "":
						label = strings.TrimSpace(member.Email)
					}
				}

				if value == "" || label == "" {
					continue
				}

				resources = append(resources, core.IntegrationResource{
					Type: ResourceTypeAlertTarget,
					ID:   value,
					Name: "User · " + label,
				})
			}

			return resources, nil

		case alertTargetTypeTeam:
			teams, err := client.ListProjectTeams(projectSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to list project teams: %w", err)
			}

			resources := make([]core.IntegrationResource, 0, len(teams))
			for _, team := range teams {
				if strings.TrimSpace(team.ID) == "" || strings.TrimSpace(team.Name) == "" {
					continue
				}

				resources = append(resources, core.IntegrationResource{
					Type: ResourceTypeAlertTarget,
					ID:   strings.TrimSpace(team.ID),
					Name: "Team · " + strings.TrimSpace(team.Name),
				})
			}

			return resources, nil
		}

		return []core.IntegrationResource{}, nil

	case ResourceTypeRelease:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create sentry client: %w", err)
		}

		projectSlug := strings.TrimSpace(ctx.Parameters["project"])

		releases, err := client.ListReleases()
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(releases))
		for _, release := range releases {
			if projectSlug != "" && !releaseContainsProject(release, projectSlug) {
				continue
			}

			version := strings.TrimSpace(release.Version)
			if version == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeRelease,
				ID:   version,
				Name: version,
			})
		}

		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (s *Sentry) updateMetadata(ctx core.SyncContext, config ...Configuration) error {
	var cfg Configuration
	if len(config) > 0 {
		cfg = config[0]
	} else {
		decoded, err := s.loadConfiguration(ctx.Integration)
		if err != nil {
			return err
		}
		cfg = decoded
	}

	client := NewAPIClient(ctx.HTTP, cfg.BaseURL, cfg.UserToken)
	if inferredSlug := orgSlugFromBaseURL(cfg.BaseURL); inferredSlug != "" {
		client.orgSlug = inferredSlug
		return s.populateMetadataFromOrg(ctx, client)
	}

	organizations, err := client.ListOrganizations()
	if err != nil {
		var sentryAPIError *apiError
		if errors.As(err, &sentryAPIError) && sentryAPIError.StatusCode == http.StatusUnauthorized {
			if _, authErr := client.GetAuthIdentity(); authErr == nil {
				return fmt.Errorf(
					"token is accepted by /auth but is not authorized for organization listing; use your org-specific URL as Sentry URL (e.g., https://your-org.sentry.io)",
				)
			}
		}

		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(organizations) != 1 {
		return fmt.Errorf("expected exactly one organization, got %d", len(organizations))
	}

	client.orgSlug = organizations[0].Slug
	return s.populateMetadataFromOrg(ctx, client)
}

func (s *Sentry) populateMetadataFromOrg(ctx core.SyncContext, client *Client) error {
	organization, err := client.GetOrganization()
	if err != nil {
		return fmt.Errorf("failed to retrieve organization: %w", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	metadata.AppSlug = strings.TrimSpace(metadata.AppSlug)
	metadata.Organization = &OrganizationSummary{
		ID:   organization.ID,
		Slug: organization.Slug,
		Name: organization.Name,
	}
	metadata.Projects = projects
	metadata.Teams = teams

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (s *Sentry) loadConfiguration(integration core.IntegrationContext) (Configuration, error) {
	clientSecret := optionalConfig(integration, "clientSecret")

	// Sentry rotates the client secret when the app is updated via PUT.
	// If a newer secret was captured automatically, prefer it over the stored config.
	if secrets, err := integration.GetSecrets(); err == nil {
		for _, secret := range secrets {
			if secret.Name == "clientSecret" && len(secret.Value) > 0 {
				clientSecret = string(secret.Value)
				break
			}
		}
	}

	return Configuration{
		BaseURL:         normalizeBaseURL(optionalConfig(integration, "baseUrl")),
		IntegrationName: optionalConfig(integration, "integrationName"),
		UserToken:       optionalConfig(integration, "userToken"),
		ClientSecret:    clientSecret,
	}, nil
}

func optionalConfig(integration core.IntegrationContext, name string) string {
	value, err := integration.GetConfig(name)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(value))
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultBaseURL
	}

	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}

	return strings.TrimSuffix(raw, "/")
}

func orgSlugFromBaseURL(baseURL string) string {
	parsed, err := url.Parse(normalizeBaseURL(baseURL))
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "sentry.io" || host == "www.sentry.io" {
		return ""
	}

	if strings.HasSuffix(host, ".sentry.io") {
		slug := strings.TrimSuffix(host, ".sentry.io")
		if slug != "" && !strings.Contains(slug, ".") {
			return slug
		}
	}

	return ""
}

// sentryUIBaseURL returns the base URL for Sentry's web UI.
// Org-specific subdomains (e.g. https://your-org.sentry.io) are for data ingest only;
// settings and developer pages always live on https://sentry.io.
func sentryUIBaseURL(baseURL string) string {
	parsed, err := url.Parse(normalizeBaseURL(baseURL))
	if err != nil {
		return "https://sentry.io"
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "sentry.io" || strings.HasSuffix(host, ".sentry.io") {
		return "https://sentry.io"
	}
	// Self-hosted Sentry instance — use the provided base URL as-is.
	return normalizeBaseURL(baseURL)
}

func developerSettingsURL(baseURL, orgSlug string) string {
	base := sentryUIBaseURL(baseURL)
	if orgSlug != "" {
		return fmt.Sprintf("%s/settings/%s/developer-settings/", base, orgSlug)
	}
	return fmt.Sprintf("%s/settings/", base)
}

func newInternalIntegrationURL(baseURL string) string {
	orgSlug := orgSlugFromBaseURL(baseURL)
	base := sentryUIBaseURL(baseURL)
	if orgSlug != "" {
		return fmt.Sprintf("%s/settings/%s/developer-settings/new-internal/", base, orgSlug)
	}
	return fmt.Sprintf("%s/settings/", base)
}

func eventsURL(ctx core.SyncContext) string {
	baseURL := strings.TrimSuffix(ctx.BaseURL, "/")
	if ctx.WebhooksBaseURL != "" {
		baseURL = strings.TrimSuffix(ctx.WebhooksBaseURL, "/")
	}
	return fmt.Sprintf("%s/api/v1/integrations/%s/events", baseURL, ctx.Integration.ID().String())
}

func verifyWebhookSignature(signature string, body, secret []byte) error {
	signature = strings.TrimSpace(signature)
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return fmt.Errorf("missing signature")
	}
	if len(secret) == 0 {
		return fmt.Errorf("missing webhook secret")
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func displayIntegrationName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "SuperPlane"
	}

	return strings.TrimSpace(name)
}

func normalizedIssueTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	separatorIndex := strings.Index(title, ":")
	if separatorIndex <= 0 || separatorIndex >= len(title)-1 {
		return title
	}

	prefix := strings.TrimSpace(title[:separatorIndex])
	suffix := strings.TrimSpace(title[separatorIndex+1:])
	if prefix == "" || suffix == "" {
		return title
	}

	return suffix
}

func displayIssueLabel(shortID, title string) string {
	normalizedTitle := normalizedIssueTitle(title)
	shortID = strings.TrimSpace(shortID)

	switch {
	case shortID != "" && normalizedTitle != "":
		return fmt.Sprintf("%s · %s", shortID, normalizedTitle)
	case normalizedTitle != "":
		return normalizedTitle
	default:
		return shortID
	}
}

func isExpressionValue(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "{{") || strings.Contains(value, "$[")
}
func alertRuleContainsProject(alertRule MetricAlertRule, projectSlug string) bool {
	projectSlug = strings.TrimSpace(projectSlug)
	if projectSlug == "" {
		return true
	}

	return slices.ContainsFunc(alertRule.Projects, func(project string) bool {
		return strings.TrimSpace(project) == projectSlug
	})
}

func displayAlertRuleLabel(alertRule MetricAlertRule) string {
	name := strings.TrimSpace(alertRule.Name)
	if name == "" {
		return strings.TrimSpace(alertRule.ID)
	}

	if len(alertRule.Projects) == 1 {
		project := strings.TrimSpace(alertRule.Projects[0])
		if project != "" {
			return fmt.Sprintf("%s · %s", name, project)
		}
	}

	return name
}

func releaseContainsProject(release Release, projectSlug string) bool {
	projectSlug = strings.TrimSpace(projectSlug)
	if projectSlug == "" {
		return true
	}

	return slices.ContainsFunc(release.Projects, func(project ReleaseProject) bool {
		return strings.TrimSpace(project.Slug) == projectSlug
	})
}
func missingCredentialsMessage(config Configuration) string {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(config.UserToken) == "" {
		missing = append(missing, "User Token")
	}
	if strings.TrimSpace(config.ClientSecret) == "" {
		missing = append(missing, "Client Secret")
	}

	if len(missing) == 0 {
		return "Sentry credentials are required"
	}

	return fmt.Sprintf("Sentry configuration is incomplete: missing %s", strings.Join(missing, " and "))
}

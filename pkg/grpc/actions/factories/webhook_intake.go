package factories

import (
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	defaultWebhookIntakeTitlePath       = "title"
	defaultWebhookIntakeDescriptionPath = "description"
	defaultWebhookIntakeLinkPath        = "url"
	webhookIntakeTitleFallback          = "Webhook request"
	webhookIntakeMaxPathLength          = 200
)

var (
	webhookIntakePathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	webhookIntakeBodyPathPattern    = regexp.MustCompile(`root\(\)\.data\.body((?:\?\.\["[A-Za-z0-9_-]+"\])+)`)
	webhookIntakeSegmentPattern     = regexp.MustCompile(`\?\.\["([A-Za-z0-9_-]+)"\]`)
)

func defaultWebhookIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.AuthorsWithAccess = false
	settings.WebhookTitlePath = defaultWebhookIntakeTitlePath
	settings.WebhookDescriptionPath = defaultWebhookIntakeDescriptionPath
	settings.WebhookLinkPath = defaultWebhookIntakeLinkPath
	return settings
}

func validateWebhookIntakeSettings(settings intakeSettings) error {
	if settings.WebhookTitlePath == "" {
		return invalidArgument("a title field is required")
	}
	for label, path := range map[string]string{
		"title":       settings.WebhookTitlePath,
		"description": settings.WebhookDescriptionPath,
		"link":        settings.WebhookLinkPath,
	} {
		if !validWebhookIntakePath(path) {
			return invalidArgument(fmt.Sprintf("%s field must be dot-separated names with letters, digits, - or _", label))
		}
	}
	return nil
}

func validWebhookIntakePath(path string) bool {
	if path == "" {
		return true
	}
	if len(path) > webhookIntakeMaxPathLength {
		return false
	}
	for _, segment := range strings.Split(path, ".") {
		if !webhookIntakePathSegmentPattern.MatchString(segment) {
			return false
		}
	}
	return true
}

// webhookIntakeBodyValue reads a body field without failing on a missing
// parent: the request body is whatever the sender posts.
func webhookIntakeBodyValue(path string) string {
	var value strings.Builder
	value.WriteString("root().data.body")
	for _, segment := range strings.Split(path, ".") {
		fmt.Fprintf(&value, `?.[%q]`, segment)
	}
	return value.String()
}

// webhookIntakeTextExpression renders a body field as text. A non-text value
// is written as JSON so an object or a number still reads in the task.
func webhookIntakeTextExpression(path, fallback string) string {
	if path == "" {
		return fallback
	}
	value := webhookIntakeBodyValue(path)
	return fmt.Sprintf(`{{ %s == nil || %s == "" ? %q : (type(%s) == "string" ? %s : toJSON(%s)) }}`,
		value, value, fallback, value, value, value)
}

func webhookIntakeCreateConfiguration(settings intakeSettings) map[string]any {
	return map[string]any{
		"title":       webhookIntakeTextExpression(settings.WebhookTitlePath, webhookIntakeTitleFallback),
		"description": webhookIntakeTextExpression(settings.WebhookDescriptionPath, ""),
		"originUrl":   webhookIntakeTextExpression(settings.WebhookLinkPath, ""),
	}
}

// webhookIntakePathFromExpression reads the body path back out of a generated
// expression. A hand-edited expression without a body path reports "".
func webhookIntakePathFromExpression(expression any) string {
	text, _ := expression.(string)
	match := webhookIntakeBodyPathPattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	segments := webhookIntakeSegmentPattern.FindAllStringSubmatch(match[1], -1)
	path := make([]string, 0, len(segments))
	for _, segment := range segments {
		path = append(path, segment[1])
	}
	return strings.Join(path, ".")
}

func webhookIntakeSettingsFromCreateNode(create *models.Node, settings intakeSettings) intakeSettings {
	if create == nil {
		return settings
	}
	if title := webhookIntakePathFromExpression(create.Configuration["title"]); title != "" {
		settings.WebhookTitlePath = title
	}
	settings.WebhookDescriptionPath = webhookIntakePathFromExpression(create.Configuration["description"])
	settings.WebhookLinkPath = webhookIntakePathFromExpression(create.Configuration["originUrl"])
	return settings
}

func applyWebhookIntakeSettingsToGraph(graph intakeGraph, settings intakeSettings, nodes []models.Node) ([]models.Node, error) {
	if err := validateWebhookIntakeSettings(settings); err != nil {
		return nil, err
	}
	for i := range nodes {
		if nodes[i].ID != graph.CreateNodeID {
			continue
		}
		configuration := maps.Clone(nodes[i].Configuration)
		if configuration == nil {
			configuration = map[string]any{}
		}
		maps.Copy(configuration, webhookIntakeCreateConfiguration(settings))
		nodes[i].Configuration = configuration
		return nodes, nil
	}
	return nil, invalidArgument("intake automation has no task creation node to update")
}

// webhookIntakeRunTitle matches the task title the create node renders for a
// text field. Every request is a run, so a missing field uses the fallback.
func webhookIntakeRunTitle(titlePath string, event models.CanvasEvent) string {
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return ""
	}
	path := append([]string{"body"}, strings.Split(titlePath, ".")...)
	if title := nestedString(payload, path...); title != "" {
		return title
	}
	return webhookIntakeTitleFallback
}

// resolveWebhookIntakeBinding points the trigger at a workspace webhook. A
// webhook has no integration, so the binding carries only the trigger field.
func resolveWebhookIntakeBinding(tx *gorm.DB, factory *models.Factory, webhookID string) (*intakeBinding, error) {
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		return nil, invalidArgument("a webhook is required")
	}
	id, err := uuid.Parse(webhookID)
	if err != nil {
		return nil, invalidArgument("webhook is invalid")
	}
	if _, err := models.FindFactoryWebhook(tx, factory.OrganizationID, factory.ID, id); err != nil {
		return nil, invalidArgument("webhook was not found")
	}

	return &intakeBinding{Configuration: map[string]any{"webhook": id.String()}}, nil
}

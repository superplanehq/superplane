package factories

import (
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

type prFeedbackSettings struct {
	Repository             string
	Mention                string
	IgnoreBots             bool
	AllowedBots            []string
	CheckNames             []string
	MaximumAttempts        int
	RunnerIntegrationIDs   []string
	RunnerIntegrationNames []string
}

func defaultPRFeedbackSettings() prFeedbackSettings {
	return prFeedbackSettings{
		Mention:         prFeedbackDefaultMention,
		IgnoreBots:      true,
		MaximumAttempts: prFeedbackDefaultMaximumAttempts,
	}
}

func (s prFeedbackSettings) normalized() prFeedbackSettings {
	s.Repository = strings.TrimSpace(s.Repository)
	s.Mention = strings.TrimSpace(s.Mention)
	if s.Mention == "" {
		s.Mention = prFeedbackDefaultMention
	}
	s.AllowedBots = normalizedAllowedBots(s.AllowedBots)
	s.CheckNames = normalizedCheckNames(s.CheckNames)
	s.RunnerIntegrationIDs = normalizedUniqueStrings(s.RunnerIntegrationIDs)
	s.RunnerIntegrationNames = normalizedUniqueStrings(s.RunnerIntegrationNames)
	return s
}

func normalizedCheckNames(names []string) []string {
	return normalizedUniqueStrings(names)
}

func normalizedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

// normalizedAllowedBots trims entries, strips a leading @, drops empties, and
// removes duplicates while preserving order.
func normalizedAllowedBots(bots []string) []string {
	seen := make(map[string]struct{}, len(bots))
	normalized := make([]string, 0, len(bots))
	for _, bot := range bots {
		bot = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(bot), "@"))
		if bot == "" {
			continue
		}
		key := strings.ToLower(bot)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, bot)
	}
	return normalized
}

// allowedBotsNodeValue converts a string slice into the []any shape that
// node configuration validation expects for list fields.
func allowedBotsNodeValue(bots []string) []any {
	values := make([]any, 0, len(bots))
	for _, bot := range bots {
		values = append(values, bot)
	}
	return values
}

func prFeedbackSettingsFromGraph(graph prFeedbackGraph, spec models.LiveCanvasSpec) prFeedbackSettings {
	settings := defaultPRFeedbackSettings()
	for _, nodeID := range graph.triggerNodeIDs() {
		node := findIntakeNode(spec.Nodes, nodeID)
		if node == nil {
			continue
		}
		if repository := strings.TrimSpace(prFeedbackNodeString(node, "repository")); repository != "" {
			settings.Repository = repository
		}
		if mention := strings.TrimSpace(prFeedbackNodeString(node, "contentFilter")); mention != "" {
			settings.Mention = mention
		}
		settings.IgnoreBots = prFeedbackNodeBool(node, "ignoreBots", settings.IgnoreBots)
		settings.AllowedBots = prFeedbackNodeStringSlice(node, "allowedBots")
		break
	}

	if graph.isChecks() {
		wait := findIntakeNode(spec.Nodes, graph.WaitChecksNodeID)
		settings.CheckNames = prFeedbackNodeStringSlice(wait, "checkNames")
		if settings.Repository == "" {
			settings.Repository = strings.TrimSpace(prFeedbackNodeString(wait, "repository"))
		}
		settings.RunnerIntegrationNames = runnerExtraIntegrationNames(findIntakeNode(spec.Nodes, graph.RunnerNodeID))
	}

	return settings.normalized()
}

func runnerExtraIntegrationNames(node *models.Node) []string {
	if node == nil {
		return nil
	}
	entries, ok := node.Configuration["environmentFrom"].([]any)
	if !ok {
		return nil
	}

	var names []string
	for i, entry := range entries {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		integration, ok := item["integration"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := integration["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if i == 0 {
			continue
		}
		names = append(names, name)
	}
	return names
}

func serializePRFeedbackSettings(settings prFeedbackSettings) *pb.FactoryPRFeedbackHandler_Settings {
	settings = settings.normalized()
	maximumAttempts := int32(settings.MaximumAttempts)
	return &pb.FactoryPRFeedbackHandler_Settings{
		Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
			Repository: settings.Repository,
		},
		Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
			Mention:     settings.Mention,
			IgnoreBots:  settings.IgnoreBots,
			AllowedBots: settings.AllowedBots,
		},
		Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{
			Names:                settings.CheckNames,
			MaximumAttempts:      &maximumAttempts,
			RunnerIntegrationIds: settings.RunnerIntegrationIDs,
		},
	}
}

func validatePRFeedbackSettingsForSource(
	tx *gorm.DB,
	orgID uuid.UUID,
	source string,
	settings prFeedbackSettings,
	requested *pb.FactoryPRFeedbackHandler_Settings,
) error {
	if source == models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
		if requested != nil && requested.GetChecks() != nil && len(requested.GetChecks().GetRunnerIntegrationIds()) > 0 {
			return invalidArgument("discussion handlers do not accept runner integrations")
		}
		if strings.TrimSpace(settings.Mention) == "" {
			return invalidArgument("mention cannot be empty")
		}
		return nil
	}

	if settings.MaximumAttempts < prFeedbackMaximumAttemptsMin || settings.MaximumAttempts > prFeedbackMaximumAttemptsMax {
		return invalidArgument("maximum attempts must be between 1 and 10")
	}
	for _, name := range settings.CheckNames {
		if strings.TrimSpace(name) == "" {
			return invalidArgument("check names must be non-empty")
		}
	}
	return validateRunnerIntegrationIDs(tx, orgID, settings.RunnerIntegrationIDs)
}

func validateRunnerIntegrationIDs(tx *gorm.DB, orgID uuid.UUID, integrationIDs []string) error {
	for _, raw := range integrationIDs {
		integrationID, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return invalidArgument("runner integration id is not valid")
		}
		if _, err := models.FindIntegrationInTransaction(tx, orgID, integrationID); err != nil {
			return invalidArgument("runner integration must belong to the organization")
		}
	}
	return nil
}

func resolveRunnerIntegrationNames(tx *gorm.DB, orgID uuid.UUID, settings *prFeedbackSettings) error {
	names := make([]string, 0, len(settings.RunnerIntegrationIDs))
	for _, raw := range settings.RunnerIntegrationIDs {
		integrationID, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return invalidArgument("runner integration id is not valid")
		}
		integration, err := models.FindIntegrationInTransaction(tx, orgID, integrationID)
		if err != nil {
			return invalidArgument("runner integration must belong to the organization")
		}
		names = append(names, integration.InstallationName)
	}
	settings.RunnerIntegrationNames = names
	return nil
}

func resolveRunnerIntegrationIDs(tx *gorm.DB, orgID uuid.UUID, settings *prFeedbackSettings) error {
	ids := make([]string, 0, len(settings.RunnerIntegrationNames))
	for _, name := range settings.RunnerIntegrationNames {
		integration, err := models.FindIntegrationByName(tx, orgID, name)
		if err != nil {
			continue
		}
		ids = append(ids, integration.ID.String())
	}
	settings.RunnerIntegrationIDs = ids
	return nil
}

func parsePRFeedbackSettings(current prFeedbackSettings, requested *pb.FactoryPRFeedbackHandler_Settings) prFeedbackSettings {
	if requested == nil {
		return current
	}

	updated := current
	if requested.GetSubject() != nil {
		updated.Repository = strings.TrimSpace(requested.GetSubject().GetRepository())
	}
	if requested.GetDiscussion() != nil {
		updated.Mention = strings.TrimSpace(requested.GetDiscussion().GetMention())
		updated.IgnoreBots = requested.GetDiscussion().GetIgnoreBots()
		updated.AllowedBots = requested.GetDiscussion().GetAllowedBots()
	}
	if requested.GetChecks() != nil {
		updated.CheckNames = requested.GetChecks().GetNames()
		if requested.GetChecks().MaximumAttempts != nil {
			updated.MaximumAttempts = int(requested.GetChecks().GetMaximumAttempts())
		}
		updated.RunnerIntegrationIDs = requested.GetChecks().GetRunnerIntegrationIds()
	}
	return updated.normalized()
}

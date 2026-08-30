package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

type prFeedbackSettings struct {
	Repository  string
	Mention     string
	IgnoreBots  bool
	AllowedBots []string
}

func defaultPRFeedbackSettings() prFeedbackSettings {
	return prFeedbackSettings{
		Mention:    prFeedbackDefaultMention,
		IgnoreBots: true,
	}
}

func (s prFeedbackSettings) normalized() prFeedbackSettings {
	s.Repository = strings.TrimSpace(s.Repository)
	s.Mention = strings.TrimSpace(s.Mention)
	if s.Mention == "" {
		s.Mention = prFeedbackDefaultMention
	}
	s.AllowedBots = normalizedAllowedBots(s.AllowedBots)
	return s
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
	return settings.normalized()
}

func serializePRFeedbackSettings(settings prFeedbackSettings) *pb.FactoryPRFeedbackHandler_Settings {
	settings = settings.normalized()
	return &pb.FactoryPRFeedbackHandler_Settings{
		Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
			Repository: settings.Repository,
		},
		Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
			Mention:     settings.Mention,
			IgnoreBots:  settings.IgnoreBots,
			AllowedBots: settings.AllowedBots,
		},
	}
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
	return updated.normalized()
}

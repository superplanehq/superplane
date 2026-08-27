package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

type prFeedbackSettings struct {
	Repository string
	Mention    string
	IgnoreBots bool
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
	return s
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
			Mention:    settings.Mention,
			IgnoreBots: settings.IgnoreBots,
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
	}
	return updated.normalized()
}

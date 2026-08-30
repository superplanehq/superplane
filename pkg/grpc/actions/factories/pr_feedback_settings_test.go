package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func Test__PRFeedbackSettings__Normalized(t *testing.T) {
	t.Run("trims, drops leading @, and de-duplicates allowed bots", func(t *testing.T) {
		settings := prFeedbackSettings{
			Mention:     prFeedbackDefaultMention,
			AllowedBots: []string{" @CodeRabbitAI ", "coderabbitai", "bugbot", "", "  "},
		}

		normalized := settings.normalized()

		assert.Equal(t, []string{"CodeRabbitAI", "bugbot"}, normalized.AllowedBots)
	})

	t.Run("nil allowed bots stays empty", func(t *testing.T) {
		settings := prFeedbackSettings{Mention: prFeedbackDefaultMention}

		normalized := settings.normalized()

		assert.Empty(t, normalized.AllowedBots)
	})
}

func Test__PRFeedbackSettingsFromGraph__ReadsAllowedBots(t *testing.T) {
	spec := prFeedbackSpecFromTemplate(t, "acme/app")
	graph := resolvePRFeedbackGraph(spec)

	for i := range spec.Nodes {
		if spec.Nodes[i].ID == graph.CommentTriggerNodeID {
			spec.Nodes[i].Configuration["allowedBots"] = []any{"coderabbitai", "bugbot"}
		}
	}

	settings := prFeedbackSettingsFromGraph(graph, spec)
	assert.Equal(t, []string{"coderabbitai", "bugbot"}, settings.AllowedBots)
}

func Test__PRFeedbackSettings__SerializeAndParseAllowedBots(t *testing.T) {
	settings := prFeedbackSettings{
		Repository:  "acme/app",
		Mention:     prFeedbackDefaultMention,
		AllowedBots: []string{"coderabbitai", "bugbot"},
	}

	serialized := serializePRFeedbackSettings(settings)
	assert.Equal(t, []string{"coderabbitai", "bugbot"}, serialized.GetDiscussion().GetAllowedBots())

	parsed := parsePRFeedbackSettings(defaultPRFeedbackSettings(), &pb.FactoryPRFeedbackHandler_Settings{
		Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{Repository: "acme/app"},
		Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
			Mention:     prFeedbackDefaultMention,
			AllowedBots: []string{"coderabbitai", "bugbot"},
		},
	})
	assert.Equal(t, []string{"coderabbitai", "bugbot"}, parsed.AllowedBots)
}

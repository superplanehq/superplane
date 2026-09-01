package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
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

func Test__PRFeedbackSettings__Checks(t *testing.T) {
	t.Run("trims and de-duplicates check names", func(t *testing.T) {
		settings := prFeedbackSettings{
			CheckNames:      []string{" lint ", "lint", "unit", "", "  "},
			MaximumAttempts: 3,
		}

		normalized := settings.normalized()

		assert.Equal(t, []string{"lint", "unit"}, normalized.CheckNames)
	})

	t.Run("serializes check names and attempt limit", func(t *testing.T) {
		settings := prFeedbackSettings{
			Repository:      "acme/app",
			CheckNames:      []string{"lint", "unit"},
			MaximumAttempts: 4,
		}

		serialized := serializePRFeedbackSettings(settings)
		assert.Equal(t, []string{"lint", "unit"}, serialized.GetChecks().GetNames())
		assert.Equal(t, int32(4), serialized.GetChecks().GetMaximumAttempts())
	})

	t.Run("rejects attempt values outside 1 through 10", func(t *testing.T) {
		zero := int32(0)
		parsed := parsePRFeedbackSettings(defaultPRFeedbackSettings(), &pb.FactoryPRFeedbackHandler_Settings{
			Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{MaximumAttempts: &zero},
		})
		err := validatePRFeedbackSettingsForSource(
			nil,
			uuid.Nil,
			models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
			parsed,
			&pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{MaximumAttempts: &zero},
			},
		)
		require.Error(t, err)

		tooHigh := int32(11)
		parsed = parsePRFeedbackSettings(defaultPRFeedbackSettings(), &pb.FactoryPRFeedbackHandler_Settings{
			Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{MaximumAttempts: &tooHigh},
		})
		err = validatePRFeedbackSettingsForSource(
			nil,
			uuid.Nil,
			models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
			parsed,
			&pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{MaximumAttempts: &tooHigh},
			},
		)
		require.Error(t, err)
	})

	t.Run("reads check names from a generated graph", func(t *testing.T) {
		spec := prFeedbackChecksSpecFromTemplate(t, "acme/app")
		graph := resolvePRFeedbackGraph(spec)
		for i := range spec.Nodes {
			if spec.Nodes[i].ID == graph.WaitChecksNodeID {
				spec.Nodes[i].Configuration["checkNames"] = []any{"lint", "unit"}
			}
		}

		settings := prFeedbackSettingsFromGraph(graph, spec)
		assert.Equal(t, []string{"lint", "unit"}, settings.CheckNames)
		assert.Equal(t, "acme/app", settings.Repository)
	})
}

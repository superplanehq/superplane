package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__AgentModelsForSource__UsesTheKeyModelList(t *testing.T) {
	t.Parallel()

	model, planning := agentModelsForSource(modelSourceAnthropic, []string{
		"claude-haiku-4-5",
		"claude-opus-4-6",
		"claude-sonnet-4-6",
	})
	assert.Equal(t, "claude-sonnet-4-6", model)
	assert.Equal(t, "claude-opus-4-6", planning)

	model, planning = agentModelsForSource(modelSourceAnthropic, []string{
		"claude-opus-4-6",
		"claude-opus-5-5",
		"claude-sonnet-4-6",
	})
	assert.Equal(t, "claude-opus-5-5", model)
	assert.Equal(t, "claude-opus-5-5", planning)

	model, planning = agentModelsForSource(modelSourceAnthropic, []string{
		"claude-opus-5-5",
		"claude-sonnet-4-6",
	})
	assert.Equal(t, "claude-opus-5-5", model)
	assert.Equal(t, "claude-opus-5-5", planning)

	model, planning = agentModelsForSource(modelSourceOpenAI, []string{"gpt-4.1", "gpt-5", "o3"})
	assert.Equal(t, "gpt-5", model)
	assert.Equal(t, "gpt-5", planning)

	model, planning = agentModelsForSource(modelSourceOpenRouter, []string{
		"openai/gpt-4.1",
		"anthropic/claude-sonnet-4-6",
	})
	assert.Equal(t, "anthropic/claude-sonnet-4-6", model)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", planning)

	model, planning = agentModelsForSource(modelSourceOpenRouter, []string{
		"anthropic/claude-opus-4-6",
		"anthropic/claude-sonnet-4-6",
	})
	assert.Equal(t, "anthropic/claude-sonnet-4-6", model)
	assert.Equal(t, "anthropic/claude-opus-4-6", planning)

	model, planning = agentModelsForSource(modelSourceOpenRouter, []string{
		"anthropic/claude-sonnet-4-6",
		"x-ai/grok-4.7",
	})
	assert.Equal(t, "x-ai/grok-4.7", model)
	assert.Equal(t, "x-ai/grok-4.7", planning)
}

func Test__AgentModelsForSource__UsesVersionedIdsWhenTheKeyReturnsNoModels(t *testing.T) {
	t.Parallel()

	model, planning := agentModelsForSource(modelSourceAnthropic, nil)
	assert.Equal(t, "claude-opus-5-5", model)
	assert.Equal(t, "claude-opus-5-5", planning)

	model, planning = agentModelsForSource(modelSourceOpenRouter, []string{" "})
	assert.Equal(t, "anthropic/claude-sonnet-4-6", model)
	assert.Equal(t, "anthropic/claude-opus-5-5", planning)
}

func Test__CompareModelIDs__OrdersNumericRuns(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"claude-opus-4-6", "claude-opus-10"}, uniqueSortedModelIDs([]string{
		"claude-opus-10",
		"claude-opus-4-6",
	}))
}

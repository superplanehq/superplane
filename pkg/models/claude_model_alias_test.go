package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestConcreteClaudeModelID(t *testing.T) {
	candidates := []string{"claude-haiku-4-5", "claude-opus-4-6", "claude-opus-5-5", "claude-sonnet-4-6", "claude-sonnet-5"}

	assert.Equal(t, "claude-sonnet-5", models.ConcreteClaudeModelID("sonnet", candidates))
	assert.Equal(t, "claude-opus-5-5", models.ConcreteClaudeModelID("opus", candidates))
	assert.Equal(t, "anthropic/claude-opus-5-5", models.ConcreteClaudeModelID("anthropic/opus", candidates))
	assert.Equal(t, "claude-sonnet-4-6", models.ConcreteClaudeModelID("claude-sonnet-4-6", candidates))
	assert.Equal(t, "claude-sonnet-4-6", models.ConcreteClaudeModelID("sonnet", nil))
	assert.Equal(t, "claude-opus-5-5", models.ConcreteClaudeModelID("opus", []string{"sonnet"}))
	assert.Equal(t, "gpt-5", models.ConcreteClaudeModelID("gpt-5", nil))
	assert.False(t, models.IsClaudeFamilyAlias("claude-opus-4-6"))
	assert.True(t, models.IsClaudeFamilyAlias("anthropic/sonnet"))
}

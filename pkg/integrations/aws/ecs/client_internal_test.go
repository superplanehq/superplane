package ecs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__isNetworkConfigurationTemplate(t *testing.T) {
	t.Run("template after ToMap omits empty arrays", func(t *testing.T) {
		isTemplate := isNetworkConfigurationTemplate(map[string]any{
			"awsvpcConfiguration": map[string]any{
				"assignPublicIp": "DISABLED",
			},
		})

		assert.True(t, isTemplate)
	})

	t.Run("template with explicit empty arrays", func(t *testing.T) {
		isTemplate := isNetworkConfigurationTemplate(map[string]any{
			"awsvpcConfiguration": map[string]any{
				"assignPublicIp": "DISABLED",
				"subnets":        []string{},
				"securityGroups": []string{},
			},
		})

		assert.True(t, isTemplate)
	})

	t.Run("non-template with populated subnet list", func(t *testing.T) {
		isTemplate := isNetworkConfigurationTemplate(map[string]any{
			"awsvpcConfiguration": map[string]any{
				"assignPublicIp": "DISABLED",
				"subnets":        []string{"subnet-123"},
			},
		})

		assert.False(t, isTemplate)
	})

	t.Run("non-template with extra fields", func(t *testing.T) {
		isTemplate := isNetworkConfigurationTemplate(map[string]any{
			"awsvpcConfiguration": map[string]any{
				"assignPublicIp": "DISABLED",
				"foo":            "bar",
			},
		})

		assert.False(t, isTemplate)
	})
}

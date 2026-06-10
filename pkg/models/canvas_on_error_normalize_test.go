package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__FindOnErrorNodeID__ReturnsMarkedNode(t *testing.T) {
	nodes := []models.Node{
		{ID: "trigger", Type: models.NodeTypeTrigger, OnError: true},
		{ID: "handler", Type: models.NodeTypeComponent, OnError: true},
		{ID: "other", Type: models.NodeTypeComponent},
	}

	assert.Equal(t, "handler", models.FindOnErrorNodeID(nodes))
}

func Test__NormalizeOnErrorNodes__KeepsSingleHandler(t *testing.T) {
	nodes := []models.Node{
		{ID: "first", Type: models.NodeTypeComponent, OnError: true},
		{ID: "second", Type: models.NodeTypeComponent, OnError: true},
		{ID: "trigger", Type: models.NodeTypeTrigger, OnError: true},
	}

	normalized := models.NormalizeOnErrorNodes(nodes)

	assert.True(t, normalized[0].OnError)
	assert.False(t, normalized[1].OnError)
	assert.False(t, normalized[2].OnError)
}

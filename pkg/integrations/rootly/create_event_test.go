package rootly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEvent_Name(t *testing.T) {
	component := &CreateEvent{}
	assert.Equal(t, "rootly.createEvent", component.Name())
}

func TestCreateEvent_Label(t *testing.T) {
	component := &CreateEvent{}
	assert.Equal(t, "Create Event", component.Label())
}

func TestCreateEvent_Description(t *testing.T) {
	component := &CreateEvent{}
	assert.Equal(t, "Add a timeline event to a Rootly incident", component.Description())
}

func TestCreateEvent_Configuration(t *testing.T) {
	component := &CreateEvent{}
	config := component.Configuration()

	assert.Len(t, config, 3)

	// Incident ID field
	assert.Equal(t, "incidentId", config[0].Name)
	assert.True(t, config[0].Required)

	// Body field
	assert.Equal(t, "body", config[1].Name)
	assert.True(t, config[1].Required)

	// Visibility field
	assert.Equal(t, "visibility", config[2].Name)
	assert.False(t, config[2].Required)
	assert.Equal(t, "internal", config[2].Default)
}

func TestCreateEvent_OutputChannels(t *testing.T) {
	component := &CreateEvent{}
	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestCreateEvent_ExampleOutput(t *testing.T) {
	component := &CreateEvent{}
	example := component.ExampleOutput()

	assert.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "body")
	assert.Contains(t, example, "visibility")
	assert.Contains(t, example, "occurred_at")
	assert.Contains(t, example, "created_at")
}

func TestCreateEvent_Icon(t *testing.T) {
	component := &CreateEvent{}
	assert.Equal(t, "message-square", component.Icon())
}

func TestCreateEvent_Color(t *testing.T) {
	component := &CreateEvent{}
	assert.Equal(t, "gray", component.Color())
}

func TestCreateEvent_Actions(t *testing.T) {
	component := &CreateEvent{}
	actions := component.Actions()
	assert.Empty(t, actions)
}

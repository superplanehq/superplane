package pubsub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PublishMessageConfiguration(t *testing.T) {
	c := &PublishMessage{}
	fields := c.Configuration()
	require.NotEmpty(t, fields)

	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	assert.Contains(t, names, "topic")
	assert.Contains(t, names, "format")
	assert.Contains(t, names, "json")
	assert.Contains(t, names, "text")
}

func Test_PublishMessageName(t *testing.T) {
	c := &PublishMessage{}
	assert.Equal(t, "gcp.pubsub.publishMessage", c.Name())
}

func Test_PublishMessageLabel(t *testing.T) {
	c := &PublishMessage{}
	assert.Equal(t, "Pub/Sub • Publish Message", c.Label())
}

func Test_PublishMessageOutputChannels(t *testing.T) {
	c := &PublishMessage{}
	channels := c.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test_PublishMessageBuildMessageData(t *testing.T) {
	c := &PublishMessage{}

	t.Run("text format", func(t *testing.T) {
		text := "hello world"
		data, err := c.buildMessageData(PublishMessageConfiguration{
			Format: PublishFormatText,
			Text:   &text,
		})
		require.NoError(t, err)
		assert.Equal(t, "hello world", data)
	})

	t.Run("json format", func(t *testing.T) {
		jsonData := any(map[string]any{"key": "value"})
		data, err := c.buildMessageData(PublishMessageConfiguration{
			Format: PublishFormatJSON,
			JSON:   &jsonData,
		})
		require.NoError(t, err)
		assert.JSONEq(t, `{"key":"value"}`, data)
	})

	t.Run("text format without text returns error", func(t *testing.T) {
		_, err := c.buildMessageData(PublishMessageConfiguration{
			Format: PublishFormatText,
			Text:   nil,
		})
		require.Error(t, err)
	})

	t.Run("json format without json returns error", func(t *testing.T) {
		_, err := c.buildMessageData(PublishMessageConfiguration{
			Format: PublishFormatJSON,
			JSON:   nil,
		})
		require.Error(t, err)
	})
}

func Test_PublishMessageExampleOutput(t *testing.T) {
	c := &PublishMessage{}
	example := c.ExampleOutput()
	require.NotNil(t, example)
	assert.NotEmpty(t, example["messageId"])
	assert.NotEmpty(t, example["topic"])
}

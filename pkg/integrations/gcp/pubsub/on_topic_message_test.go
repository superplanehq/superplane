package pubsub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_OnTopicMessageConfiguration(t *testing.T) {
	trigger := &OnTopicMessage{}
	fields := trigger.Configuration()
	require.NotEmpty(t, fields)

	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	assert.Contains(t, names, "topic")
}

func Test_OnTopicMessageName(t *testing.T) {
	trigger := &OnTopicMessage{}
	assert.Equal(t, "gcp.pubsub.onTopicMessage", trigger.Name())
}

func Test_OnTopicMessageLabel(t *testing.T) {
	trigger := &OnTopicMessage{}
	assert.Equal(t, "Pub/Sub • On Topic Message", trigger.Label())
}

func Test_OnTopicMessageExampleData(t *testing.T) {
	trigger := &OnTopicMessage{}
	example := trigger.ExampleData()
	require.NotNil(t, example)
	assert.NotEmpty(t, example["topic"])
	assert.NotEmpty(t, example["messageId"])
}

func Test_TopicSubscriptionPattern(t *testing.T) {
	pattern := TopicSubscriptionPattern("my-topic")
	assert.Equal(t, "pubsub.topic", pattern["type"])
	assert.Equal(t, "my-topic", pattern["topic"])
}

func Test_SanitizeID(t *testing.T) {
	t.Run("keeps alphanumeric and dashes", func(t *testing.T) {
		result := sanitizeID("abc-123-def")
		assert.Equal(t, "abc-123-def", result)
	})

	t.Run("lowercases input", func(t *testing.T) {
		result := sanitizeID("ABC-DEF")
		assert.Equal(t, "abc-def", result)
	})

	t.Run("removes special characters", func(t *testing.T) {
		result := sanitizeID("abc!@#$%^&*()def")
		assert.Equal(t, "abcdef", result)
	})

	t.Run("truncates to 60 characters", func(t *testing.T) {
		long := "a"
		for i := 0; i < 70; i++ {
			long += "b"
		}
		result := sanitizeID(long)
		assert.Len(t, result, 60)
	})
}

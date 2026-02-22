package sqs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__QueueNameFromURL(t *testing.T) {

	t.Run("valid URL -> returns queue name", func(t *testing.T) {
		name := queueNameFromURL("https://sqs.us-east-1.amazonaws.com/123456789012/my-queue")
		assert.Equal(t, "my-queue", name)
	})

	t.Run("invalid URL -> returns trimmed input", func(t *testing.T) {
		name := queueNameFromURL(" not-a-url ")
		assert.Equal(t, "not-a-url", name)
	})
}

package runner

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBrokerTaskIDFromExecutionMetadata(t *testing.T) {
	assert.Equal(t, "tb-1", BrokerTaskIDFromExecutionMetadata(map[string]any{
		ExecutionMetadataBrokerTaskID: "tb-1",
	}))
	assert.Equal(t, "99", BrokerTaskIDFromExecutionMetadata(map[string]any{
		ExecutionMetadataBrokerTaskID: 99,
	}))
	assert.Equal(t, "", BrokerTaskIDFromExecutionMetadata(map[string]any{}))
}

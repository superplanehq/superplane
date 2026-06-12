package ec2

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ManageElasticIP__Setup(t *testing.T) {
	component := &ManageElasticIP{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       " ",
				"operation":    "associate",
				"allocationId": "eipalloc-abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("invalid operation -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"operation": "detach",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("missing allocation ID when associating -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"operation": "associate",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "allocation ID is required")
	})

	t.Run("missing instance when associating -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"operation":    "associate",
				"allocationId": "eipalloc-abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("missing association ID when disassociating -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"operation": "disassociate",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "association ID is required")
	})

	t.Run("valid associate configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"operation":    "associate",
				"allocationId": "eipalloc-abc123",
				"instance":     "i-abc123",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(ManageElasticIPNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "associate", stored.Operation)
	})
}

func Test__ManageElasticIP__Execute_Associate_MissingInstance(t *testing.T) {
	component := &ManageElasticIP{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":       "us-east-1",
			"operation":    "associate",
			"allocationId": "eipalloc-abc123",
		},
		Metadata: &contexts.MetadataContext{},
	})
	require.ErrorContains(t, err, "instance ID is required")
}

func Test__ManageElasticIP__Execute_Associate(t *testing.T) {
	component := &ManageElasticIP{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{okResponse(associateAddressXML("eipassoc-xyz789"))},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":       "us-east-1",
			"operation":    "associate",
			"allocationId": "eipalloc-abc123",
			"instance":     "i-abc123",
		},
		HTTP:           httpContext,
		ExecutionState: executionState,
		Integration:    elasticIPIntegration(),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageElasticIPAssociatePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "eipassoc-xyz789", data["associationId"])
	assert.Equal(t, "eipalloc-abc123", data["allocationId"])
	assert.Equal(t, "i-abc123", data["instanceId"])
	assert.Equal(t, "us-east-1", data["region"])

	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=AssociateAddress")
	assert.Contains(t, string(body), "AllocationId=eipalloc-abc123")
	assert.Contains(t, string(body), "InstanceId=i-abc123")
	assert.Contains(t, string(body), "AllowReassociation=true")
}

func Test__ManageElasticIP__Execute_Disassociate(t *testing.T) {
	component := &ManageElasticIP{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{okResponse(disassociateAddressXML())},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":        "us-east-1",
			"operation":     "disassociate",
			"associationId": "eipassoc-xyz789",
		},
		HTTP:           httpContext,
		ExecutionState: executionState,
		Integration:    elasticIPIntegration(),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageElasticIPDisassociatePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "eipassoc-xyz789", data["associationId"])
	assert.Equal(t, "us-east-1", data["region"])

	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Action=DisassociateAddress")
	assert.Contains(t, string(body), "AssociationId=eipassoc-xyz789")
}

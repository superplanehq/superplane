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

func Test__AllocateElasticIP__Setup(t *testing.T) {
	component := &AllocateElasticIP{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("invalid IP source -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"ipSource": "custom",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid IP source")
	})

	t.Run("BYOIP without pool -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"ipSource": "byoip",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "public IPv4 pool is required")
	})

	t.Run("valid configuration -> stores region and IP source in metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"ipSource": "amazon",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(AllocateElasticIPNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "amazon", stored.IPSource)
	})
}

func Test__AllocateElasticIP__Execute(t *testing.T) {
	component := &AllocateElasticIP{}

	t.Run("allocate address -> emits allocation details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(allocateAddressXML("eipalloc-abc123", "203.0.113.10")),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AllocateElasticIPPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "eipalloc-abc123", data["allocationId"])
		assert.Equal(t, "203.0.113.10", data["publicIp"])
		assert.Equal(t, "vpc", data["domain"])
		assert.Equal(t, "us-east-1", data["region"])

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Action=AllocateAddress")
		assert.Contains(t, string(body), "Domain=vpc")
	})

	t.Run("allocate from BYOIP pool -> sends PublicIpv4Pool", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(allocateAddressXML("eipalloc-byoip", "198.51.100.42")),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"ipSource":       "byoip",
				"publicIpv4Pool": "ipv4pool-ec2-abc123",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "PublicIpv4Pool=ipv4pool-ec2-abc123")
	})

	t.Run("tags are sent as TagSpecification params", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(allocateAddressXML("eipalloc-abc123", "203.0.113.10")),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"tags": []any{
					map[string]any{"key": "env", "value": "prod"},
					map[string]any{"key": "owner", "value": "team-a"},
				},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "TagSpecification.1.ResourceType=elastic-ip")
		assert.Contains(t, string(body), "TagSpecification.1.Tag.1.Key=env")
		assert.Contains(t, string(body), "TagSpecification.1.Tag.1.Value=prod")
		assert.Contains(t, string(body), "TagSpecification.1.Tag.2.Key=owner")
		assert.Contains(t, string(body), "TagSpecification.1.Tag.2.Value=team-a")
	})

	t.Run("Amazon pool ignores stale address in configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(allocateAddressXML("eipalloc-abc123", "203.0.113.10")),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"ipSource": "amazon",
				"address":  "18.97.0.41",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.NotContains(t, string(body), "Address=")
	})

	t.Run("allocate from IPAM pool with address -> sends IpamPoolId and Address", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(allocateAddressXML("eipalloc-ipam", "18.97.0.41")),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"ipSource":   "ipam",
				"ipamPoolId": "ipam-pool-abc123",
				"address":    "18.97.0.41",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    elasticIPIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "IpamPoolId=ipam-pool-abc123")
		assert.Contains(t, string(body), "Address=18.97.0.41")
	})
}

func allocateAddressXML(allocationID, publicIP string) string {
	return `
		<AllocateAddressResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-allocate</requestId>
			<publicIp>` + publicIP + `</publicIp>
			<domain>vpc</domain>
			<allocationId>` + allocationID + `</allocationId>
		</AllocateAddressResponse>`
}

func elasticIPIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func releaseAddressXML() string {
	return `<ReleaseAddressResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><requestId>req-release</requestId><return>true</return></ReleaseAddressResponse>`
}

func associateAddressXML(associationID string) string {
	return `
		<AssociateAddressResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-associate</requestId>
			<associationId>` + associationID + `</associationId>
		</AssociateAddressResponse>`
}

func disassociateAddressXML() string {
	return `<DisassociateAddressResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><requestId>req-disassociate</requestId><return>true</return></DisassociateAddressResponse>`
}

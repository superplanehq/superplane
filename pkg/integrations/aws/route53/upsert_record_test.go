package route53

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestUpsertRecord_Setup(t *testing.T) {
	component := &UpsertRecord{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing hosted zone -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"recordName": "example.com",
				"recordType": "A",
				"ttl":        300,
				"values":     []string{"1.2.3.4"},
			},
		})

		require.ErrorContains(t, err, "hosted zone is required")
	})

	t.Run("missing record name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
		})

		require.ErrorContains(t, err, "record name is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "api.example.com",
				"recordType":   "CNAME",
				"ttl":          60,
				"values":       []string{"lb.example.com"},
			},
		})

		require.NoError(t, err)
	})
}

func TestUpsertRecord_Execute(t *testing.T) {
	component := &UpsertRecord{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "example.com",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{},
			},
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "credentials")
	})

	t.Run("status PENDING -> schedules poll and does not emit", func(t *testing.T) {
		xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C9876543210</Id>
    <Status>PENDING</Status>
    <SubmittedAt>2026-02-13T14:00:00.000Z</SubmittedAt>
  </ChangeInfo>
</ChangeResourceRecordSetsResponse>`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(xmlResponse)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "api.example.com",
				"recordType":   "CNAME",
				"ttl":          60,
				"values":       []string{"lb.example.com"},
			},
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 0, "should not emit until INSYNC")
		assert.Equal(t, pollChangeActionName, requests.Action)
		assert.Equal(t, pollInterval, requests.Duration)
		stored, ok := metadata.Metadata.(RecordChangePollMetadata)
		require.True(t, ok)
		assert.Equal(t, "/change/C9876543210", stored.ChangeID)
		assert.Equal(t, "api.example.com", stored.RecordName)
		assert.Equal(t, "CNAME", stored.RecordType)
	})

	t.Run("status INSYNC -> emits record change immediately", func(t *testing.T) {
		xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C9876543210</Id>
    <Status>INSYNC</Status>
    <SubmittedAt>2026-02-13T14:00:00.000Z</SubmittedAt>
  </ChangeInfo>
</ChangeResourceRecordSetsResponse>`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(xmlResponse)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "api.example.com",
				"recordType":   "CNAME",
				"ttl":          60,
				"values":       []string{"lb.example.com"},
			},
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		require.True(t, execState.Passed)
		require.Equal(t, "aws.route53.change", execState.Type)
		payload := execState.Payloads[0].(map[string]any)
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		change, ok := data["change"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/change/C9876543210", change["id"])
		assert.Equal(t, "INSYNC", change["status"])
		assert.Equal(t, "2026-02-13T14:00:00.000Z", change["submittedAt"])
	})
}

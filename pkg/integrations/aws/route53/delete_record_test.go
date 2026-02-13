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

func TestDeleteRecord_Setup(t *testing.T) {
	component := &DeleteRecord{}

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

	t.Run("missing values -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "example.com",
				"recordType":   "A",
				"ttl":          300,
			},
		})

		require.ErrorContains(t, err, "at least one record value is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    metadata,
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "old.example.com",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(DeleteRecordMetadata)
		require.True(t, ok)
		assert.Equal(t, "Z123", stored.HostedZoneID)
		assert.Equal(t, "old.example.com", stored.RecordName)
	})
}

func TestDeleteRecord_Execute(t *testing.T) {
	component := &DeleteRecord{}

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

	t.Run("success -> emits record change", func(t *testing.T) {
		xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C5555555555</Id>
    <Status>PENDING</Status>
    <SubmittedAt>2026-02-13T15:00:00.000Z</SubmittedAt>
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
				"recordName":   "old.example.com",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
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
		require.Equal(t, "aws.route53.record", execState.Type)

		payload := execState.Payloads[0].(map[string]any)
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/change/C5555555555", data["changeId"])
		assert.Equal(t, "PENDING", data["status"])
		assert.Equal(t, "old.example.com", data["recordName"])
		assert.Equal(t, "A", data["recordType"])
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		xmlError := `<?xml version="1.0" encoding="UTF-8"?>
<ErrorResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Error>
    <Code>InvalidChangeBatch</Code>
    <Message>The resource record set does not exist</Message>
  </Error>
</ErrorResponse>`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(xmlError)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "nonexistent.example.com",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "InvalidChangeBatch")
	})
}

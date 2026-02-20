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

func TestCreateRecord_Setup(t *testing.T) {
	component := &CreateRecord{}

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

	t.Run("missing record type -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "example.com",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
		})

		require.ErrorContains(t, err, "record type is required")
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

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"hostedZoneId": "  Z123  ",
				"recordName":   "  example.com  ",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
			},
		})

		require.NoError(t, err)
	})
}

func TestCreateRecord_Execute(t *testing.T) {
	component := &CreateRecord{}

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
    <Id>/change/C1234567890</Id>
    <Status>PENDING</Status>
    <SubmittedAt>2026-01-28T10:30:00.000Z</SubmittedAt>
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
				"recordName":   "example.com",
				"recordType":   "A",
				"ttl":          300,
				"values":       []string{"1.2.3.4"},
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
		assert.Equal(t, "/change/C1234567890", stored.ChangeID)
		assert.Equal(t, "example.com", stored.RecordName)
		assert.Equal(t, "A", stored.RecordType)
		assert.Equal(t, "2026-01-28T10:30:00.000Z", stored.SubmittedAt)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "hostedzone/Z123/rrset")
	})

	t.Run("status INSYNC -> emits record change immediately", func(t *testing.T) {
		xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C1234567890</Id>
    <Status>INSYNC</Status>
    <SubmittedAt>2026-01-28T10:30:00.000Z</SubmittedAt>
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
				"recordName":   "example.com",
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
		require.Equal(t, "aws.route53.change", execState.Type)
		payload := execState.Payloads[0].(map[string]any)
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		change, ok := data["change"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/change/C1234567890", change["id"])
		assert.Equal(t, "INSYNC", change["status"])
		assert.Equal(t, "2026-01-28T10:30:00.000Z", change["submittedAt"])
	})

	t.Run("API error ErrorResponse format -> returns error", func(t *testing.T) {
		xmlError := `<?xml version="1.0" encoding="UTF-8"?>
<ErrorResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Error>
    <Code>AccessDenied</Code>
    <Message>User is not authorized</Message>
  </Error>
</ErrorResponse>`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(xmlError)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"hostedZoneId": "Z123",
				"recordName":   "example.com",
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
		require.Contains(t, err.Error(), "AccessDenied")
		require.Contains(t, err.Error(), "User is not authorized")
	})

	t.Run("API error InvalidChangeBatch format -> returns error", func(t *testing.T) {
		// Route53 returns this format (not ErrorResponse) for InvalidChangeBatch.
		xmlError := `<?xml version="1.0" encoding="UTF-8"?>
<InvalidChangeBatch xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Messages>
    <Message>RRSet with DNS name dev.example.com. is not permitted in zone example.com.</Message>
  </Messages>
</InvalidChangeBatch>`

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
				"recordName":   "dev.example.com",
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
		require.Contains(t, err.Error(), "RRSet with DNS name dev.example.com")
		require.Contains(t, err.Error(), "not permitted in zone example.com")
	})
}

func TestCreateRecord_HandleAction(t *testing.T) {
	component := &CreateRecord{}

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown action")
	})

	t.Run("pollChange status still PENDING -> schedules poll again", func(t *testing.T) {
		getChangeXML := `<?xml version="1.0" encoding="UTF-8"?>
<GetChangeResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C1234567890</Id>
    <Status>PENDING</Status>
    <SubmittedAt>2026-01-28T10:30:00.000Z</SubmittedAt>
  </ChangeInfo>
</GetChangeResponse>`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(getChangeXML)),
				},
			},
		}
		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:     pollChangeActionName,
			HTTP:     httpContext,
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: RecordChangePollMetadata{
					ChangeID:    "/change/C1234567890",
					RecordName:  "example.com",
					RecordType:  "A",
					SubmittedAt: "2026-01-28T10:30:00.000Z",
				},
			},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, pollChangeActionName, requests.Action)
		assert.Equal(t, pollInterval, requests.Duration)
	})

	t.Run("pollChange status INSYNC -> emits record change", func(t *testing.T) {
		getChangeXML := `<?xml version="1.0" encoding="UTF-8"?>
<GetChangeResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeInfo>
    <Id>/change/C1234567890</Id>
    <Status>INSYNC</Status>
    <SubmittedAt>2026-01-28T10:30:00.000Z</SubmittedAt>
  </ChangeInfo>
</GetChangeResponse>`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(getChangeXML)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name:           pollChangeActionName,
			HTTP:           httpContext,
			ExecutionState: execState,
			Metadata: &contexts.MetadataContext{
				Metadata: RecordChangePollMetadata{
					ChangeID:    "/change/C1234567890",
					RecordName:  "example.com",
					RecordType:  "A",
					SubmittedAt: "2026-01-28T10:30:00.000Z",
				},
			},
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
		assert.Equal(t, "/change/C1234567890", change["id"])
		assert.Equal(t, "INSYNC", change["status"])
		assert.Equal(t, "2026-01-28T10:30:00.000Z", change["submittedAt"])
	})
}

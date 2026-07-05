package hetzner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newTestS3Client(t *testing.T, httpCtx *contexts.HTTPContext) *HetznerS3Client {
	t.Helper()
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"s3AccessKeyId":     "AKIAEXAMPLE",
			"s3SecretAccessKey": "secret",
			"s3Region":          "fsn1",
		},
	}
	client, err := NewHetznerS3Client(httpCtx, integration)
	require.NoError(t, err)
	return client
}

func TestHetznerS3Client_ListObjects_SurfacesTruncatedFlag(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <IsTruncated>true</IsTruncated>
  <Contents>
    <Key>a.txt</Key>
    <Size>10</Size>
  </Contents>
</ListBucketResult>`

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(xmlBody)),
			},
		},
	}
	client := newTestS3Client(t, httpCtx)

	result, err := client.ListObjects("my-bucket", "", 1)
	require.NoError(t, err)
	require.True(t, result.Truncated)
	require.Len(t, result.Items, 1)
	require.Equal(t, "a.txt", result.Items[0].Key)
}

func TestHetznerS3Client_ListObjects_NotTruncated(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>a.txt</Key>
    <Size>10</Size>
  </Contents>
</ListBucketResult>`

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(xmlBody)),
			},
		},
	}
	client := newTestS3Client(t, httpCtx)

	result, err := client.ListObjects("my-bucket", "", 10)
	require.NoError(t, err)
	require.False(t, result.Truncated)
}

package clouddns

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRecordSet_EncodesQueryParams(t *testing.T) {
	var requestedURL string
	client := &mockClient{
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			requestedURL = fullURL
			return json.Marshal(map[string]any{"rrsets": []any{}})
		},
	}

	record, err := getRecordSet(
		context.Background(),
		client,
		"my-project",
		"my-zone",
		"api.example.com.&test=1",
		"TXT&SPF",
	)
	require.NoError(t, err)
	assert.Nil(t, record)

	parsedURL, err := url.Parse(requestedURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "api.example.com.&test=1", query.Get("name"))
	assert.Equal(t, "TXT&SPF", query.Get("type"))
	assert.Empty(t, query.Get("test"))
}

func TestListRecordSetsByName_EncodesQueryParams(t *testing.T) {
	var requestedURL string
	client := &mockClient{
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			requestedURL = fullURL
			return json.Marshal(map[string]any{"rrsets": []any{}})
		},
	}

	records, err := listRecordSetsByName(
		context.Background(),
		client,
		"my-project",
		"my-zone",
		"api.example.com.&test=1",
	)
	require.NoError(t, err)
	assert.Empty(t, records)

	parsedURL, err := url.Parse(requestedURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "api.example.com.&test=1", query.Get("name"))
	assert.Empty(t, query.Get("test"))
}

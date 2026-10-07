package azure

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
)

func TestNewProviderRequiresAccountAndContainer(t *testing.T) {
	t.Setenv(envAccount, "")
	t.Setenv(envContainer, "superplane")
	_, err := NewProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), envAccount)

	t.Setenv(envAccount, "superplanefiles")
	t.Setenv(envContainer, "")
	_, err = NewProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), envContainer)
}

func TestSignedBlobURLKeepsPathSegmentsAndMarker(t *testing.T) {
	signed := signedBlobURL("acct", "superplane", "orgs/id/file name", "sv=2024-11-04&sig=abc")
	assert.Contains(t, signed, "https://acct.blob.core.windows.net/superplane/orgs/id/file%20name?")
	assert.Contains(t, signed, blob.SignedURLMarkerParam+"="+blob.SignedURLMarkerValue)
	assert.Contains(t, signed, "sig=abc")
}

func TestIsNotFoundMapsBlob404(t *testing.T) {
	assert.True(t, isNotFound(&azcore.ResponseError{StatusCode: http.StatusNotFound}))
	assert.False(t, isNotFound(&azcore.ResponseError{StatusCode: http.StatusForbidden}))
	assert.False(t, isNotFound(assert.AnError))
}

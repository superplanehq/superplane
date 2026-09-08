package blob

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileAccessURLRoundTrip(t *testing.T) {
	fileID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	key := []byte("signing-secret")
	t.Setenv("BASE_URL", "https://app.example.test")

	raw, err := FileAccessURL(fileID, time.Hour, key)
	require.NoError(t, err)
	assert.Contains(t, raw, "https://app.example.test/api/v1/public/files/"+fileID.String())
	assert.Contains(t, raw, SignedURLMarkerParam+"="+SignedURLMarkerValue)

	expires := time.Now().Add(time.Hour).Unix()
	sig, err := SignFileAccess(fileID, expires, key)
	require.NoError(t, err)
	require.NoError(t, VerifyFileAccess(fileID, expires, sig, key))
	require.Error(t, VerifyFileAccess(fileID, time.Now().Add(-time.Minute).Unix(), sig, key))
}

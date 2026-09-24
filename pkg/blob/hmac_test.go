package blob

import (
	"net/url"
	"strconv"
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

func TestFileAccessURLStaysStableInsideExpiryBucket(t *testing.T) {
	fileID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	key := []byte("signing-secret")
	t.Setenv("BASE_URL", "https://app.example.test")

	first, err := FileAccessURL(fileID, time.Hour, key)
	require.NoError(t, err)
	second, err := FileAccessURL(fileID, time.Hour, key)
	require.NoError(t, err)
	assert.Equal(t, first, second)

	parsed, err := url.Parse(first)
	require.NoError(t, err)
	expires, err := strconv.ParseInt(parsed.Query().Get("expires"), 10, 64)
	require.NoError(t, err)
	assert.Equal(t, StableExpiry(time.Hour).Unix(), expires)
}

func TestFileIDFromSignedURLReadsHMACAndGCSPaths(t *testing.T) {
	fileID := uuid.MustParse("471f6c01-c28c-49da-afb9-c48964ac9a1d")
	workOrderID := uuid.MustParse("2ecbe7d5-d0f8-4564-b6c1-a04e6c69b178")

	hmacURL := "https://app.example/api/v1/public/files/" + fileID.String() + "?expires=1&sig=abc&sp_file=1"
	id, ok := FileIDFromSignedURL(hmacURL)
	require.True(t, ok)
	assert.Equal(t, fileID, id)

	gcsURL := "https://storage.googleapis.com/superplane-prod-global/881b70a0-5c9e-47da-a4ca-395f402f3aea/orgs/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/workspaces/9155053b-45c3-4a96-b4bc-63820b0f9a98/tasks/" + workOrderID.String() + "/" + fileID.String() + "?X-Goog-Algorithm=GOOG4-RSA-SHA256&sp_file=1"
	id, ok = FileIDFromSignedURL(gcsURL)
	require.True(t, ok)
	assert.Equal(t, fileID, id)

	virtualHost := "https://superplane-prod-global.storage.googleapis.com/881b70a0-5c9e-47da-a4ca-395f402f3aea/orgs/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/workspaces/9155053b-45c3-4a96-b4bc-63820b0f9a98/tasks/" + workOrderID.String() + "/" + fileID.String() + "?sp_file=1"
	id, ok = FileIDFromSignedURL(virtualHost)
	require.True(t, ok)
	assert.Equal(t, fileID, id)

	_, ok = FileIDFromSignedURL("https://example.com/" + fileID.String())
	assert.False(t, ok)

	_, ok = FileIDFromSignedURL("https://example.test/" + fileID.String() + "?sp_file=1")
	assert.False(t, ok)

	s3URL := "https://superplane-files.s3.us-west-2.amazonaws.com/install/orgs/org/workspaces/ws/tasks/" + workOrderID.String() + "/" + fileID.String() + "?X-Amz-Algorithm=AWS4-HMAC-SHA256&sp_file=1"
	id, ok = FileIDFromSignedURL(s3URL)
	require.True(t, ok)
	assert.Equal(t, fileID, id)

	pathStyle := "https://s3.us-west-2.amazonaws.com/superplane-files/install/" + fileID.String() + "?sp_file=1"
	id, ok = FileIDFromSignedURL(pathStyle)
	require.True(t, ok)
	assert.Equal(t, fileID, id)
}

func TestRewriteSignedFileURLsRestoresRefsAndDropsUnknown(t *testing.T) {
	fileID := uuid.MustParse("471f6c01-c28c-49da-afb9-c48964ac9a1d")
	foreign := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	known := "https://storage.googleapis.com/bucket/orgs/x/workspaces/y/tasks/z/" + fileID.String() + "?sp_file=1"
	unknown := "https://app.example/api/v1/public/files/" + foreign.String() + "?expires=1&sig=abc&sp_file=1"
	markdown := "See ![](" + known + ") and ![](" + unknown + ")"

	rewritten := RewriteSignedFileURLs(markdown, func(id uuid.UUID) (string, bool) {
		if id == fileID {
			return FileRef(id), true
		}
		return "", false
	})
	assert.Contains(t, rewritten, FileRef(fileID))
	assert.NotContains(t, rewritten, "sp_file=1")
	assert.NotContains(t, rewritten, foreign.String())
}

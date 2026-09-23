package s3

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
)

func TestNewProviderRequiresBucketAndRegion(t *testing.T) {
	t.Setenv("BLOB_STORAGE_BUCKET", "")
	t.Setenv("BLOB_STORAGE_REGION", "")
	t.Setenv("AWS_REGION", "")

	_, err := NewProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BLOB_STORAGE_BUCKET")

	t.Setenv("BLOB_STORAGE_BUCKET", "superplane-files")
	_, err = NewProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BLOB_STORAGE_REGION")
}

func TestNewProviderUsesRegionAndBucket(t *testing.T) {
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("BLOB_STORAGE_BUCKET", "superplane-files")
	t.Setenv("BLOB_STORAGE_REGION", "us-west-2")
	t.Setenv("AWS_REGION", "")

	store, err := NewProvider()
	require.NoError(t, err)
	assert.Equal(t, blob.ProviderS3, store.Name())
	assert.Equal(t, "superplane-files", store.bucket)
}

func TestStorePutGetHeadDelete(t *testing.T) {
	objects := newFakeObjects()
	store, err := newStore("superplane-files", objects, rejectPresign{})
	require.NoError(t, err)

	ctx := context.Background()
	key := "install/orgs/abc/file-1"
	payload := []byte("png-bytes")
	require.NoError(t, store.Put(ctx, key, bytes.NewReader(payload), blob.PutOptions{ContentType: "image/png"}))

	info, err := store.Head(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), info.Size)
	assert.Equal(t, "image/png", info.ContentType)

	reader, err := store.Get(ctx, key)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, payload, got)

	require.NoError(t, store.Delete(ctx, key))
	_, err = store.Head(ctx, key)
	assert.ErrorIs(t, err, blob.ErrNotFound)
	_, err = store.Get(ctx, key)
	assert.ErrorIs(t, err, blob.ErrNotFound)
	require.NoError(t, store.Delete(ctx, key))
}

func TestSignedGetURLIncludesFileMarker(t *testing.T) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-west-2"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET", "")),
	)
	require.NoError(t, err)
	client := awss3.NewFromConfig(cfg)
	store, err := newStore("superplane-files", newFakeObjects(), awsPresignClient{client: awss3.NewPresignClient(client)})
	require.NoError(t, err)

	fileID := uuid.MustParse("471f6c01-c28c-49da-afb9-c48964ac9a1d")
	key := "install/orgs/abc/" + fileID.String()
	first, err := store.SignedGetURL(context.Background(), key, 0)
	require.NoError(t, err)
	second, err := store.SignedGetURL(context.Background(), key, 0)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Contains(t, first, "X-Amz-Signature=")
	assert.Contains(t, first, blob.SignedURLMarkerParam+"="+blob.SignedURLMarkerValue)

	id, ok := blob.FileIDFromSignedURL(first)
	require.True(t, ok)
	assert.Equal(t, fileID, id)
}

type rejectPresign struct{}

func (rejectPresign) PresignGetObject(context.Context, *awss3.GetObjectInput, ...func(*awss3.PresignOptions)) (*v4Presign, error) {
	return nil, assert.AnError
}

type storedObject struct {
	body        []byte
	contentType string
}

type fakeObjects struct {
	objects map[string]storedObject
}

func newFakeObjects() *fakeObjects {
	return &fakeObjects{objects: map[string]storedObject{}}
}

func (f *fakeObjects) PutObject(_ context.Context, params *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	contentType := ""
	if params.ContentType != nil {
		contentType = *params.ContentType
	}
	f.objects[aws.ToString(params.Key)] = storedObject{body: body, contentType: contentType}
	return &awss3.PutObjectOutput{}, nil
}

func (f *fakeObjects) GetObject(_ context.Context, params *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	object, ok := f.objects[aws.ToString(params.Key)]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &awss3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(object.body))}, nil
}

func (f *fakeObjects) HeadObject(_ context.Context, params *awss3.HeadObjectInput, _ ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	object, ok := f.objects[aws.ToString(params.Key)]
	if !ok {
		return nil, &types.NotFound{}
	}
	size := int64(len(object.body))
	return &awss3.HeadObjectOutput{
		ContentLength: &size,
		ContentType:   aws.String(object.contentType),
	}, nil
}

func (f *fakeObjects) DeleteObject(_ context.Context, params *awss3.DeleteObjectInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	delete(f.objects, aws.ToString(params.Key))
	return &awss3.DeleteObjectOutput{}, nil
}

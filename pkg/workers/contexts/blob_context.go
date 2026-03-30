package contexts

import (
	"context"
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/blobstorage"
)

type BlobContext struct {
	store blobstorage.BlobStorage
}

func NewBlobContext(store blobstorage.BlobStorage) *BlobContext {
	return &BlobContext{store: store}
}

func (c *BlobContext) Put(key string, body io.Reader, size int64, contentType string) (string, error) {
	if c.store == nil {
		return "", fmt.Errorf("blob storage is not configured")
	}

	output, err := c.store.Put(context.Background(), blobstorage.PutInput{
		Key:         key,
		Body:        body,
		Size:        size,
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return output.ETag, nil
}

func (c *BlobContext) Get(key string) (io.ReadCloser, int64, string, error) {
	if c.store == nil {
		return nil, 0, "", fmt.Errorf("blob storage is not configured")
	}

	output, err := c.store.Get(context.Background(), key)
	if err != nil {
		return nil, 0, "", err
	}

	return output.Body, output.Size, output.ContentType, nil
}

func (c *BlobContext) Delete(key string) error {
	if c.store == nil {
		return fmt.Errorf("blob storage is not configured")
	}

	return c.store.Delete(context.Background(), key)
}

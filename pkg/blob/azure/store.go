package azure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	azureblob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/superplanehq/superplane/pkg/blob"
)

const (
	envAccount   = "BLOB_STORAGE_ACCOUNT"
	envContainer = "BLOB_STORAGE_BUCKET"
)

type Store struct {
	account   string
	container string
	client    *azblob.Client
	service   *service.Client
}

func NewProvider() (*Store, error) {
	account := strings.TrimSpace(os.Getenv(envAccount))
	if account == "" {
		return nil, fmt.Errorf("%s is not set", envAccount)
	}
	container := strings.TrimSpace(os.Getenv(envContainer))
	if container == "" {
		return nil, fmt.Errorf("%s is not set", envContainer)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure credential: %w", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", account)
	client, err := azblob.NewClient(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure Blob client: %w", err)
	}
	return &Store{
		account:   account,
		container: container,
		client:    client,
		service:   client.ServiceClient(),
	}, nil
}

func (s *Store) Name() string {
	return blob.ProviderAzure
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) error {
	uploadOpts := &azblob.UploadStreamOptions{}
	if contentType := strings.TrimSpace(opts.ContentType); contentType != "" {
		uploadOpts.HTTPHeaders = &azureblob.HTTPHeaders{
			BlobContentType: &contentType,
		}
	}
	if contentEncoding := strings.TrimSpace(opts.ContentEncoding); contentEncoding != "" {
		if uploadOpts.HTTPHeaders == nil {
			uploadOpts.HTTPHeaders = &azureblob.HTTPHeaders{}
		}
		uploadOpts.HTTPHeaders.BlobContentEncoding = &contentEncoding
	}
	_, err := s.client.UploadStream(ctx, s.container, key, r, uploadOpts)
	if err != nil {
		return fmt.Errorf("write Azure blob: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.download(ctx, key, azblob.HTTPRange{})
}

func (s *Store) GetRange(
	ctx context.Context,
	key string,
	offset, length int64,
) (io.ReadCloser, error) {
	if offset < 0 {
		offset = 0
	}
	httpRange := azblob.HTTPRange{Offset: offset}
	if length >= 0 {
		httpRange.Count = length
	}
	return s.download(ctx, key, httpRange)
}

func (s *Store) download(
	ctx context.Context,
	key string,
	httpRange azblob.HTTPRange,
) (io.ReadCloser, error) {
	response, err := s.client.DownloadStream(ctx, s.container, key, &azblob.DownloadStreamOptions{
		Range: httpRange,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("read Azure blob: %w", err)
	}
	return response.Body, nil
}

func (s *Store) Head(ctx context.Context, key string) (*blob.ObjectInfo, error) {
	properties, err := s.service.
		NewContainerClient(s.container).
		NewBlobClient(key).
		GetProperties(ctx, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("head Azure blob: %w", err)
	}
	info := &blob.ObjectInfo{}
	if properties.ContentLength != nil {
		info.Size = *properties.ContentLength
	}
	if properties.ContentType != nil {
		info.ContentType = *properties.ContentType
	}
	if properties.ContentEncoding != nil {
		info.ContentEncoding = *properties.ContentEncoding
	}
	return info, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteBlob(ctx, s.container, key, nil)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete Azure blob: %w", err)
	}
	return nil
}

func (s *Store) SignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	expires := blob.StableExpiry(ttl)
	start := expires.Add(-ttl - 10*time.Minute)
	if start.After(time.Now().UTC()) {
		start = time.Now().UTC().Add(-10 * time.Minute)
	}
	permissions := sas.BlobPermissions{Read: true}
	credential, err := s.service.GetUserDelegationCredential(ctx, service.KeyInfo{
		Start:  to.Ptr(start.UTC().Format(sas.TimeFormat)),
		Expiry: to.Ptr(expires.UTC().Format(sas.TimeFormat)),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("get Azure user delegation key: %w", err)
	}
	query, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     start,
		ExpiryTime:    expires,
		Permissions:   permissions.String(),
		ContainerName: s.container,
		BlobName:      key,
	}.SignWithUserDelegation(credential)
	if err != nil {
		return "", fmt.Errorf("sign Azure blob GET: %w", err)
	}
	return signedBlobURL(s.account, s.container, key, query.Encode()), nil
}

func signedBlobURL(account, containerName, key, sasQuery string) string {
	objectURL := fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s/%s",
		account,
		url.PathEscape(containerName),
		escapeBlobKey(key),
	)
	values, err := url.ParseQuery(sasQuery)
	if err != nil {
		values = url.Values{}
	}
	values.Set(blob.SignedURLMarkerParam, blob.SignedURLMarkerValue)
	return objectURL + "?" + values.Encode()
}

func escapeBlobKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func isNotFound(err error) bool {
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		return responseErr.StatusCode == 404
	}
	return false
}

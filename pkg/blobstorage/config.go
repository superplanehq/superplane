package blobstorage

import (
	"fmt"
	"os"
)

const (
	EnvBlobStorageBackend        = "BLOB_STORAGE_BACKEND"
	EnvBlobStorageFilesystemPath = "BLOB_STORAGE_FILESYSTEM_PATH"
)

func NewFromEnv() (BlobStorage, error) {
	backend := os.Getenv(EnvBlobStorageBackend)
	if backend == "" {
		backend = BackendMemory
	}

	switch backend {
	case BackendMemory:
		return NewInMemoryBlobStorage(), nil
	case BackendFilesystem:
		basePath := os.Getenv(EnvBlobStorageFilesystemPath)
		if basePath == "" {
			return nil, fmt.Errorf("%s must be set when %s=%s", EnvBlobStorageFilesystemPath, EnvBlobStorageBackend, BackendFilesystem)
		}

		return NewFilesystemBlobStorage(basePath), nil
	case BackendS3, BackendGCS:
		return nil, fmt.Errorf("blob storage backend %q is not implemented yet", backend)
	default:
		return nil, fmt.Errorf("invalid blob storage backend %q", backend)
	}
}

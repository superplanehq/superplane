package blobs

import (
	"context"
	"fmt"
	"os"

	gcsstorage "cloud.google.com/go/storage"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/api/option"
)

const (
	EnvBackend = "BLOB_STORAGE_BACKEND"

	EnvFilesystemPath = "BLOB_STORAGE_FILESYSTEM_PATH"

	EnvS3Bucket    = "BLOB_STORAGE_S3_BUCKET"
	EnvS3Region    = "BLOB_STORAGE_S3_REGION"
	EnvS3Endpoint  = "BLOB_STORAGE_S3_ENDPOINT"
	EnvS3AccessKey = "BLOB_STORAGE_S3_ACCESS_KEY"
	EnvS3SecretKey = "BLOB_STORAGE_S3_SECRET_KEY"

	EnvGCSBucket          = "BLOB_STORAGE_GCS_BUCKET"
	EnvGCSCredentialsFile = "BLOB_STORAGE_GCS_CREDENTIALS_FILE"
)

func NewFromEnv() (Storage, error) {
	backend := os.Getenv(EnvBackend)
	if backend == "" {
		backend = BackendMemory
	}

	switch backend {
	case BackendMemory:
		return NewMemoryStorage(), nil

	case BackendFilesystem:
		basePath := os.Getenv(EnvFilesystemPath)
		if basePath == "" {
			return nil, fmt.Errorf("%s is required when backend is %s", EnvFilesystemPath, BackendFilesystem)
		}
		return NewFilesystemStorage(basePath), nil

	case BackendS3:
		return newS3StorageFromEnv()

	case BackendGCS:
		return newGCSStorageFromEnv()

	default:
		return nil, fmt.Errorf("unknown blob storage backend: %q", backend)
	}
}

func newS3StorageFromEnv() (Storage, error) {
	bucket := os.Getenv(EnvS3Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("%s is required when backend is %s", EnvS3Bucket, BackendS3)
	}

	region := os.Getenv(EnvS3Region)
	if region == "" {
		region = "us-east-1"
	}

	ctx := context.Background()
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	accessKey := os.Getenv(EnvS3AccessKey)
	secretKey := os.Getenv(EnvS3SecretKey)
	if accessKey != "" && secretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var s3Opts []func(*s3.Options)
	if endpoint := os.Getenv(EnvS3Endpoint); endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, s3Opts...)
	return NewS3Storage(bucket, client), nil
}

func newGCSStorageFromEnv() (Storage, error) {
	bucket := os.Getenv(EnvGCSBucket)
	if bucket == "" {
		return nil, fmt.Errorf("%s is required when backend is %s", EnvGCSBucket, BackendGCS)
	}

	ctx := context.Background()
	var opts []option.ClientOption

	if credFile := os.Getenv(EnvGCSCredentialsFile); credFile != "" {
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, credFile))
	}

	client, err := gcsstorage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return NewGCSStorage(bucket, client), nil
}

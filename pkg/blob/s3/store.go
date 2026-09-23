package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/superplanehq/superplane/pkg/blob"
)

type objectAPI interface {
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
}

type presignAPI interface {
	PresignGetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.PresignOptions)) (*v4Presign, error)
}

// v4Presign matches the URL field of the AWS presign result without importing
// the signer package into every caller. The real presign client is adapted below.
type v4Presign struct {
	URL string
}

type awsPresignClient struct {
	client *awss3.PresignClient
}

func (c awsPresignClient) PresignGetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.PresignOptions)) (*v4Presign, error) {
	result, err := c.client.PresignGetObject(ctx, params, optFns...)
	if err != nil {
		return nil, err
	}
	return &v4Presign{URL: result.URL}, nil
}

type Store struct {
	bucket    string
	objects   objectAPI
	presigner presignAPI
}

func NewProvider() (*Store, error) {
	bucket := strings.TrimSpace(os.Getenv("BLOB_STORAGE_BUCKET"))
	if bucket == "" {
		return nil, fmt.Errorf("BLOB_STORAGE_BUCKET is not set")
	}
	region := strings.TrimSpace(os.Getenv("BLOB_STORAGE_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	}
	if region == "" {
		return nil, fmt.Errorf("BLOB_STORAGE_REGION is not set")
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	client := awss3.NewFromConfig(cfg)
	return newStore(bucket, client, awsPresignClient{client: awss3.NewPresignClient(client)})
}

func newStore(bucket string, objects objectAPI, presigner presignAPI) (*Store, error) {
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if objects == nil || presigner == nil {
		return nil, fmt.Errorf("S3 client is required")
	}
	return &Store{bucket: bucket, objects: objects, presigner: presigner}, nil
}

func (s *Store) Name() string {
	return blob.ProviderS3
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) error {
	input := &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	}
	if contentType := strings.TrimSpace(opts.ContentType); contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := s.objects.PutObject(ctx, input); err != nil {
		return fmt.Errorf("write S3 object: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.objects.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("read S3 object: %w", err)
	}
	return output.Body, nil
}

func (s *Store) Head(ctx context.Context, key string) (*blob.ObjectInfo, error) {
	output, err := s.objects.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("head S3 object: %w", err)
	}
	info := &blob.ObjectInfo{}
	if output.ContentLength != nil {
		info.Size = *output.ContentLength
	}
	if output.ContentType != nil {
		info.ContentType = *output.ContentType
	}
	return info, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.objects.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func (s *Store) SignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	expires := blob.StableExpiry(ttl)
	lifetime := time.Until(expires)
	if lifetime < time.Second {
		lifetime = time.Second
	}
	result, err := s.presigner.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *awss3.PresignOptions) {
		opts.Expires = lifetime
		opts.ClientOptions = append(opts.ClientOptions, func(options *awss3.Options) {
			options.APIOptions = append(options.APIOptions, addSignedQuery(blob.SignedURLMarkerParam, blob.SignedURLMarkerValue))
		})
	})
	if err != nil {
		return "", fmt.Errorf("sign S3 GET: %w", err)
	}
	return result.URL, nil
}

func addSignedQuery(name, value string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Build.Add(middleware.BuildMiddlewareFunc("AddBlobSignedQuery", func(
			ctx context.Context,
			in middleware.BuildInput,
			next middleware.BuildHandler,
		) (middleware.BuildOutput, middleware.Metadata, error) {
			request, ok := in.Request.(*smithyhttp.Request)
			if !ok {
				return next.HandleBuild(ctx, in)
			}
			query := request.URL.Query()
			query.Set(name, value)
			request.URL.RawQuery = query.Encode()
			return next.HandleBuild(ctx, in)
		}), middleware.Before)
	}
}

func isNotFound(err error) bool {
	var notFound *types.NotFound
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &notFound) || errors.As(err, &noSuchKey) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return true
		}
	}
	return false
}

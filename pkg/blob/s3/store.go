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

type Store struct {
	bucket    string
	client    *awss3.Client
	presigner *awss3.PresignClient
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
	return &Store{
		bucket:    bucket,
		client:    client,
		presigner: awss3.NewPresignClient(client),
	}, nil
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
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("write S3 object: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
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
	output, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
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
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
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

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

/*
 * S3Storage is a storage implementation that uses Amazon S3.
 */
type S3Storage struct {
	bucket string
	region string
	client *s3.Client
}

func NewS3Storage(ctx context.Context, bucket, region string) (*S3Storage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &S3Storage{
		bucket: bucket,
		region: region,
		client: s3.NewFromConfig(cfg),
	}, nil
}

func (s *S3Storage) Class() string {
	return "s3"
}

func (s *S3Storage) Provision(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &s.bucket,
	})
	if err == nil {
		return nil
	}

	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		return fmt.Errorf("head s3 bucket %q: %w", s.bucket, err)
	}

	input := &s3.CreateBucketInput{
		Bucket: &s.bucket,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		},
	}

	_, err = s.client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("create s3 bucket %q: %w", s.bucket, err)
	}

	return nil
}

func (s *S3Storage) Write(ctx context.Context, path string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
		Body:   body,
	})

	if err != nil {
		return fmt.Errorf("write s3 object %q: %w", path, err)
	}

	return nil
}

func (s *S3Storage) Read(ctx context.Context, path string) (io.Reader, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	})

	if err != nil {
		return nil, fmt.Errorf("read s3 object %q: %w", path, err)
	}

	return output.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	})
	if err != nil {
		return fmt.Errorf("delete s3 object %q: %w", path, err)
	}

	return nil
}

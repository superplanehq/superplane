package blobs

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3Storage struct {
	bucket        string
	client        *s3.Client
	presignClient *s3.PresignClient
}

func NewS3Storage(bucket string, client *s3.Client) *S3Storage {
	return &S3Storage{
		bucket:        bucket,
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}
}

func (s *S3Storage) Put(ctx context.Context, scope Scope, path string, body io.Reader, opts PutOptions) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	_, err = s.client.PutObject(ctx, input)
	return err
}

func (s *S3Storage) Get(ctx context.Context, scope Scope, path string) (io.ReadCloser, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, ErrBlobNotFound
		}
		return nil, err
	}

	return output.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, scope Scope, path string) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}

	_, err = s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return ErrBlobNotFound
		}
		return err
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return ErrBlobNotFound
		}
		return err
	}

	return nil
}

func (s *S3Storage) List(ctx context.Context, scope Scope, input ListInput) (*ListOutput, error) {
	prefix, err := scopePrefix(scope)
	if err != nil {
		return nil, err
	}

	maxKeys := int32(100)
	if input.MaxResults > 0 && input.MaxResults <= 1000 {
		maxKeys = int32(input.MaxResults)
	}

	listInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	if input.ContinuationToken != "" {
		listInput.ContinuationToken = aws.String(input.ContinuationToken)
	}

	output, err := s.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return nil, err
	}

	blobs := make([]BlobInfo, 0, len(output.Contents))
	for _, obj := range output.Contents {
		blobs = append(blobs, BlobInfo{
			Path:      strings.TrimPrefix(aws.ToString(obj.Key), prefix),
			Size:      aws.ToInt64(obj.Size),
			UpdatedAt: aws.ToTime(obj.LastModified),
		})
	}

	var nextToken string
	if output.NextContinuationToken != nil {
		nextToken = *output.NextContinuationToken
	}

	return &ListOutput{
		Blobs:     blobs,
		NextToken: nextToken,
	}, nil
}

func (s *S3Storage) PresignPut(ctx context.Context, scope Scope, path string, opts PutOptions, expiry time.Duration) (*PresignedURL, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	result, err := s.presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, err
	}

	return &PresignedURL{
		URL:       result.URL,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

func (s *S3Storage) PresignGet(ctx context.Context, scope Scope, path string, expiry time.Duration) (*PresignedURL, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	result, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, err
	}

	return &PresignedURL{
		URL:       result.URL,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

func isS3NotFound(err error) bool {
	var notFound *types.NoSuchKey
	if errors.As(err, &notFound) {
		return true
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey"
	}

	return false
}

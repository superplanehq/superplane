package s3

import "encoding/xml"

// Bucket represents an S3 bucket returned by API operations.
type Bucket struct {
	Name     string `json:"name" mapstructure:"name"`
	Location string `json:"location,omitempty" mapstructure:"location"`
}

// Object represents an S3 object in a bucket listing.
type Object struct {
	Key          string `json:"key" mapstructure:"key"`
	Size         int64  `json:"size" mapstructure:"size"`
	ETag         string `json:"etag,omitempty" mapstructure:"etag"`
	StorageClass string `json:"storageClass,omitempty" mapstructure:"storageClass"`
	LastModified string `json:"lastModified,omitempty" mapstructure:"lastModified"`
}

// ObjectMetadata represents metadata from a HeadObject response.
type ObjectMetadata struct {
	Key           string `json:"key" mapstructure:"key"`
	Bucket        string `json:"bucket" mapstructure:"bucket"`
	ContentLength int64  `json:"contentLength" mapstructure:"contentLength"`
	ContentType   string `json:"contentType,omitempty" mapstructure:"contentType"`
	ETag          string `json:"etag,omitempty" mapstructure:"etag"`
	LastModified  string `json:"lastModified,omitempty" mapstructure:"lastModified"`
	StorageClass  string `json:"storageClass,omitempty" mapstructure:"storageClass"`
}

// BucketInfo represents metadata from a HeadBucket response.
type BucketInfo struct {
	Name   string `json:"name" mapstructure:"name"`
	Region string `json:"region,omitempty" mapstructure:"region"`
	Exists bool   `json:"exists" mapstructure:"exists"`
}

// CopyObjectResult represents the result of a CopyObject operation.
type CopyObjectResult struct {
	SourceBucket string `json:"sourceBucket" mapstructure:"sourceBucket"`
	SourceKey    string `json:"sourceKey" mapstructure:"sourceKey"`
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	Key          string `json:"key" mapstructure:"key"`
	ETag         string `json:"etag,omitempty" mapstructure:"etag"`
	LastModified string `json:"lastModified,omitempty" mapstructure:"lastModified"`
}

// PutObjectResult represents the result of a PutObject operation.
type PutObjectResult struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
	ETag   string `json:"etag,omitempty" mapstructure:"etag"`
}

// ObjectAttributes represents the result of a GetObjectAttributes operation.
type ObjectAttributes struct {
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	Key          string `json:"key" mapstructure:"key"`
	ETag         string `json:"etag,omitempty" mapstructure:"etag"`
	StorageClass string `json:"storageClass,omitempty" mapstructure:"storageClass"`
	ObjectSize   int64  `json:"objectSize" mapstructure:"objectSize"`
}

// EmptyBucketResult represents the result of emptying a bucket.
type EmptyBucketResult struct {
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	DeletedCount int    `json:"deletedCount" mapstructure:"deletedCount"`
}

// XML response types for S3 API parsing.

type copyObjectResponse struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

type listObjectsV2Response struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	Contents              []objectEntry  `xml:"Contents"`
	IsTruncated           bool           `xml:"IsTruncated"`
	NextContinuationToken string         `xml:"NextContinuationToken"`
	KeyCount              int            `xml:"KeyCount"`
	CommonPrefixes        []commonPrefix `xml:"CommonPrefixes"`
}

type objectEntry struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	ETag         string `xml:"ETag"`
	StorageClass string `xml:"StorageClass"`
	LastModified string `xml:"LastModified"`
}

type commonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type deleteRequest struct {
	XMLName xml.Name                 `xml:"Delete"`
	Quiet   bool                     `xml:"Quiet"`
	Objects []deleteObjectIdentifier `xml:"Object"`
}

type deleteObjectIdentifier struct {
	Key string `xml:"Key"`
}

type deleteObjectsResponse struct {
	XMLName xml.Name       `xml:"DeleteResult"`
	Deleted []deletedEntry `xml:"Deleted"`
	Errors  []deleteError  `xml:"Error"`
}

type deletedEntry struct {
	Key string `xml:"Key"`
}

type deleteError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type getObjectAttributesResponse struct {
	XMLName      xml.Name `xml:"GetObjectAttributesResponse"`
	ETag         string   `xml:"ETag"`
	StorageClass string   `xml:"StorageClass"`
	ObjectSize   int64    `xml:"ObjectSize"`
}

type listBucketsResponse struct {
	XMLName xml.Name      `xml:"ListAllMyBucketsResult"`
	Buckets []bucketEntry `xml:"Buckets>Bucket"`
}

type bucketEntry struct {
	Name string `xml:"Name"`
}

type createBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint"`
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

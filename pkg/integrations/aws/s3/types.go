package s3

import "encoding/xml"

type Bucket struct {
	Name     string `json:"name" mapstructure:"name"`
	Region   string `json:"region" mapstructure:"region"`
	Location string `json:"location,omitempty" mapstructure:"location"`
}

type HeadBucketResult struct {
	BucketName    string `json:"bucketName" mapstructure:"bucketName"`
	Region        string `json:"region,omitempty" mapstructure:"region"`
	AccessPointID string `json:"accessPointId,omitempty" mapstructure:"accessPointId"`
}

type PutObjectResult struct {
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
	ETag   string `json:"etag,omitempty" mapstructure:"etag"`
}

type CopyObjectResult struct {
	SourceBucket string `json:"sourceBucket" mapstructure:"sourceBucket"`
	SourceKey    string `json:"sourceKey" mapstructure:"sourceKey"`
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	Key          string `json:"key" mapstructure:"key"`
	ETag         string `json:"etag,omitempty" mapstructure:"etag"`
	LastModified string `json:"lastModified,omitempty" mapstructure:"lastModified"`
}

type HeadObjectResult struct {
	Bucket        string `json:"bucket" mapstructure:"bucket"`
	Key           string `json:"key" mapstructure:"key"`
	ContentLength string `json:"contentLength,omitempty" mapstructure:"contentLength"`
	ContentType   string `json:"contentType,omitempty" mapstructure:"contentType"`
	ETag          string `json:"etag,omitempty" mapstructure:"etag"`
	LastModified  string `json:"lastModified,omitempty" mapstructure:"lastModified"`
	StorageClass  string `json:"storageClass,omitempty" mapstructure:"storageClass"`
}

type ObjectAttributes struct {
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	Key          string `json:"key" mapstructure:"key"`
	ETag         string `json:"etag,omitempty" mapstructure:"etag"`
	StorageClass string `json:"storageClass,omitempty" mapstructure:"storageClass"`
	ObjectSize   int64  `json:"objectSize,omitempty" mapstructure:"objectSize"`
}

type ObjectSummary struct {
	Key  string `json:"key" mapstructure:"key"`
	ETag string `json:"etag,omitempty" mapstructure:"etag"`
	Size int64  `json:"size" mapstructure:"size"`
}

type BucketSummary struct {
	Name         string `json:"name" mapstructure:"name"`
	CreationDate string `json:"creationDate,omitempty" mapstructure:"creationDate"`
}

// XML request/response types

type createBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint"`
}

type copyObjectResponse struct {
	ETag         string `xml:"ETag"`
	LastModified string `xml:"LastModified"`
}

type getObjectAttributesResponse struct {
	ETag         string `xml:"ETag"`
	StorageClass string `xml:"StorageClass"`
	ObjectSize   int64  `xml:"ObjectSize"`
}

type listObjectsV2Response struct {
	Contents              []listObjectContent `xml:"Contents"`
	IsTruncated           bool                `xml:"IsTruncated"`
	NextContinuationToken string              `xml:"NextContinuationToken"`
}

type listObjectContent struct {
	Key  string `xml:"Key"`
	ETag string `xml:"ETag"`
	Size int64  `xml:"Size"`
}

type listBucketsResponse struct {
	Buckets []listBucketEntry `xml:"Buckets>Bucket"`
}

type listBucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type s3ErrorPayload struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

// bucketPayloadType is the type tag emitted by every Cloud Storage bucket
// component, so downstream steps can branch on a single, stable payload type.
const bucketPayloadType = "gcp.storage.bucket"

// roleHintAdmin is the IAM role required to manage (create/delete) buckets;
// roleHintViewer is the read-only role sufficient for the get component.
const (
	roleHintAdmin  = "roles/storage.admin"
	roleHintViewer = "roles/storage.admin (or roles/storage.legacyBucketReader)"
)

// Bucket models the subset of a Cloud Storage bucket resource the components use.
type Bucket struct {
	Kind             string            `json:"kind"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	SelfLink         string            `json:"selfLink"`
	ProjectNumber    string            `json:"projectNumber"`
	Location         string            `json:"location"`
	LocationType     string            `json:"locationType"`
	StorageClass     string            `json:"storageClass"`
	TimeCreated      string            `json:"timeCreated"`
	Updated          string            `json:"updated"`
	Metageneration   string            `json:"metageneration"`
	Labels           map[string]string `json:"labels"`
	Versioning       *bucketVersioning `json:"versioning"`
	IamConfiguration *iamConfiguration `json:"iamConfiguration"`
	Etag             string            `json:"etag"`
}

type bucketVersioning struct {
	Enabled bool `json:"enabled"`
}

type iamConfiguration struct {
	UniformBucketLevelAccess *uniformBucketLevelAccess `json:"uniformBucketLevelAccess"`
}

type uniformBucketLevelAccess struct {
	Enabled bool `json:"enabled"`
}

// BucketNodeMetadata is the node metadata shared by the bucket components so the
// collapsed node can show what it targets.
type BucketNodeMetadata struct {
	Bucket string `json:"bucket,omitempty" mapstructure:"bucket"`
}

func bucketsURL(project string) string {
	return fmt.Sprintf("%s/b?project=%s", storageBaseURL, url.QueryEscape(project))
}

func bucketURL(bucket string) string {
	return fmt.Sprintf("%s/b/%s", storageBaseURL, url.PathEscape(bucket))
}

// consoleURL is the Cloud Console URL for browsing a bucket's objects. It gives
// the user a one-click way to open the bucket in the GCP console.
func consoleURL(bucket string) string {
	return fmt.Sprintf("https://console.cloud.google.com/storage/browser/%s", url.PathEscape(bucket))
}

// createBucket creates a bucket in the project and returns the created bucket.
// Cloud Storage bucket creation is synchronous, so the response already carries
// the full resource.
func createBucket(ctx context.Context, client Client, project string, body map[string]any) (*Bucket, error) {
	respBody, err := client.PostURL(ctx, bucketsURL(project), body)
	if err != nil {
		return nil, err
	}
	return parseBucket(respBody)
}

// getBucket fetches a single bucket.
func getBucket(ctx context.Context, client Client, bucket string) (*Bucket, error) {
	respBody, err := client.GetURL(ctx, bucketURL(bucket))
	if err != nil {
		return nil, err
	}
	return parseBucket(respBody)
}

// deleteBucket deletes a bucket. The bucket must be empty; Cloud Storage returns
// a 409 otherwise. Deletion is synchronous and returns an empty body.
func deleteBucket(ctx context.Context, client Client, bucket string) error {
	_, err := client.DeleteURL(ctx, bucketURL(bucket))
	return err
}

// ListBuckets lists the Cloud Storage buckets in the project, following
// pagination so projects with more buckets than one page are fully listed.
func ListBuckets(ctx context.Context, client Client, project string) ([]Bucket, error) {
	var all []Bucket
	pageToken := ""
	for {
		u := bucketsURL(project)
		if pageToken != "" {
			u += "&pageToken=" + url.QueryEscape(pageToken)
		}
		respBody, err := client.GetURL(ctx, u)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Items         []Bucket `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("parse buckets list: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextPageToken == "" {
			return all, nil
		}
		pageToken = resp.NextPageToken
	}
}

func parseBucket(body []byte) (*Bucket, error) {
	var b Bucket
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, fmt.Errorf("parse bucket response: %w", err)
	}
	return &b, nil
}

// bucketPayload converts a Bucket into the component output payload.
func bucketPayload(b *Bucket) map[string]any {
	payload := map[string]any{
		"id":           b.ID,
		"name":         b.Name,
		"location":     b.Location,
		"locationType": b.LocationType,
		"storageClass": b.StorageClass,
		"selfLink":     b.SelfLink,
		"consoleUrl":   consoleURL(b.Name),
	}
	if b.ProjectNumber != "" {
		payload["projectNumber"] = b.ProjectNumber
	}
	if b.TimeCreated != "" {
		payload["timeCreated"] = b.TimeCreated
	}
	if b.Updated != "" {
		payload["updated"] = b.Updated
	}
	if b.Versioning != nil {
		payload["versioning"] = b.Versioning.Enabled
	}
	if b.IamConfiguration != nil && b.IamConfiguration.UniformBucketLevelAccess != nil {
		payload["uniformBucketLevelAccess"] = b.IamConfiguration.UniformBucketLevelAccess.Enabled
	}
	if len(b.Labels) > 0 {
		payload["labels"] = b.Labels
	}
	return payload
}

// apiErrorMessage formats an API error for the execution state, appending the
// IAM role the component needs on a 403 (a missing role is the most common
// cause). Callers pass the role appropriate to the operation (read vs. write).
func apiErrorMessage(action string, err error, roleHint string) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHint)
	}
	return fmt.Sprintf("%s: %v", action, err)
}

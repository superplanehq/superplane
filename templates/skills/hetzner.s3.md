# hetzner.s3 — Object Storage Components

S3-compatible object storage operations for Hetzner Object Storage buckets.

## Prerequisites

The Hetzner integration must have S3 credentials configured: **S3 Access Key ID**, **S3 Secret Access Key**, and **S3 Region** (`fsn1` or `nbg1`). These are separate from the Hetzner Cloud API token.

---

## hetzner.createBucket

Create a new bucket in Hetzner Object Storage.

### When to use
- Provision a bucket as part of environment setup (e.g. before uploading deployment artifacts)
- Create tenant-specific buckets in multi-tenant provisioning workflows

### Expected inputs
- `Bucket`: Name of the bucket to create (supports expressions)

### Output
- `bucket`: Bucket name
- `region`: Hetzner region (e.g. `fsn1`)
- `endpoint`: Full bucket endpoint URL

---

## hetzner.deleteBucket

Delete an existing Hetzner Object Storage bucket. The bucket must be empty.

### When to use
- Tear down ephemeral environment buckets after the environment is destroyed
- Clean up test buckets at the end of a workflow

### Expected inputs
- `Bucket`: Bucket to delete (dropdown from integration)

### Output
- `bucket`: Deleted bucket name

---

## hetzner.uploadObject

Upload content to an object in a Hetzner Object Storage bucket.

### When to use
- Store deployment artifacts (build outputs, Docker tarballs) after a CI pipeline
- Write workflow-generated reports or JSON payloads to object storage
- Save init scripts or config files for later use during environment setup

### Expected inputs
- `Bucket`: Target bucket (dropdown or expression)
- `Key`: Object key / path within the bucket (supports expressions, e.g. `artifacts/{{ $.run.id }}.json`)
- `Content`: Content to upload (expression). Strings are uploaded as-is; objects/arrays are JSON-serialized automatically.
- `Content Type` (optional): MIME type (e.g. `application/json`, `text/plain`). Defaults to `application/octet-stream`.

### Output
- `bucket`, `key`, `size` (bytes), `etag`

---

## hetzner.downloadObject

Download an object from a Hetzner Object Storage bucket.

### When to use
- Fetch a config file stored in object storage during a deployment workflow
- Retrieve a previously uploaded artifact for inspection or processing in a later workflow node

### Expected inputs
- `Bucket`: Source bucket (dropdown or expression)
- `Key`: Object key to download (supports expressions)

### Output
- `bucket`, `key`, `content` (string), `contentType`, `size` (bytes)

---

## hetzner.deleteObject

Delete a specific object from a Hetzner Object Storage bucket.

### When to use
- Clean up old artifacts as part of a post-deployment workflow
- Remove temporary files uploaded during environment setup

### Expected inputs
- `Bucket`: Bucket containing the object (dropdown or expression)
- `Key`: Object key to delete (supports expressions)

### Output
- `bucket`, `key`

---

## hetzner.listObjects

List objects in a Hetzner Object Storage bucket, with optional prefix filtering.

### When to use
- Check whether rollback artifacts exist before triggering a rollback path
- Audit bucket contents as part of a compliance or reporting workflow
- Feed a list of objects into a downstream processing loop

### Expected inputs
- `Bucket`: Bucket to list (dropdown or expression)
- `Prefix` (optional): Filter by key prefix (e.g. `releases/`)
- `Max Keys` (optional): Max objects to return (default: 100, max: 1000)

### Output
- `bucket`, `prefix`, `count`
- `objects`: array of `{ key, size, lastModified, etag }`

---

## hetzner.presignedUrl

Generate a time-limited presigned URL for an object — no credentials needed to access it.

### When to use
- Share a generated report or build artifact via Slack or email after a workflow completes
- Allow an external agent or CI system to upload a file to a specific location without permanent access
- Provide temporary download access to an artifact for a third-party system

### Expected inputs
- `Bucket`: Bucket containing the object (dropdown or expression)
- `Key`: Object key (supports expressions)
- `Method`: `GET` (download) or `PUT` (upload)
- `Expires In` (optional): Expiry in seconds (default: 3600 = 1 hour; max: 604800 = 7 days)

### Output
- `bucket`, `key`, `url` (presigned URL), `expiresAt` (ISO 8601 timestamp)

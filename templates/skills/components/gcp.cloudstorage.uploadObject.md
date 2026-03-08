# GCP Cloud Storage • Upload Object Skill

Use this guidance when planning or configuring `gcp.cloudstorage.uploadObject`.

## Purpose

`gcp.cloudstorage.uploadObject` uploads content as an object to a Google Cloud Storage bucket.

## Required Configuration

- `bucket` (required): The Cloud Storage bucket to upload to. Selected from the integration's available buckets.
- `object` (required): Destination path within the bucket (e.g. `output/results.json`).
- `content` (required): The content to upload.
- `contentType` (optional): MIME type for the uploaded object. Defaults to `application/octet-stream`.

## Planning Rules

When generating workflow operations that include `gcp.cloudstorage.uploadObject`:

1. Always set `configuration.bucket` to a valid bucket name.
2. Always set `configuration.object` to the destination path within the bucket.
3. Always set `configuration.content` to the content to upload.
4. Set `configuration.contentType` to the appropriate MIME type when known (e.g. `application/json`, `text/csv`).
5. `gcp.cloudstorage.uploadObject` emits on the `default` channel.
6. If the object already exists, it will be overwritten.

## Output Fields

- `data.name`: Object name (path within the bucket).
- `data.bucket`: Bucket name.
- `data.size`: Uploaded object size in bytes (string).
- `data.contentType`: MIME type of the uploaded object.
- `data.timeCreated`: Timestamp when the object was created.
- `data.storageClass`: Storage class (e.g. STANDARD).
- `data.md5Hash`: MD5 hash of the uploaded content.
- `data.generation`: Object generation number.
- `data.selfLink`: API URL for the object.

## Configuration Example

```yaml
bucket: "my-output-bucket"
object: "results/workflow-output.json"
content: '{"status": "complete", "count": 42}'
contentType: "application/json"
```

## Accessing Output in Downstream Nodes

- Object name: `{{ $["Upload Object"].data.name }}`
- Object size: `{{ $["Upload Object"].data.size }}`
- Self link: `{{ $["Upload Object"].data.selfLink }}`

## Mistakes To Avoid

- Forgetting to set `content` — it is required even for empty objects.
- Not setting `contentType` when the downstream consumer needs it for correct parsing.
- Connecting from this component with a channel other than `default`.

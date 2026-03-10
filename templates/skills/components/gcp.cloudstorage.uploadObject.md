# GCP Cloud Storage • Upload Object Skill

Use this guidance when planning or configuring `gcp.cloudstorage.uploadObject`.

## Purpose

`gcp.cloudstorage.uploadObject` uploads content to a Google Cloud Storage object, creating or overwriting it.

## Required Configuration

- `bucket` (required): Cloud Storage bucket name. Selected from the integration's available buckets.
- `object` (required): The name (path) of the object to create or overwrite, e.g. `reports/output.json`.
- `content` (required): The content to upload as the object body.
- `contentType` (optional): MIME type for the uploaded object. Defaults to `application/octet-stream`.

## Planning Rules

When generating workflow operations that include `gcp.cloudstorage.uploadObject`:

1. Always set `configuration.bucket` to a valid bucket name from the connected GCP project.
2. Always set `configuration.object` to the desired object path within the bucket.
3. Always set `configuration.content` to the data to upload.
4. Set `configuration.contentType` when the MIME type is known (e.g. `application/json`, `text/plain`).
5. `gcp.cloudstorage.uploadObject` emits on the `default` channel.
6. If the object already exists, it will be overwritten.

## Output Fields

- `data.bucket`: Bucket name.
- `data.name`: Object name (path).
- `data.size`: Object size in bytes (string).
- `data.contentType`: MIME type of the uploaded object.
- `data.selfLink`: URL for the object metadata.

## Configuration Example

```yaml
bucket: "my-data-bucket"
object: "reports/output.json"
content: '{"status": "complete", "count": 42}'
contentType: "application/json"
```

## Accessing Output in Downstream Nodes

- Object name: `{{ $["Upload Object"].data.name }}`
- Object size: `{{ $["Upload Object"].data.size }}`

## Mistakes To Avoid

- Forgetting to set `content` — this field is required.
- Uploading very large files — this action is designed for small to medium content. For large files, use other GCP tools.
- Connecting from this component with a channel other than `default`.

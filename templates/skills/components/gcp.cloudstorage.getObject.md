# GCP Cloud Storage • Get Object Skill

Use this guidance when planning or configuring `gcp.cloudstorage.getObject`.

## Purpose

`gcp.cloudstorage.getObject` retrieves the metadata of an object stored in a Google Cloud Storage bucket.

## Required Configuration

- `bucket` (required): The Cloud Storage bucket containing the object. Selected from the integration's available buckets.
- `object` (required): Full path of the object within the bucket (e.g. `data/report.csv`).

## Planning Rules

When generating workflow operations that include `gcp.cloudstorage.getObject`:

1. Always set `configuration.bucket` to a valid bucket name.
2. Always set `configuration.object` to the full object path within the bucket.
3. `gcp.cloudstorage.getObject` emits on the `default` channel.
4. The output contains object metadata — not the object content itself.
5. This is useful for checking if an object exists, verifying its size, or reading its metadata before downstream processing.

## Output Fields

- `data.name`: Object name (path within the bucket).
- `data.bucket`: Bucket name.
- `data.size`: Object size in bytes (string).
- `data.contentType`: MIME type of the object.
- `data.timeCreated`: Timestamp when the object was created.
- `data.updated`: Timestamp when the object was last updated.
- `data.storageClass`: Storage class (e.g. STANDARD, NEARLINE).
- `data.md5Hash`: MD5 hash of the object content.
- `data.generation`: Object generation number.
- `data.selfLink`: API URL for the object.

## Configuration Example

```yaml
bucket: "my-data-bucket"
object: "reports/2025/q1-summary.csv"
```

## Accessing Output in Downstream Nodes

- Object name: `{{ $["Get Object"].data.name }}`
- Object size: `{{ $["Get Object"].data.size }}`
- Content type: `{{ $["Get Object"].data.contentType }}`

## Mistakes To Avoid

- Forgetting the full object path (e.g. using just `file.json` when the object is at `data/file.json`).
- Expecting object content in the output — this component returns metadata only.
- Connecting from this component with a channel other than `default`.

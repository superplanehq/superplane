# GCP Cloud Storage • Get Object Skill

Use this guidance when planning or configuring `gcp.cloudstorage.getObject`.

## Purpose

`gcp.cloudstorage.getObject` retrieves metadata for a specific object in a Google Cloud Storage bucket.

## Required Configuration

- `bucket` (required): Cloud Storage bucket name. Selected from the integration's available buckets.
- `object` (required): The name (path) of the object to retrieve, e.g. `folder/file.json`.

## Planning Rules

When generating workflow operations that include `gcp.cloudstorage.getObject`:

1. Always set `configuration.bucket` to a valid bucket name from the connected GCP project.
2. Always set `configuration.object` to the full object path within the bucket.
3. `gcp.cloudstorage.getObject` emits on the `default` channel.
4. The output `data` contains the full object metadata from the Cloud Storage JSON API.
5. This action retrieves metadata only — it does not download the object content.

## Output Fields

- `data.bucket`: Bucket name.
- `data.name`: Object name (path).
- `data.size`: Object size in bytes (string).
- `data.contentType`: MIME type of the object.
- `data.updated`: Last modification timestamp.
- `data.md5Hash`: Base64-encoded MD5 hash of the object data.
- `data.selfLink`: URL for the object metadata.

## Configuration Example

```yaml
bucket: "my-data-bucket"
object: "reports/2025/summary.json"
```

## Accessing Output in Downstream Nodes

- Object size: `{{ $["Get Object"].data.size }}`
- Content type: `{{ $["Get Object"].data.contentType }}`
- Last updated: `{{ $["Get Object"].data.updated }}`

## Mistakes To Avoid

- Forgetting to URL-encode special characters in the object name — SuperPlane handles this automatically.
- Expecting the object content in the output — this action returns metadata only.
- Connecting from this component with a channel other than `default`.

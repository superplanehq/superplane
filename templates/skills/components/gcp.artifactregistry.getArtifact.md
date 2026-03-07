# GCP Artifact Registry • Get Artifact Skill

Use this guidance when planning or configuring `gcp.artifactregistry.getArtifact`.

## Purpose

`gcp.artifactregistry.getArtifact` retrieves a Docker image's metadata from a Google Artifact Registry repository. Use this to fetch image details like tags, size, upload time, and URI before downstream processing (e.g., deployment, scanning, notifications).

## Required Configuration

- `location` (required): GCP region of the Artifact Registry repository (e.g. `us-central1`).
- `repository` (required): Artifact Registry repository name.
- `image` (required): Docker image name with digest (e.g. `my-image@sha256:abc123`).

## Planning Rules

When generating workflow operations that include `gcp.artifactregistry.getArtifact`:

1. Always provide all three required fields: `location`, `repository`, and `image`.
2. The `image` field must include a digest (`@sha256:...`).
3. This component emits on the `default` channel.
4. The output contains the full DockerImage resource from the Artifact Registry API.

## Output Fields

- `data.name`: Full resource name of the Docker image.
- `data.uri`: URI to access the image (e.g. `us-central1-docker.pkg.dev/project/repo/image@sha256:...`).
- `data.tags`: Array of tags attached to the image.
- `data.imageSizeBytes`: Image size in bytes.
- `data.uploadTime`: When the image was uploaded.
- `data.mediaType`: Image media type.
- `data.buildTime`: When the image was built.
- `data.updateTime`: When the image was last updated.

## Accessing Output in Downstream Nodes

- Image URI: `{{ $["Get Artifact"].data.uri }}`
- Image tags: `{{ $["Get Artifact"].data.tags }}`
- Upload time: `{{ $["Get Artifact"].data.uploadTime }}`

## Mistakes To Avoid

- Omitting the digest from the `image` field.
- Using a repository path instead of just the repository name.

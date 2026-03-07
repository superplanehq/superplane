# GCP Artifact Registry • Get Artifact Analysis Skill

Use this guidance when planning or configuring `gcp.artifactregistry.getArtifactAnalysis`.

## Purpose

`gcp.artifactregistry.getArtifactAnalysis` retrieves existing vulnerability analysis results for a container image from Container Analysis (Artifact Analysis). Unlike `analyzeArtifact`, this is a one-shot retrieval that returns immediately with whatever results are currently available.

## Required Configuration

- `resourceUrl` (required): Full resource URL of the container image in the format `https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/IMAGE@sha256:DIGEST`.

## Planning Rules

When generating workflow operations that include `gcp.artifactregistry.getArtifactAnalysis`:

1. Always provide the full `resourceUrl` including the `https://` prefix and the `@sha256:` digest.
2. This component emits on the `default` channel.
3. Returns immediately with current results (may be empty if scanning hasn't completed).
4. Use `analyzeArtifact` instead if you need to wait for scan completion.

## Output Fields

- `data.occurrences`: Array of vulnerability occurrences found for the image (may be empty).
- Each occurrence includes `kind`, `severity`, `vulnerability.packageIssue`, and `resourceUri`.

## Accessing Output in Downstream Nodes

- Occurrences: `{{ $["Get Artifact Analysis"].data.occurrences }}`

## Mistakes To Avoid

- Omitting the `https://` prefix from the resource URL.
- Expecting results immediately after an image push — use `analyzeArtifact` to wait for results.

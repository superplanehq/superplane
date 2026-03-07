# GCP Artifact Registry • Analyze Artifact Skill

Use this guidance when planning or configuring `gcp.artifactregistry.analyzeArtifact`.

## Purpose

`gcp.artifactregistry.analyzeArtifact` queries Container Analysis for vulnerability occurrences on a container image and waits until scan results are available. Use this when you need to ensure an image has been analyzed before proceeding (e.g., security gates, compliance checks).

## Required Configuration

- `resourceUrl` (required): Full resource URL of the container image in the format `https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/IMAGE@sha256:DIGEST`.

## Planning Rules

When generating workflow operations that include `gcp.artifactregistry.analyzeArtifact`:

1. Always provide the full `resourceUrl` including the `https://` prefix and the `@sha256:` digest.
2. This component emits on the `default` channel.
3. If no vulnerability occurrences are found yet, the component polls every 30 seconds until results appear.
4. The Container Analysis API and automatic scanning must be enabled in the project.

## Output Fields

- `data.occurrences`: Array of vulnerability occurrences found for the image.
- Each occurrence includes `kind`, `severity`, `vulnerability.packageIssue`, and `resourceUri`.

## Accessing Output in Downstream Nodes

- Occurrences: `{{ $["Analyze Artifact"].data.occurrences }}`

## Mistakes To Avoid

- Omitting the `https://` prefix from the resource URL.
- Using this component without enabling Container Analysis automatic scanning in the GCP project.

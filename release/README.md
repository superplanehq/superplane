# SuperPlane Release Scripts

This directory contains scripts for building and publishing SuperPlane releases.

## Prerequisites

### Syft (SBOM Generation)

The release process generates a Software Bill of Materials (SBOM) using [Syft](https://github.com/anchore/syft). You need to have Syft installed to create releases.

Install Syft:

```bash
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
```

Verify installation:

```bash
syft --version
```

## Release Process

### 1. Create a Tag

Create a new version tag (patch, minor, or major):

```bash
./release/create_tag.sh patch  # For bug fixes (0.0.x)
./release/create_tag.sh minor  # For new features (0.x.0)
```

### 2. Build the Docker Image

```bash
# Build and tag the image
make image.build IMAGE_TAG=vX.Y.Z

# Push to registry
make image.push IMAGE_TAG=vX.Y.Z
```

### 3. Build Release Artifacts

Build the self-hosted tarball (includes SBOM generation):

```bash
./release/superplane-single-host-tarball/build.sh vX.Y.Z
```

This script will:
- Create the self-hosted tarball with docker-compose configuration
- Generate an SBOM (Software Bill of Materials) in SPDX JSON format

### 4. Create GitHub Release

Create the GitHub release and upload artifacts:

```bash
GITHUB_TOKEN=your_token node release/create-github-release.js vX.Y.Z
```

This will upload:
- `superplane-single-host.tar.gz` - Self-hosted installation package
- `superplane-sbom.json` - Software Bill of Materials (SBOM)
- CLI binaries (if available in `release/cli/`)

## SBOM Format

The SBOM is generated in SPDX 2.3 JSON format and includes:
- All packages and dependencies in the Docker image
- Package versions and licenses
- File metadata and checksums
- Relationship information between components

This allows users to:
- Audit dependencies and licenses
- Track vulnerabilities
- Ensure supply chain security
- Meet compliance requirements

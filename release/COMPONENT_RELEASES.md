# Component release configuration

Runner and Fleet Manager releases use two manual promotions. Run a preparation
promotion from a passed `main` workflow. Inspect its workflow artifacts. Then,
run the protected publish promotion.

## Semaphore deployment targets

The Semaphore project contains these deployment targets:

- `runner-releases`
- `fleet-manager-releases`

Both targets allow role `5c7e90e1-f923-4578-883c-cd062cab249e` on the exact
`main` branch. They do not allow tags, pull requests, or all refs. Both targets
record `VERSION` as a deployment bookmark.

Enter all protected values in the Semaphore UI. Do not pass a value to `sem`
and do not export a target to this repository.

Add these values to both targets:

- `GITHUB_RELEASE_APP_ID`: The repository-scoped release app ID.
- `GITHUB_RELEASE_APP_INSTALLATION_ID`: The app installation ID.
- `/home/semaphore/github-release-app.pem`: The app private key as a protected
  file.

Give the GitHub App `Metadata: read` and `Contents: write` permissions only.
Install it only on `superplanehq/superplane`.

The Fleet Manager publish pipeline uses the existing `ghcr-push-token` project
secret. The secret provides `GHCR_PUSH_TOKEN`. Do not add GHCR credentials to
the `fleet-manager-releases` deployment target. Remove `GHCR_USERNAME` and
`GHCR_TOKEN` from that target if they are present.

Runner version releases publish only to GitHub Releases. SHA development
artifacts can continue to use their existing storage path. Do not add AWS
credentials to the `runner-releases` deployment target. Remove
`RUNNER_RELEASE_AWS_ROLE_ARN` from that target if it is present.

## GitHub configuration

Before the first component release:

1. Add an active tag ruleset for `refs/tags/runner/v*` and
   `refs/tags/fleet-manager/v*`.
2. Restrict tag creation, update, and deletion.
3. Give bypass access only to the release GitHub App.
4. Enable immutable releases for the repository.

The component release script creates a draft, uploads all assets, and then
publishes the release. This order is compatible with immutable releases.

Stable releases include notes that `git-cliff` generates from `cliff.toml`.
The preparation pipeline limits the notes to the selected component and stores
their checksum in the release manifest. Release candidates do not include
generated notes. A stable component release becomes the latest release for the
repository. A release candidate does not become the latest release. GitHub has
one latest release for the repository, not one latest release per component.

## First release

Use a prerelease such as `v0.1.0-rc.1`. Start the applicable preparation
promotion from a passed `main` workflow. Confirm the manifest source SHA,
asset names, and checksums. Then, start the publish promotion.

Confirm the following results:

- The component tag points to the selected source SHA.
- GitHub marks the release as immutable.
- GitHub contains all Runner assets from the preparation manifest.
- The Fleet Manager tag resolves to the digest in the GitHub release body.
- The Fleet Manager image contains `linux/amd64` and `linux/arm64` images.

To roll back, select an older immutable Runner version or Fleet Manager image
digest. Do not rebuild or republish an existing version.

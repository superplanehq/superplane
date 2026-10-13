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

Add `RUNNER_RELEASE_AWS_ROLE_ARN` to `runner-releases`. This value is the ARN of
the role that trusts Semaphore OIDC.

Add `GHCR_USERNAME` and `GHCR_TOKEN` to `fleet-manager-releases`. Limit this
token to `write:packages`. Do not grant `delete:packages`. Limit package access
to `fleet-manager`.

## AWS configuration

Create an AWS OIDC provider for `https://superplanehq.semaphoreci.com`. Use the
same URL as the audience. Create the Runner release role with these templates:

- [OIDC trust policy](runner/aws/oidc-trust-policy.json)
- [role policy](runner/aws/role-policy.json)

Replace `<AWS_ACCOUNT_ID>` before you apply the trust policy. Do not add list,
delete, ACL, or bucket administration permissions.

Merge the statements in
[bucket-policy-statements.json](runner/aws/bucket-policy-statements.json) into
the existing `superplanehq-releases` bucket policy. Keep the existing public
`s3:GetObject` access. Do not add public `s3:ListBucket` access.

Enable bucket versioning. Configure an AWS CloudTrail data event selector for
write events on `arn:aws:s3:::superplanehq-releases/runner/`. Send the trail to
the central audit log destination. Confirm that the destination has retention
and access controls that meet the organization policy.

## GitHub configuration

Before the first component release:

1. Add an active tag ruleset for `refs/tags/runner/v*` and
   `refs/tags/fleet-manager/v*`.
2. Restrict tag creation, update, and deletion.
3. Give bypass access only to the release GitHub App.
4. Enable immutable releases for the repository.

The application release script creates a draft, uploads all assets, and then
publishes the release. This order is compatible with immutable releases.

## First release

Use a prerelease such as `v0.1.0-rc.1`. Start the applicable preparation
promotion from a passed `main` workflow. Confirm the manifest source SHA,
asset names, and checksums. Then, start the publish promotion.

Confirm the following results:

- The component tag points to the selected source SHA.
- GitHub marks the release as immutable.
- Runner objects and GitHub Runner assets have identical checksums.
- The Fleet Manager tag resolves to the digest in the GitHub release body.
- The Fleet Manager image contains `linux/amd64` and `linux/arm64` images.

To roll back, select an older immutable Runner version or Fleet Manager image
digest. Do not rebuild or republish an existing version.

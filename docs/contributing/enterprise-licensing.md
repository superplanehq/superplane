# Enterprise licensing

SuperPlane is one binary. An installation runs in Community mode until an
installation administrator installs a valid Enterprise license. The license
enables Enterprise features at runtime. SuperPlane verifies the license
offline. It downloads signed updates to its trusted keys and keeps working
without them.

## License sources

1. When `SUPERPLANE_LICENSE_PATH` is set, the file is the only source.
   Administrators cannot change the license in the UI. In Helm, set
   `license.secretName` to mount a license from a Kubernetes secret.
2. Otherwise, the license that an administrator installs in the UI is the
   source.

## Installing a license

Installation administrators manage the license on the **Installation Admin >
License** page. During owner setup, Community is the default. The owner adds
an Enterprise license only if they have one, and can add it later. SuperPlane
accepts only a license that is valid now. To renew a license or add features,
install the new license.

Installation administrators see a banner when the license is expired or expires
within 30 days. `SUPERPLANE_LICENSE_HIDE_EXPIRY_BANNER` hides that banner when
it is `yes` or `true`. The default is `no`. Hosted SuperPlane sets it to `yes`.

When the installation has one organization, that organization receives every
licensed feature. When it has more than one organization, open each
organization in Installation Admin and enable the features there. A new
organization does not receive grants.

The license page asks for confirmation before an install or a removal that
removes grants. It shows how many organizations use each feature and the
activation history.

## Organization grants

A feature is available in an organization only when all of these are true:

1. The license state is active.
2. The license contains the feature.
3. The organization has a grant for the feature, or the license is the
   development license.

A nil organization ID is not entitled. A grant lookup error denies access.
The license check stays in the formula, so access stops when the license is
no longer active even if grant removal is late.

| Open activation | Loaded state | Same license | Result |
| --- | --- | --- | --- |
| None | Active | Not applicable | Activate. Prune grants the license does not contain. If exactly one organization exists, grant it every licensed feature. |
| None | Any other state | Not applicable | No change |
| Yes | Active | Yes | No change. A removed grant stays removed. |
| Yes | Active | No | Replace. Keep grants for features the new license contains. Prune the rest. One organization also receives new features. |
| Yes | Expired | Any | Deactivate. Remove all grants. |
| Yes | Revoked | Any | Deactivate. Remove all grants. |
| Yes | None | Not applicable | Deactivate. Remove all grants. |
| Yes | Invalid | Any | No change. Access stays denied by the license check. |
| Yes | Not yet valid | Any | No change. Access stays denied until the license is active. |
| Any | Load error | Not applicable | No change. The handler does not run. |

Expiry, revocation, and removal delete all grants. There is no grace period.
Renew the license before it expires to keep the organization settings. A
renewal before expiry keeps grants for features that the new license still
contains. An invalid license or a failed license load does not remove grants.

Installation administrators manage grants on **Installation Admin >
Organizations > [organization] > Enterprise features**. Members of an
organization see only that a feature is not available. They do not see
license details.

## Enterprise code

Code in the `ee/` directory is subject to the SuperPlane Enterprise Edition
License in `ee/LICENSE`. All other code is licensed under the Apache License,
Version 2.0. To add an Enterprise feature, follow
[`ee/README.md`](../../ee/README.md).

Billing source in `ee/billing` uses the Enterprise source license. Billing
does not require an installation license or a runtime license feature. The
server always installs billing, including when billing enforcement is disabled.
Usage costs remain recorded in that mode.

The operations below stay the same. The check is now per organization. A
grant in one organization does not enable the feature in another.

The Enterprise features are:

| Feature             | Operations that require the license                                                                                          |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `custom_roles`      | Create or update a custom role. Assign a custom role.                                                                        |
| `groups`            | Create or update a group. Add a user to a group.                                                                             |
| `github_enterprise` | Create or configure a GitHub integration with a host other than github.com. Create or install a private GitHub App on that host. |

GitHub Enterprise Cloud also needs the organization experimental feature
`github_enterprise` during rollout. An installation administrator turns that
feature on for each organization. github.com connections do not use the license
or the experimental feature.

When a license expires, SuperPlane removes all organization grants. Existing custom roles and groups continue to work,
and administrators can still remove them. Existing GitHub Enterprise
integrations continue to run: webhooks, API calls, and repository lists
keep working, and administrators can still view and delete them. Turning the
experimental feature off only stops new connections.

## Feature registry

`licensing.Service` answers whether an organization can use a feature such
as `custom_roles` or `groups`. `Status.IsEntitled` answers whether the
active license contains the feature. `enterprise.Registry` answers which
code runs that feature. The checks stay separate. The component registry in
`pkg/registry/registry.go` uses the same lookup: register an implementation
under a name, then ask for that name and receive the interface.

```go
rbac, err := enterprise.Get[enterprise.Rbac](registry, enterprise.RBAC)
```

Go has no generic methods, so `Get` is a function. A missing key or a value of
the wrong type is an error. `NewRegistry` registers a Community `Rbac` that
refuses every operation, so a server with no Enterprise registration fails
closed.

One registry key can cover more than one license feature. `custom_roles` and
`groups` are two license features and one `Rbac` implementation. A new feature
registers a new interface under a new key. It does not add methods to `Rbac`.

`pkg/server` is the only registration site, because it is the only core
package that may import `ee/`. The steps for adding a feature are in
[`ee/README.md`](../../ee/README.md).

## Local development

`make dev.server` uses a development license when no license is installed.
That license allows every Enterprise feature in every organization and does
not write grants. A real license installed in development uses the normal
grant rules. To test Community mode, set
`SUPERPLANE_LICENSE_DEV_ENTERPRISE=false` in `.env` and restart the server.

Tests sign licenses with `pkg/licensing/licensingtest`. Do not commit private
keys or license files.

## Trusted keys

SuperPlane verifies a license offline with the signing keys it already trusts.
It learns those keys from a signed list. The trust anchor for that list is
`pkg/licensing/trustedkeys/root.jwks.json`, which ships in the binary. A
downloaded list is accepted only when one of those public keys signed it and
the list version is newer than the list already trusted. The accepted list is
stored in the database. A failed download leaves that list in place.

The default list URL is `https://licensing.superplane.com/.well-known/license-keys.jws`.
`SUPERPLANE_LICENSE_KEYS_URL` selects another `http` or `https` URL, including
a local issuer. Set it to `none` to turn downloads off. A plain HTTP URL is
accepted because the list must still verify with an embedded root. An
installation administrator can upload a signed list when the installation
cannot download one.

`pkg/licensing/trustedkeys/license-keys.jws` is the list shipped with a
release. After a reviewed change to the trust anchor, run
`make license.keys.update` so the shipped list verifies with that anchor.

This is the same shape as a [trust anchor](https://datatracker.ietf.org/doc/html/rfc5280#section-6.1.1)
and the [TUF root role](https://theupdateframework.github.io/specification/latest/#the-root-role):
the application ships the key that verifies later key material. Creating,
publishing, and rotating that key are issuer operations. They are documented
in the licensing repository.

A failed download, or a list signed by a root this binary does not contain,
leaves the trusted list in place. Licenses signed by keys in that list keep
verifying. This installation learns a new signing key only from a later list
that one of its embedded roots signed. Replacing the root therefore requires
a release that embeds the new public key. Restoring the issuer key and
choosing that release are issuer operations. They are documented in the
licensing repository.

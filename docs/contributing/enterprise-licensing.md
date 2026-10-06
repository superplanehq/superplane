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

## Enterprise code

Code in the `ee/` directory is subject to the SuperPlane Enterprise Edition
License in `ee/LICENSE`. All other code is licensed under the Apache License,
Version 2.0. To add an Enterprise feature, follow
[`ee/README.md`](../../ee/README.md).

The Enterprise features are:

| Feature        | Operations that require the license                    |
| -------------- | ------------------------------------------------------ |
| `custom_roles` | Create or update a custom role. Assign a custom role.  |
| `groups`       | Create or update a group. Add a user to a group.       |

When a license expires, existing custom roles and groups continue to work,
and administrators can still remove them.

## Local development

`make dev.server` enables every Enterprise feature when no license is
installed. To test Community mode, set `SUPERPLANE_LICENSE_DEV_ENTERPRISE=false`
in `.env` and restart the server.

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
`SUPERPLANE_LICENSE_KEYS_URL` selects another `https` URL. Set it to `none` to
turn downloads off. An installation administrator can upload a signed list when
the installation cannot download one.

`pkg/licensing/trustedkeys/license-keys.jws` is the list shipped with a
release. After a reviewed change to the trust anchor, run
`make license.keys.update` so the shipped list verifies with that anchor.

This is the same shape as a [trust anchor](https://datatracker.ietf.org/doc/html/rfc5280#section-6.1.1)
and the [TUF root role](https://theupdateframework.github.io/specification/latest/#the-root-role):
the application ships the key that verifies later key material. Creating,
publishing, and rotating that key are issuer operations. They are documented
in the licensing repository.

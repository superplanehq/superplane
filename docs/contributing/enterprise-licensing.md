# Enterprise licensing

SuperPlane is one binary. An installation runs in Community mode until an
installation administrator installs a valid Enterprise license. The license
enables Enterprise features at runtime. SuperPlane verifies the license
offline. It never contacts the issuer.

## License sources

1. When `SUPERPLANE_LICENSE_PATH` is set, the file is the only source.
   Administrators cannot change the license in the UI. In Helm, set
   `license.secretName` to mount a license from a Kubernetes secret.
2. Otherwise, the license that an administrator installs in the UI is the
   source.

## Local development

`make dev.server` enables every Enterprise feature when no license is
installed. To test Community mode, set `SUPERPLANE_LICENSE_DEV_ENTERPRISE=false`
in `.env` and restart the server.

Tests sign licenses with `pkg/licensing/licensingtest`. Do not commit private
keys or license files.

## Trusted keys

`pkg/licensing/trustedkeys/production.jwks.json` contains the public keys that
SuperPlane trusts. Change it only in a reviewed pull request, and run
`make check.license.keys` before a release. Never reuse a key ID.

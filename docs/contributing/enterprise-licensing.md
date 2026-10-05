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

## Local development

`make dev.server` enables every Enterprise feature when no license is
installed. To test Community mode, set `SUPERPLANE_LICENSE_DEV_ENTERPRISE=false`
in `.env` and restart the server.

Tests sign licenses with `pkg/licensing/licensingtest`. Do not commit private
keys or license files.

## Trusted keys

`pkg/licensing/trustedkeys/root.jwks.json` contains the root public keys. They
sign the list of license signing keys, which SuperPlane downloads every six
hours and when a license uses an unknown key. Set
`SUPERPLANE_LICENSE_KEYS_URL=none` to turn downloads off.

`pkg/licensing/trustedkeys/license-keys.jws` is the list at release time. Run
`make license.keys.update` before a release. Change root keys only in a
reviewed pull request.

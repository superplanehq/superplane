# Enterprise licensing

SuperPlane is one binary. An installation runs in Community mode until an
installation administrator installs a valid Enterprise license. The license
enables Enterprise features at runtime. There is no separate Enterprise build.

## How verification works

- `pkg/licensing` verifies the license fully offline. It never contacts the
  issuer.
- The license is a compact JWS signed with ES256. The verifier accepts only the
  `alg`, `kid`, and `typ` header parameters.
- The verifier trusts only the reviewed keys in
  `pkg/licensing/trustedkeys/production.jwks.json`. The server binary embeds
  that file.
- The issuer is `https://licensing.superplane.com` and the audience is
  `superplane-self-hosted`.
- The clock tolerance is five minutes.
- Every verification failure results in Community mode with a safe reason
  category. SuperPlane never logs or returns the raw license.

## License sources

SuperPlane uses one authoritative source:

1. When `SUPERPLANE_LICENSE_PATH` is set, the file is the only source. A
   missing or invalid file results in Community mode. SuperPlane does not fall
   back to the database. Administrators cannot change the license in the UI.
2. Otherwise, the license that an administrator installs in the UI is the
   source. SuperPlane encrypts it at rest with the installation encryptor.

Every replica reloads the license every 30 seconds.

In Helm, set `license.secretName` to mount a license from a Kubernetes secret.

## Local development

`make dev.server` builds with the `licensedev` tag. Release images never use
the tag, and release binaries do not contain the development license code.

### All Enterprise features in development

The development server grants every Enterprise feature when no license is
installed. `docker-compose.dev.yml` sets `SUPERPLANE_LICENSE_DEV_ENTERPRISE`
to `true`. At startup, the server makes a signing key that exists only in
memory, trusts it, and signs a license for all recognized features. The
License page shows this license as installed.

- An installed license has priority. When you remove it, the development
  license applies again.
- When `SUPERPLANE_LICENSE_PATH` is set, the file stays the only source and
  the development license does not apply.
- To test Community mode, set `SUPERPLANE_LICENSE_DEV_ENTERPRISE=false` in
  `.env` and restart `make dev.server`.

### Licenses from a local issuer

Only the `licensedev` tag reads `SUPERPLANE_LICENSE_DEV_JWKS_PATH`, which adds
the keys of a local licensing issuer. Use it to test a real issued license.

1. Start the licensing repository and activate its local signing key.
2. Export the local keys:

   ```text
   curl -s http://localhost:8100/.well-known/jwks.json > tmp/license-dev.jwks.json
   ```

3. Set `SUPERPLANE_LICENSE_DEV_JWKS_PATH=/app/tmp/license-dev.jwks.json` in
   `.env` and restart `make dev.server`.
4. Issue a license in the local issuer and install it in SuperPlane.

Tests sign licenses with ephemeral keys from `pkg/licensing/licensingtest`.
Do not commit private keys.

## Signing-key rotation

1. The issuer registers a new key. It appears in the issuer JWKS as pending.
2. Add the new public key to `production.jwks.json` in a reviewed pull
   request. Keep all keys that can still verify an unexpired license.
3. Run `make check.license.keys`. It compares the bundled keys with the live
   issuer JWKS. CI does not run it because it uses the network.
4. Release SuperPlane with the old and new keys.
5. Activate the new key in the issuer only after that release is available.
6. Remove the old key only after every license that it signed has expired.

Never reuse a key ID.

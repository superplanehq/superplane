# Fleet Manager

This process reconciles SuperPlane runner demand with configured infrastructure
providers. A fleet can use AWS or Docker. Fleet Manager only uses the
installation admin HTTP API. Set `FLEET_MANAGER_CONFIG_FILE` to the JSON
configuration path. The default is
`/etc/superplane/fleet-manager.json`.

Use a personal API token that belongs to an installation administrator. The
Fleet Manager sends it as an HTTP bearer token. Runner instances receive only
their short-lived registration token.

AWS fleets require a release manifest URL that contains `{version}`. A
manifest has this shape:

```json
{
  "version": "1.2.3",
  "protocol_version": "runner/v1",
  "source_repository": "https://github.com/superplanehq/superplane",
  "source_commit": "<git commit SHA>",
  "artifacts": [
    {
      "operating_system": "linux",
      "architecture": "amd64",
      "url": "https://downloads.example/runner/v1.2.3/runner-linux-amd64",
      "sha256": "<hex SHA-256>",
      "signature": "<base64 Ed25519 signature of the 32-byte SHA-256 digest>"
    }
  ]
}
```

The trusted Ed25519 public key is a base64-encoded 32-byte key. Fleet Manager
downloads and verifies each exact artifact once. AWS bootstrap downloads the
same immutable URL and verifies its SHA-256 before installation.

Docker fleets use a configured runner image instead of a release artifact.
The local development configuration uses the tool-rich
`superplane-runner-local:dev` image and starts one ephemeral container for
each task.

Set `task_specific` to `true` to create one runner for each queued task.
Fleet Manager uses a stable task idempotency key so it can resume provisioning
after a restart. `warm_capacity` adds unbound runners.

# Fleet Manager

This process reconciles SuperPlane runner demand with configured infrastructure
providers. A fleet can use AWS or Docker. Fleet Manager only uses the
installation admin HTTP API. Set `FLEET_MANAGER_CONFIG_FILE` to the JSON
configuration path. The default is
`/etc/superplane/fleet-manager.json`.

Use a personal API token that belongs to an installation administrator. The
Fleet Manager sends it as an HTTP bearer token. Runner instances receive only
their short-lived registration token.

AWS fleets require `runner_release_base_url`. Fleet Manager selects a release
from this layout:

```text
<runner_release_base_url>/<version>/runner-linux-amd64.tar.gz
<runner_release_base_url>/<version>/runner-linux-arm64.tar.gz
<runner_release_base_url>/<version>/checksums.txt
```

Each archive contains the runner binary and `install.sh`. Fleet Manager reads
the selected archive checksum from `checksums.txt`. AWS bootstrap downloads
the archive, verifies its SHA-256, extracts it, and runs the bundled installer.

Docker fleets use a configured runner image instead of a release artifact.
The local development configuration uses the tool-rich
`superplane-runner-local:dev` image and starts one ephemeral container for
each task.

Set `task_specific` to `true` to create one runner for each queued task.
Fleet Manager uses a stable task idempotency key so it can resume provisioning
after a restart. `warm_capacity` adds unbound runners.

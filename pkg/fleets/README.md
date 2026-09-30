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
`superplane-runner-local:dev` image.

Fleet Manager creates unbound ephemeral runners. An idle runner can reserve the
next queued task. The target capacity is the number of queued tasks plus
`warm_capacity`. Set `warm_capacity` to the number of idle runners that the
fleet must keep when no tasks are queued.

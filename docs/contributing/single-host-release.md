# Single-host release

Every SuperPlane release includes `superplane-single-host.tar.gz`. The archive
installs SuperPlane on one VM with Docker Compose.

`release/superplane/build-assets.sh` resolves the latest stable Runner and
Fleet Manager tags before it builds the archive. It writes those tags into
`release/superplane-helm-chart/helm/values.yaml`. If GitHub is unreachable,
the committed values stay in place. The Helm chart in that same release uses
those tags. It does not call GitHub again. A standalone chart build still
resolves the latest stable tags.

`release/superplane-single-host-tarball/build.sh` reads `runnerVersion` and
`fleetManagerVersion` from that file and writes them into the Compose file
and `install.sh`.

The archive contains:

- `docker-compose.yml` for the app, Fleet Manager, Postgres, RabbitMQ, and Caddy
- `install.sh`, which keeps existing secrets when you run it again
- `fleet-manager.yaml`, which configures the local Docker runner fleet
- `Caddyfile` for automatic HTTPS

The installer starts Fleet Manager automatically. Set
`SUPERPLANE_RUNNER_MAX_CAPACITY` and `SUPERPLANE_RUNNER_WARM_CAPACITY` when
you run `install.sh` to change the default capacities of five and one.

Fleet Manager and the runner image are separate release artifacts:

- `ghcr.io/superplanehq/fleet-manager:<version>-docker` includes the Docker CLI
- `ghcr.io/superplanehq/runner:<version>` is the tool image for runner containers

The release pipeline publishes the archive. It does not boot it. Boot the
archive on a machine before you rely on a new release.

To test the archive without starting containers:

```bash
bash release/superplane-single-host-tarball/test.sh
```

To boot a local install from an image you already built:

```bash
bash release/superplane-single-host-tarball/build.sh v9.9.9
bash release/superplane-single-host-tarball/smoke-test.sh \
  build/superplane-single-host-tarball-v9.9.9/superplane-single-host.tar.gz \
  ghcr.io/superplanehq/superplane:v9.9.9
```

Set `SUPERPLANE_SMOKE_KEEP=yes` to keep that install running.

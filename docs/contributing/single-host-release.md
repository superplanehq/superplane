# Single-host release

Every SuperPlane release includes `superplane-single-host.tar.gz`. The archive
installs SuperPlane on one VM with Docker Compose.

`release/superplane-single-host-tarball/build.sh` builds the archive. It reads
`runnerVersion` and `fleetManagerVersion` from
`release/superplane-helm-chart/helm/values.yaml` and writes those versions into
the Compose file and `install.sh`.

The archive contains:

- `docker-compose.yml` for the app, Postgres, RabbitMQ, and Caddy
- `install.sh`, which keeps existing secrets when you run it again
- `enable-runners.sh`, which starts Fleet Manager with the Docker provider
- `Caddyfile` for automatic HTTPS

Fleet Manager and the runner image are separate release artifacts:

- `ghcr.io/superplanehq/fleet-manager:<version>-docker` includes the Docker CLI
- `ghcr.io/superplanehq/runner:<version>` is the tool image for runner containers

`.semaphore/prepare-superplane-release.yml` runs
`release/superplane-single-host-tarball/smoke-test.sh` before it promotes the
release. The smoke test installs the tarball with the release image and checks
health, secret reuse, and restart.

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

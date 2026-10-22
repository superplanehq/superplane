# Coolify

These Compose files run SuperPlane on one Coolify VM: the app, PostgreSQL, RabbitMQ, and one Docker runner.

Set `ENCRYPTION_MASTER_KEYS` once. Keep the value. If you lose it, encrypted integration secrets are lost.

```bash
echo "env://coolify-v1:$(openssl rand -base64 32)"
```

On an ARM VM, set `RUNNER_FLEET_ID=e1-large-arm64` and `RUNNER_ARCH=arm64`.

The runner network uses the Compose project name. Two stacks on one Docker host stay isolated. Set `RUNNER_NETWORK` if you override the project name.

## Git / build from source

Use [docker-compose.source.yml](docker-compose.source.yml) when Coolify clones this repository.

1. Create a Docker Compose application from the Git repo.
2. Set the base directory to `/`.
3. Set the Compose path to `deploy/coolify/docker-compose.source.yml`.
4. Set `SUPERPLANE_VERSION` to a published `ghcr.io/superplanehq/superplane` tag.

Coolify builds the runner and Fleet Manager from `release/`. Raise the build timeout. Fleet Manager creates an admin token after you complete owner setup.

## Paste / published images

Use [docker-compose.image.yml](docker-compose.image.yml) when you do not want a Git checkout.

1. Create a Docker Compose Empty resource in Coolify.
2. Paste the image Compose file.
3. Set `SUPERPLANE_VERSION`.
4. Set `ENCRYPTION_MASTER_KEYS`.
5. Set `FLEET_MANAGER_VERSION` to a Fleet Manager release that publishes the `-docker` image.
6. Deploy and complete owner setup.
7. Create an installation-admin personal API token.
8. Set `INSTALLATION_ADMIN_TOKEN` and restart Fleet Manager.

`RUNNER_VERSION` defaults to `v0.2.0`. `fleet-manager/v0.1.4` does not publish the `-docker` image.

The published Fleet Manager image cannot mint a token from PostgreSQL. Until you set `INSTALLATION_ADMIN_TOKEN`, that container exits and Coolify starts it again.

## Coolify one-click catalog

The image Compose file includes Coolify service metadata. Coolify accepts one-click services from repositories with at least 1,000 GitHub stars.

After that threshold, open a pull request against Coolify `next`:

- Add `templates/compose/superplane.yaml` from `docker-compose.image.yml`.
- Add `svgs/superplane.svg` from [superplane.svg](superplane.svg).
- Add the matching Coolify docs page.

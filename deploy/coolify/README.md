# Coolify

These Compose files run SuperPlane on one Coolify VM: the app, PostgreSQL,
RabbitMQ, Fleet Manager, and one Docker runner.

## Domain

Attach the public URL to the **app** service on port **8000**.

Coolify reads `SERVICE_URL_APP_8000` and routes HTTPS to `app:8000`.
That generated URL includes `:8000` as a proxy hint. Do not use that string as
the public site URL. The Compose file strips `:8000` from HTTPS `BASE_URL` and
`WEBHOOKS_BASE_URL` so GitHub, OpenRouter, and other callbacks use
`https://your-domain` with no port.

If you add a custom domain:

1. Open **Domains**.
2. Select service `app`.
3. Set the internal port to `8000`.
4. Enable HTTPS.

Do not attach the domain to `fleet-manager`, `db`, or `rabbitmq`.

If the domain does not respond, wait until `app` is healthy.
Then confirm the domain uses port `8000`.
If Coolify shows **No Available Server**, the domain is on the wrong service
or the wrong port.

If an OAuth callback still includes `:8000`, paste the updated Compose file and
restart `app`. Then set the provider callback to `https://your-domain` with no
port.

## Encryption key

Coolify generates `SERVICE_REALBASE64_32_ENCRYPTION` (32 random bytes, Base64).
The Compose file builds:

```text
ENCRYPTION_MASTER_KEYS=env://coolify-v1:<generated-value>
```

Leave that value. Do not replace it with a password.
SuperPlane rejects a raw password with `unsupported encryption master key entry`.

Keep `ENCRYPTION_MASTER_KEYS`. If you lose it, encrypted integration secrets
are lost.

If an earlier deploy stored a bad key, delete `ENCRYPTION_MASTER_KEYS` in
**Environment Variables**. Redeploy so Coolify can generate a valid key.

If you must set the key yourself:

```bash
echo "env://coolify-v1:$(openssl rand -base64 32)"
```

## Fleet Manager

Complete owner setup in the SuperPlane UI. Do not create a personal API token
for the default path.

### Published images (`template/superplane.yaml`)

1. Deploy the stack and wait until `app` is healthy.
2. Open the SuperPlane URL and complete owner setup.
3. Wait. `fleet-token` writes a token after the owner account exists.
4. `fleet-manager` reads that token and starts.

Set `INSTALLATION_ADMIN_TOKEN` only when you replace that generated token.
`fleet-token` then skips minting. After a password change, redeploy so
`fleet-token` can replace a revoked token.

### Git / source (`docker-compose.source.yml`)

Fleet Manager waits for the owner account, then mints the token itself.

## Git / build from source

Use [docker-compose.source.yml](docker-compose.source.yml) when Coolify clones
this repository.

1. Create a Docker Compose application from the Git repo.
2. Set the base directory to `/`.
3. Set the Compose path to `deploy/coolify/docker-compose.source.yml`.
4. Leave `SUPERPLANE_VERSION` at `v1.0.0`, or set a published GHCR tag.
5. Attach the domain to `app` on port `8000`.
6. Deploy. Complete owner setup.

Coolify builds the runner and Fleet Manager from `release/`. Raise the build
timeout.

## Paste / published images

Use [template/superplane.yaml](template/superplane.yaml) when you do not want a
Git checkout. This is also the Coolify one-click catalog file.

1. Create a Docker Compose Empty resource in Coolify.
2. Paste `template/superplane.yaml`.
3. Leave `SUPERPLANE_VERSION` at `v1.0.0`, or set a published GHCR tag.
4. Leave the encryption variables as Coolify generated them.
5. Attach the domain to `app` on port `8000`.
6. Deploy. Complete owner setup. Wait for Fleet Manager.

`SUPERPLANE_VERSION` defaults to `v1.0.0`.
`FLEET_MANAGER_VERSION` defaults to `v0.1.6`.
`RUNNER_VERSION` defaults to `v0.2.1`.

## Runner network and architecture

The runner network and Fleet Manager ID use the Compose project name. The
source stack also tags the local runner image with that name. Two stacks on
one Docker host stay isolated. Set `RUNNER_NETWORK` or `FLEET_MANAGER_ID` if
you override the project name.

On an ARM VM, set `RUNNER_FLEET_ID=e1-large-arm64` and `RUNNER_ARCH=arm64`.

The destination must allow the Docker socket. Fleet Manager starts runners on
the same host.

## Coolify one-click catalog

[template/](template/) holds the files for the Coolify repository. Copy
`template/superplane.yaml` to `templates/compose/superplane.yaml` and
`template/svgs/superplane.svg` to `svgs/superplane.svg`.

For coolify-docs, copy `coolify-docs-superplane.mdx` to
`content/docs/services/superplane.mdx` and the same SVG to
`public/images/services/superplane-logo.svg`.

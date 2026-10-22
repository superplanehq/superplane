# Coolify one-click submission

Do not open a Coolify pull request while external contributions are paused.
Coolify also requires at least 1,000 GitHub stars on the SuperPlane repository.

When Coolify reopens contributions, copy the files below. Test the image
Compose file as a **Docker Compose Empty** resource first.

## Coolify template repository

Repository: [coollabsio/coolify](https://github.com/coollabsio/coolify)

Open the template pull request against the current `release/v4.x` branch
(for example `release/v4.5`). Do not target `next`.

| Source in this directory | Destination in Coolify |
| --- | --- |
| `docker-compose.image.yml` | `templates/compose/superplane.yaml` |
| `superplane.svg` | `svgs/superplane.svg` |

Before you copy the Compose file:

1. Pin `SUPERPLANE_VERSION` to the current published GHCR tag.
   Coolify rejects floating tags such as `latest`.
2. Keep `# logo: svgs/superplane.svg` and `# port: 8000`.
3. Keep `SERVICE_URL_APP_8000` and `SERVICE_REALBASE64_32_ENCRYPTION`.
4. Do not add `container_name`.
5. Do not publish host `ports`. Coolify routes to `app` on port `8000`.

## Coolify docs repository

Repository: [coollabsio/coolify-docs](https://github.com/coollabsio/coolify-docs)

The docs pull request is required before Coolify merges the template.

| Source | Destination in coolify-docs |
| --- | --- |
| [coolify-docs-superplane.mdx](coolify-docs-superplane.mdx) | `content/docs/services/superplane.mdx` |
| `superplane.svg` | `public/images/services/superplane.svg` |

Link the docs pull request from the template pull request.

## Checklist

- [ ] SuperPlane has at least 1,000 GitHub stars.
- [ ] Coolify accepts external service contributions again.
- [ ] Docker Compose Empty deploy of `docker-compose.image.yml` succeeds.
- [ ] Domain on service `app`, port `8000`, HTTPS works.
- [ ] Owner setup completes. Fleet Manager starts without a pasted PAT.
- [ ] Encryption key stays `env://coolify-v1:` plus generated Base64.
- [ ] Template PR targets `release/v4.x`.
- [ ] Docs PR is open and linked.

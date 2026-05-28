# supergit

Git-backed file storage for SuperPlane, exposed over HTTP. supergit manages bare Git repositories on disk and lets callers create repos, list and read files, and write commits without running Git in the client.

The API is intentionally similar to [code.storage](https://code.storage/docs/reference/api/overview) so SuperPlane can swap storage backends with a small HTTP client.

supergit lives in the SuperPlane repo for now and is planned to move into its own repository once it stabilizes.

## Requirements

- Go 1.26+
- `git` on `PATH` (used for all repository operations)

## Run locally

```bash
cd supergit
go run ./cmd/supergit
```

By default the server listens on `:8080` and stores repositories under `/var/lib/supergit/repos`.

Health check:

```bash
curl http://localhost:8080/health
```

## Docker

From the SuperPlane repo root:

```bash
docker compose -f docker-compose.dev.yml up -d supergit
```

Or build the image directly:

```bash
docker build -t supergit ./supergit
docker run --rm -p 8080:8080 -v supergit-data:/var/lib/supergit/repos supergit
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SUPERGIT_ROOT` | `/var/lib/supergit/repos` | Directory for bare Git repositories |
| `SUPERGIT_PORT` | `8080` | HTTP listen port |
| `SUPERGIT_DEFAULT_BRANCH` | `main` | Default branch when a repo is created without one |
| `SUPERGIT_MAX_FILE_BYTES` | `10485760` (10 MiB) | Maximum size of a single file in a commit |
| `SUPERGIT_MAX_COMMIT_BYTES` | `26214400` (25 MiB) | Maximum total blob size per commit |

## SuperPlane integration

In local development, SuperPlane talks to supergit via the `supergit` canvas storage driver:

```env
CANVAS_STORAGE_DRIVER=supergit
CANVAS_STORAGE_SUPERGIT_BASE_URL=http://supergit:8080/api
```

See the main repo `docker-compose.dev.yml` for the full dev setup.

## Repository IDs

Repository IDs must match:

```text
orgs/{organization_uuid}/canvases/{canvas_uuid}
```

Both UUIDs must be valid. Paths under `.superplane/` are reserved and cannot be written by callers.

## API reference

See [docs/api.md](docs/api.md) for endpoint details, request/response shapes, and the NDJSON commit format.

## Project layout

```text
supergit/
  cmd/supergit/     HTTP server entrypoint
  internal/api/     Route handlers
  internal/storage/ Git storage implementation
  internal/config/  Environment configuration
  docs/             API documentation
```

# CLI Quickstart

This guide walks through building the SuperPlane CLI locally, authenticating against a development environment, and exercising the new `console` and `widgets` commands end-to-end. Use it as your first-time checklist when verifying CLI changes before opening a PR.

## 1. Bring up the dev stack

From the repository root:

```bash
make dev.up        # build dev-base image and start app/db/rabbitmq containers
make dev.setup     # install deps, run codegen, migrate superplane_dev
make dev.server    # in another terminal: starts air (Go) + Vite (UI) at :8000
```

Verify the API is up: `curl http://localhost:8000/health`.

## 2. Create an owner account and an API token

1. Open `http://localhost:8000` and complete the owner setup flow (`OWNER_SETUP_ENABLED=yes` is the default).
2. Once signed in, open **Organization Settings -> Profile** and generate an API token, or create a service account from **Members -> Service Accounts** and copy its one-time token.

Keep the token handy — it is the only artifact the CLI needs to talk to your local server.

## 3. Build the CLI

On Apple Silicon:

```bash
make cli.build.m1
```

This produces `./build/cli`. On other platforms, run `go build -o build/cli ./cmd/cli` from inside the `app` container, or use `go run ./cmd/cli/main.go ...` for ad hoc invocations.

## 4. Connect and select a canvas

```bash
./build/cli connect http://localhost:8000 <token>
./build/cli whoami
./build/cli canvases list
./build/cli canvases active <canvas-id>
```

After `canvases active`, every command below works without `--canvas-id`. Pass `--canvas-id` explicitly to target a different canvas.

## 5. Console workflow

```bash
# Inspect what the canvas Console looks like today.
./build/cli console get
./build/cli console get -o yaml

# Round-trip the Console as YAML.
./build/cli console export --file console.yaml
./build/cli console import --file console.yaml --yes

# Manage individual panels without rewriting the whole document.
./build/cli console panels list
./build/cli console panels upsert --file panel.yaml
./build/cli console panels delete <panel-id> --yes

# Pull live data from a runtime panel and inspect it without leaving the shell.
./build/cli console data <panel-id>

# Re-run a node from a node panel or row action.
./build/cli console trigger --node <node-name-or-id> --hook run --parameters '{"environment":"prod"}'
```

`console import` and `console clear` always replace all panels and layout because the underlying API is replace-all. Use `--yes` to skip the confirmation prompt in scripts.

## 6. Widgets workflow

The `widgets` commands operate on `TYPE_WIDGET` canvas nodes (annotations are the first supported widget). They reuse the canvas change-management flow, so a draft is created when one does not exist, updated, then published unless `--draft` is set.

```bash
# Discover available widget types from the registry.
./build/cli index widgets
./build/cli index widgets --full -o yaml

# Manage widget instances on the active canvas.
./build/cli widgets list
./build/cli widgets get <widget-id-or-name>
./build/cli widgets add --component annotation --name release-notes \
    --text "Deploys after Friday lunch" --color amber --width 320 --height 160 \
    --position-x 200 --position-y 80
./build/cli widgets update <widget-id-or-name> --text "Updated copy" --color blue
./build/cli widgets delete <widget-id-or-name> --yes
```

`--draft` keeps the change in an unpublished draft (useful when the canvas requires reviewer approval). Without `--draft`, the CLI publishes the draft for you so the change appears immediately in the UI.

## 7. Confirming the round-trip

Open the canvas in the browser at `http://localhost:8000/canvases/<canvas-id>` (or click through from the canvas list) and confirm:

- Console panels appear in the same layout you imported.
- Widget annotations show the text, color, and dimensions you set.
- `./build/cli widgets list` and the canvas inspector agree on every TYPE_WIDGET node.

If the UI does not match, re-run the matching CLI command with `-o yaml` to see exactly what the API returned, and check `make dev.server` logs for backend errors.

## 8. Run the test suite before opening a PR

```bash
make format.go
make test PKG_TEST_PACKAGES=./pkg/cli/...
make lint && make check.build.app
```

The CLI unit tests for `console` and `widgets` rely on `httptest.Server` mocks, so they do not need a running `app` container — but they do need the codegen step from `make dev.setup` to have completed at least once.

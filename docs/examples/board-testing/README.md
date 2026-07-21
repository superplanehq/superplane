# Foo Bar board tester

This pair of YAML files creates a no-integration test harness for the console board panel:

- `foo-bar-board-canvas.yaml` — a manual `start` trigger followed by `addMemory`.
- `foo-bar-board-console.yaml` — a concise inline prompt panel and a board reading `foo_bar_tasks`.

## Local setup

1. Create a canvas, open its canvas YAML importer, and import `foo-bar-board-canvas.yaml`.
2. Save/publish the canvas so its manual trigger can run.
3. Open Console, open the console YAML importer, and import `foo-bar-board-console.yaml`.
4. Enter a task prompt and click **Add task**.
5. Wait for the run to finish. The board updates from the `memory_updated` websocket event.
6. Submit again in a later second to append another row with a different timestamp-derived status, owner, and priority.

The fixture deliberately derives pseudo-random values from the current Unix timestamp. It is deterministic, needs no
runner or external integration, and gives the board realistic lane/card data quickly.

To start over, delete the `foo_bar_tasks` namespace from the Memory tab or create a fresh canvas.

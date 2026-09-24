---
title: Local database snapshot
---

# Local database snapshot

A local dump lets you create a new SuperPlane environment without owner
setup, organization setup, or GitHub connection.

Complete those steps once in the UI. Then save a dump of `superplane_dev`.
Restore that dump when you start a new local environment.

Use this path when a script or agent creates local environments.

`make db.snapshot` and `make db.restore` use this worktree's Compose stack.
Postgres in that stack is `db:5432`. `PUBLIC_API_PORT` in `.env` is the UI
port.

## Save a dump

1. Complete owner, organization, GitHub, and workspace setup in the UI.
2. Run `make db.snapshot`. SuperPlane writes `.local/superplane_dev.dump`.
3. Run `make db.snapshot` again after later setup changes you want to keep.
4. Do not commit `.local/`. The dump is local only.

## Restore a dump

If the new environment is a different worktree, copy
`.local/superplane_dev.dump` into that worktree first.

1. Run `make db.restore`. SuperPlane loads the dump and applies pending
   migrations.
2. If `make dev.server` is already running, restart it.

Restore replaces `superplane_dev` only. It does not change `superplane_test`.

## Limits

The dump stores Postgres rows. It does not restore blob files. GitHub App
credentials stay in `.env`.

## Multiple local instances

Run `make db.snapshot` and `make db.restore` in the worktree that serves the
UI you use. See [Running Multiple Local Instances](multi-instance-dev.md).

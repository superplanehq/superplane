# SuperPlane Deployment

This canvas orchestrates the SuperPlane delivery pipelines: CI for the OSS
and SaaS code bases, image promotion, Helm-based staging/production
rollouts, tagged releases of the OSS product, and post-deploy
documentation updates. Discord notifications are wired into every
failure path so the team finds out fast.

## At a glance

- **36 nodes / 30 edges** across five independent workflows
- **5 triggers**: 3 GitHub repositories (`launchpad`, `superplane`, `saas`),
  1 GitHub tag trigger, 1 inbound webhook from the docs deployment
- **Integrations**: GitHub, Semaphore, Discord, Cloudflare

## Workflows

### 1. Launchpad — Helm staging & production rollouts

Triggered by pushes to `superplanehq/launchpad@main`. The push payload is
filtered by changed paths so the canvas only rolls out the relevant
environment.

- `Filter: Helm - Staging` matches paths under `helm/staging/*`, then runs
  `Deploy: Helm - Staging` (`.semaphore/update-installation__staging.yml`).
- `Filter: Helm - Production` matches paths under `helm/production/*`, then
  runs `Deploy: Helm - Production`
  (`.semaphore/update-installation__production.yml`).
- Failures from either deploy fan into the `Discord: Helm update failed`
  message in the `deployments` channel.

### 2. SuperPlane OSS — CI, docs, dev image, prod promotion

Triggered by pushes to `superplanehq/superplane@main`. The `CI` node runs
`.semaphore/semaphore.yml` against the pushed SHA. On success it fans out
in parallel to:

- `Update Component Docs` — runs `.semaphore/update-components-docs.yml`
  on `launchpad`, passing `SUPERPLANE_SHA`. Failure pings
  `Discord: Documentation Deployment Failed`.
- `Build Dev Image Cache` — pre-warms the dev base image via
  `.semaphore/release-dev-base-image.yml` on `superplane-private`.
- `Promote - Production` — runs `.semaphore/deploy-image.yml` with
  `ENVIRONMENT=production`. On pass it posts `Discord: Deployed to Prod`,
  on fail it posts `Discord: Prod Deployment Failed`.

A CI failure on `main` directly notifies `Discord: CI on Main Failed`.

### 3. SuperPlane OSS — Tagged release

Triggered by `v.*` tags created on `superplanehq/superplane`. The release
fan-out is:

1. `Create Release` runs `.semaphore/github-release.yml` with the version
   parsed from the tag ref.
2. On pass, two release tracks run in parallel:
   - **Stable track** (`Mark Release as STABLE` →
     `.semaphore/github-release-promote.yml` with `REL_CHANNEL=stable`),
     which then fans into:
     - `Update install.superplane.com` — rewrites the Cloudflare redirect
       rule for `install.superplane.com/*` to point at the new tag.
     - `Publish to APT (S3)` —
       `.semaphore/publish-apt.yml`. Failure → `Discord: APT Publish Failed`.
     - `Publish to NPM` —
       `.semaphore/publish-npm.yml`. Failure → `Discord: NPM Publish Failed`.
   - **Beta track** (`Mark Release as BETA` with `REL_CHANNEL=beta`) →
     `Update install.superplane.com/beta` Cloudflare redirect rule.

> See the in-canvas annotation `How to create a new tag?` for the
> `make tag.create.{patch,minor,major}` shortcuts.

### 4. SaaS — CI and image promotion

Triggered by pushes to `superplanehq/saas@main`. The chain is sequential:

`SaaS CI` → `SaaS Build Image - Staging` → `SaaS Build Image - Production`,
all running on Semaphore against the pushed commit SHA.

Notifications:

- CI failure → `Discord: SaaS CI on Main Failed`.
- Production build pass → `Discord: SaaS Deployed to Prod`.
- Production build fail → `Discord: SaaS Prod Deployment Failed`.

### 5. Documentation deployment webhook

Inbound webhook `On Docs Repo Deployment Finished` (no auth) routes
straight to `Discord: Documentation Update Failed 2` to alert the team
when the docs site finishes a deployment in a failure state.

## Integrations

| Vendor | Used by |
| --- | --- |
| GitHub | `launchpad`, `superplane`, `saas` push triggers and the `superplane` tag trigger |
| Semaphore | All build, deploy, release, publish, and docs-update workflows |
| Discord | All failure and success notifications (channel `deployments`) |
| Cloudflare | `install.superplane.com` and `install.superplane.com/beta` redirect rules |

## Files

- `canvas.yaml` — the canvas definition (nodes, edges, triggers)
- `console.yaml` — the Console layout for this app
- `README.md` — this file

## Open TODOs

- The `TODO` annotation flags that marking every new tag both `:stable`
  and `:beta` is not the desired long-term behaviour; the team wants a
  promotion process instead.
- The `How to create a new tag?` annotation suggests adding manual-run
  buttons inside the canvas to kick off tag creation directly.

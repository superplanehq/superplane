# SuperPlane Dev Base

## Overview

This PRD defines a new container build strategy for SuperPlane where local development, CI test jobs,
and build stages consume a versioned prebuilt dev base image instead of reinstalling tooling per
pipeline run or per Docker build.

The implementation introduces a dedicated base image: `superplane-dev.base`.

## Problem Statement

SuperPlane's previous Docker workflow repeatedly installed core dependencies (Go, Node.js,
PostgreSQL client, migration tooling, protobuf tooling, and test browsers) across local and CI runs.

This increased build/test time, duplicated installation logic, and made CI behavior more sensitive
to network flakiness.

A deterministic, reusable image layer that can be versioned solves most of these problem.

## Proof-Of-Concept results

I ran a POC to get some numbers:

- An image that contains everything abouve is around: 800 mb
- It downloads in CI in about: 35 seconds
- This is faster by around 100 seconds from what we have atm.

## One vs. Multiple base images

I experimented with two approaches:

1/ Having one base image that is shared between the app and agent
2/ Having two base images, one for app, one for the agent

Option 1/ was faster on CI. This approach has downsides, but I'm mostly looking for performance here.

## Goals

1. Create a reusable and versioned SuperPlane dev base image that bundles shared toolchain dependencies.
2. Reduce repeated setup in CI by shifting dependency installation into the base image.
3. Provide a release pipeline that publishes multi-architecture (`amd64`, `arm64`) base images to GHCR.
4. Keep production runner images minimal while preserving existing build outputs and runtime behavior.

## Non-Goals

- Reworking production runtime architecture beyond base-image adoption in build stages.
- Introducing a new package manager or language runtime strategy beyond what is already present in this implementation.
- Redesigning application-level feature behavior (this is infrastructure/tooling focused).

## Functional Requirements

### 1) Dev Base Image Definition

- Add `release/superplane-dev-base/Dockerfile` as the canonical source for the shared development base image.

- Base image must include:
  - Ubuntu 24.04
  - Go toolchain and Go developer tools used in CI/dev
  - Node.js
  - PostgreSQL client (`createdb` and related tools)
  - `migrate`
  - `protoc`
  - Python 3.12 and `uv`
  - Playwright browser install (`chromium-headless-shell`) with dependencies

- Image should clean package caches and temporary files to reduce layer bloat.

### 2) Dev Base Image Build and Publish Flow

- Add `release/superplane-dev-base/build.sh` to build and push architecture-specific image tags.
- Add `.semaphore/release-dev-base-image.yml` to build a multi-platform image
- Push results to GHCR

### 3) App Dockerfile Adoption

- Update root `Dockerfile` so `dev` and `builder` stages use `superplane-dev-base`.
- Preserve existing app build outputs and runner packaging while removing duplicated tool installation logic from app Docker stages.

### 4) Agent Dockerfile Adoption

- Update `agent/Dockerfile` so `base`/`dev`/`builder` stages use `SUPERPLANE_DEV_BASE_IMAGE`.
- Keep agent runtime on a slim Python runner image while relying on base image in build/dev stages.

### 5) Local Development Compose Alignment

- Update `docker-compose.dev.yml` so `app` and `agent` services run from explicit base image tags instead of per-service local Docker build targets.
- Align Playwright browser path with preinstalled browsers (`/ms-playwright`).

### 6) CI Test Workflow Adjustments

- Update `Makefile` test setup/start flow to:
  - Use shared DB bootstrap script `scripts/ci_db_setup`
  - Start required services with `docker compose up --wait --wait-timeout 60 ...`
- Add `scripts/ci_db_setup` for deterministic CI database creation/migration for app and agent test DBs.
- Simplify Semaphore QA E2E flow by removing separate Playwright cache/install steps now covered by the base image.

## Acceptance Criteria

1. A versioned SuperPlane dev base image can be built and published for `amd64` and `arm64`, with a manifest tag.
2. App `dev` and `builder` stages consume the explicit base image tag.
3. Agent `dev` and `builder` stages consume the explicit base image tag.
4. `docker-compose.dev.yml` app and agent services boot from explicit image tags and remain functional for development.
5. CI QA pipeline runs without separate Playwright setup caching/installation steps.
6. Test database setup in CI is performed via `scripts/ci_db_setup` and succeeds for both app and agent test DBs.

## Success Metrics

- Reduced median CI setup time in QA jobs.
- Lower rate of CI failures caused by dependency installation/network flakiness.
- Reduced local onboarding/setup time for new developer environments.
- Fewer Docker layer rebuilds for unchanged toolchain dependencies.

## Risks and Mitigations

- **Risk:** Base image tag drift between `Dockerfile` and `docker-compose.dev.yml`.  
  **Mitigation:** Standardize version bump process and document single-source version update workflow.

- **Risk:** Large base image size due to preinstalled tooling and browsers.  
  **Mitigation:** Continue aggressive cleanup in install scripts and review layer composition each release.

- **Risk:** Toolchain updates in base image break downstream builds.  
  **Mitigation:** Require release pipeline verification and run QA suite against new base image tags
  before broad adoption.

- **Risk:** Architecture-specific differences between `amd64` and `arm64`.  
  **Mitigation:** Keep dedicated per-arch build blocks and manifest verification in release pipeline.

## Rollout Plan

1. Publish base image release tags through `.semaphore/release-dev-base-image.yml`.
2. Update consuming Dockerfiles/compose to the new versioned tag.
3. Run full Semaphore QA pipeline and local smoke tests.
4. Promote tag for team-wide development use and document update cadence.

## Open Questions

1. Should base image version references be centralized in one env file to prevent tag mismatch?
2. Should the CI pipeline pin base image digests (not just tags) for stricter reproducibility?
3. What is the target SLA for base image refreshes when critical CVEs are announced?

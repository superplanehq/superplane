#!/usr/bin/env bash
#
# Validate all built-in template canvases (templates/canvases/*.yaml) by
# generating pkg/protos and running TestCanvasesTemplatesParse, matching the
# same ParseCanvas path as template seeding.
#
# Intended to run inside the dev app container (see `make check.templates`),
# where protoc and protoc-gen-go* are preinstalled. Requires only scripts/protoc.sh
# (not the full `make pb.gen` gateway / OpenAPI pipeline).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Keep in sync with MODULES in Makefile (pb.gen).
MODULES="authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,components,actions,triggers,widgets,blueprints,canvases,service_accounts,agents,usage,private/agents"

bash scripts/protoc.sh "$MODULES"

go test ./pkg/templates/ -count=1 -run TestCanvasesTemplatesParse

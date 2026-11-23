#!/bin/sh

set -eu

MIGRATE_VERSION=v4.20.1

echo "Building Go Migrate ${MIGRATE_VERSION}"

build_dir=$(mktemp -d)
trap 'chmod -R u+w "$build_dir"; rm -rf "$build_dir"' EXIT

# Only PostgreSQL and local migration files are used by SuperPlane.
CGO_ENABLED=0 GOBIN=/usr/bin \
  GOMODCACHE="$build_dir/mod" GOCACHE="$build_dir/build" \
  go install -tags 'postgres,file' \
    -ldflags "-X main.Version=${MIGRATE_VERSION#v}" \
    "github.com/golang-migrate/migrate/v4/cmd/migrate@${MIGRATE_VERSION}"

#!/usr/bin/env bash
# Exercise publication with isolated build and upload commands. Artifact bytes
# must stay distinct while uploads overlap the second frontend build.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/repo/scripts"
cp "$repo_root/scripts/build-image" "$fixture/repo/scripts/"

cat > "$fixture/bin/make" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
for arg in "$@"; do
  case "$arg" in *=*) export "$arg" ;; esac
done
case "$1" in
  image.prepare)
    mkdir -p "$IMAGE_BUILD_DIR/app/build" "$IMAGE_BUILD_DIR/usr/bin"
    printf 'shared backend' > "$IMAGE_BUILD_DIR/app/build/superplane"
    ;;
  image.package)
    mkdir -p "$RESULTS/$IMAGE_TAG"
    cp "$IMAGE_BUILD_DIR/app/build/superplane" "$RESULTS/$IMAGE_TAG/backend"
    cp "$IMAGE_BUILD_DIR/app/pkg/web/assets/dist/index.html" "$RESULTS/$IMAGE_TAG/index.html"
    touch "$RESULTS/packaged-$IMAGE_TAG"
    ;;
  image.auth|image.push)
    test -f "$RESULTS/upload-complete"
    echo "$1" >> "$RESULTS/published"
    ;;
  *) echo "Unexpected make target: $1" >&2; exit 1 ;;
esac
MOCK
cat > "$fixture/bin/docker" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
asset_base=""
for arg in "$@"; do
  case "$arg" in VITE_ASSET_BASE_URL=*) asset_base="${arg#*=}" ;; esac
done
# Execute the real frontend command against a fixture npx in the repository.
command="${!#}"
command="${command//\/app\//$PWD/}"
VITE_ASSET_BASE_URL="$asset_base" bash -c "$command"
MOCK
cat > "$fixture/bin/npx" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
while [ "$1" != "--outDir" ]; do shift; done
shift
mkdir -p "$1"
printf '%s' "${VITE_ASSET_BASE_URL:-same-origin}" > "$1/index.html"
# Empty Vite base is the same-origin variant.
if [ ! -s "$1/index.html" ]; then printf same-origin > "$1/index.html"; fi
MOCK
cat > "$fixture/bin/git" <<'MOCK'
#!/usr/bin/env bash
printf test-sha
MOCK
cat > "$fixture/repo/scripts/upload-static-assets-r2" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
cp "$DIST_DIR/index.html" "$RESULTS/upload-start"
# The upload must overlap image packaging. Bound the wait to avoid hanging tests.
for ((attempt=0; attempt<100; attempt++)); do
  if [ -f "$RESULTS/packaged-test-sha-selfhosted" ]; then break; fi
  sleep 0.05
done
test -f "$RESULTS/packaged-test-sha-selfhosted"
cp "$DIST_DIR/index.html" "$RESULTS/upload-end"
if [ "${FAIL_UPLOAD:-0}" = 1 ]; then exit 1; fi
touch "$RESULTS/upload-complete"
MOCK
chmod +x "$fixture/bin/"*
mkdir -p "$fixture/repo/"{web_src,tmp,db/migrations,db/data_migrations,api/swagger,rbac,templates,ee,scripts/docker}
for file in LICENSE NOTICE docker-entrypoint.sh ee/LICENSE scripts/docker/install-postgresql-client.sh; do
  echo fixture > "$fixture/repo/$file"
done

export PATH="$fixture/bin:$PATH"
export SEMAPHORE_GIT_REF_TYPE=branch SEMAPHORE_GIT_BRANCH=main
export RESULTS="$fixture/success"
mkdir -p "$RESULTS"
bash "$fixture/repo/scripts/build-image"
cmp "$RESULTS/test-sha/backend" "$RESULTS/test-sha-selfhosted/backend"
test "$(cat "$RESULTS/test-sha/index.html")" = 'https://assets.superplane.com/releases/test-sha/'
test "$(cat "$RESULTS/test-sha-selfhosted/index.html")" = same-origin
cmp "$RESULTS/upload-start" "$RESULTS/upload-end"
cmp "$RESULTS/upload-end" "$RESULTS/test-sha/index.html"
test "$(grep -c image.push "$RESULTS/published")" = 2

export RESULTS="$fixture/failure" FAIL_UPLOAD=1
mkdir -p "$RESULTS"
if bash "$fixture/repo/scripts/build-image"; then
  echo 'Failed upload must fail the image job' >&2
  exit 1
fi
test ! -e "$RESULTS/published"

export RESULTS="$fixture/pr" SEMAPHORE_GIT_REF_TYPE=pull-request
mkdir -p "$RESULTS"
bash "$fixture/repo/scripts/build-image"
test "$(cat "$RESULTS/test-sha/index.html")" = same-origin
test ! -e "$RESULTS/published"
test ! -e "$RESULTS/upload-start"
echo 'CI image publication tests passed.'

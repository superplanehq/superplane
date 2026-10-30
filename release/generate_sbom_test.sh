#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT
CHECKOUT_DIR="${WORK_DIR}/checkout"

# Run the real generator and Syft in a small checkout, without local dependencies.
mkdir -p "${CHECKOUT_DIR}/release" "${CHECKOUT_DIR}/web_src" "${CHECKOUT_DIR}/tmp/bin"
cp "${SCRIPT_DIR}/generate-sbom.sh" "${CHECKOUT_DIR}/release/"
if [ -x "${REPO_ROOT}/tmp/bin/syft" ]; then
  ln -s "${REPO_ROOT}/tmp/bin/syft" "${CHECKOUT_DIR}/tmp/bin/syft"
fi
cd "${CHECKOUT_DIR}"

cat >go.mod <<'EOF'
module example.com/application

go 1.23.0

require example.com/backend-dependency v1.2.3
EOF

cat >web_src/package-lock.json <<'EOF'
{
  "name": "frontend",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "packages": {
    "": {
      "name": "frontend",
      "version": "1.0.0",
      "dependencies": { "frontend-dependency": "2.3.4" }
    },
    "node_modules/frontend-dependency": {
      "version": "2.3.4",
      "dependencies": { "transitive-dependency": "3.4.5" }
    },
    "node_modules/transitive-dependency": { "version": "3.4.5" }
  }
}
EOF

bash release/generate-sbom.sh v1.0.0
cp build/superplane-single-host-tarball-v1.0.0/superplane-sbom.json "${WORK_DIR}/clean.json"

# These files represent unrelated packages in a prepared CI workspace.
for directory in .git/objects tmp/go/pkg/mod/cached build/previous-release web_src/node_modules/installed; do
  mkdir -p "${directory}"
  cat >"${directory}/go.mod" <<'EOF'
module example.com/cached-tool

go 1.23.0

require example.com/unrelated-dependency v1.9.9
EOF
done

# Exercise symlinked workspaces explicitly, including on Linux CI.
ln -s "${CHECKOUT_DIR}" "${WORK_DIR}/linked-checkout"
cd "${WORK_DIR}/linked-checkout"
bash release/generate-sbom.sh v1.0.0
node - "${WORK_DIR}/clean.json" build/superplane-single-host-tarball-v1.0.0/superplane-sbom.json <<'EOF'
const assert = require("node:assert/strict");
const fs = require("node:fs");
const [clean, prepared] = process.argv.slice(2).map(path => JSON.parse(fs.readFileSync(path, "utf8")));
const inventory = sbom => [...new Set(sbom.packages.map(pkg => `${pkg.name}@${pkg.versionInfo || ""}`))].sort();

assert.equal(prepared.spdxVersion, "SPDX-2.3");
for (const dependency of [
  "example.com/backend-dependency@v1.2.3",
  "frontend-dependency@2.3.4",
  "transitive-dependency@3.4.5",
]) {
  assert(inventory(clean).includes(dependency), `Missing declared dependency: ${dependency}`);
}
assert.deepEqual(inventory(prepared), inventory(clean), "Local caches and build output changed the release SBOM");
EOF

echo "SBOM dependency scope tests passed."

#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "$#" -ne 4 ]; then
  echo "Usage: release/component-release-artifact.sh <push|pull> <component> <version> <release-directory>" >&2
  exit 1
fi

ACTION="$1"
COMPONENT="$2"
VERSION="$3"
RELEASE_DIR="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${ACTION}" in
  push | pull) ;;
  *)
    echo "Error: action must be push or pull" >&2
    exit 1
    ;;
esac

case "${COMPONENT}" in
  superplane | cli | runner | fleet-manager) ;;
  *)
    echo "Error: unsupported component ${COMPONENT}" >&2
    exit 1
    ;;
esac

node "${SCRIPT_DIR}/component-release.js" validate-version "${VERSION}" >/dev/null

if [ "$(basename "${RELEASE_DIR}")" != "${VERSION}" ] || \
  [ "$(basename "$(dirname "${RELEASE_DIR}")")" != "${COMPONENT}" ]; then
  echo "Error: release directory must end with ${COMPONENT}/${VERSION}" >&2
  exit 1
fi

BUNDLE_NAME="${COMPONENT}-${VERSION}.tar.gz"
BUNDLE_PATH="$(pwd)/${BUNDLE_NAME}"
RELEASE_PARENT="$(dirname "${RELEASE_DIR}")"
PARTS_MANIFEST_NAME="semaphore-artifact-parts.json"
# Semaphore rejects one workflow artifact larger than 2 GiB. Each piece stays
# under 1536 MiB. Files that remain in the gzip must also total under 2 GiB,
# because those bytes do not shrink when they are already compressed.
DEFAULT_MAX_PART_BYTES=1610612736
SEMAPHORE_FILE_LIMIT_BYTES=2147483648

positive_byte_limit() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[1-9][0-9]*$ ]]; then
    echo "Error: ${name} must be a positive integer" >&2
    exit 1
  fi
  if [ "${value}" -gt "${SEMAPHORE_FILE_LIMIT_BYTES}" ]; then
    echo "Error: ${name} must be at most 2 GiB" >&2
    exit 1
  fi
}

max_part_bytes() {
  local limit="${COMPONENT_RELEASE_ARTIFACT_MAX_BYTES:-${DEFAULT_MAX_PART_BYTES}}"
  positive_byte_limit "COMPONENT_RELEASE_ARTIFACT_MAX_BYTES" "${limit}"
  if [ "${limit}" -ge "${SEMAPHORE_FILE_LIMIT_BYTES}" ]; then
    echo "Error: COMPONENT_RELEASE_ARTIFACT_MAX_BYTES must be under 2 GiB" >&2
    exit 1
  fi
  printf '%s' "${limit}"
}

bundle_limit_bytes() {
  local limit="${COMPONENT_RELEASE_ARTIFACT_BUNDLE_LIMIT_BYTES:-${DEFAULT_MAX_PART_BYTES}}"
  positive_byte_limit "COMPONENT_RELEASE_ARTIFACT_BUNDLE_LIMIT_BYTES" "${limit}"
  printf '%s' "${limit}"
}

list_large_release_files() {
  node - "${RELEASE_DIR}" "$1" "$2" "${PARTS_MANIFEST_NAME}" <<'JS'
const fs = require("fs");
const path = require("path");

const dir = process.argv[2];
const partLimit = Number(process.argv[3]);
const bundleLimit = Number(process.argv[4]);
const manifestName = process.argv[5];
const safeName = /^[A-Za-z0-9._-]+$/;
const files = [];
let nestedBytes = 0;

function consider(full, rel) {
  const stat = fs.lstatSync(full);
  if (stat.isSymbolicLink()) {
    return;
  }
  if (stat.isDirectory()) {
    for (const name of fs.readdirSync(full)) {
      consider(path.join(full, name), `${rel}/${name}`);
    }
    return;
  }
  if (!stat.isFile()) {
    return;
  }
  if (rel.includes("/")) {
    if (stat.size > partLimit || stat.size >= bundleLimit) {
      console.error(`Error: large release file in a subdirectory is not supported: ${rel}`);
      process.exit(1);
    }
    nestedBytes += stat.size;
    return;
  }
  files.push({ name: rel, size: stat.size });
}

for (const name of fs.readdirSync(dir)) {
  consider(path.join(dir, name), name);
}

const selected = new Set();
for (const file of files) {
  if (file.size > partLimit) {
    selected.add(file.name);
  }
}

function remainingBytes() {
  return nestedBytes + files.filter((file) => !selected.has(file.name)).reduce((sum, file) => sum + file.size, 0);
}

const bySize = [...files].sort((left, right) => right.size - left.size || left.name.localeCompare(right.name));
while (remainingBytes() >= bundleLimit) {
  const next = bySize.find((file) => !selected.has(file.name));
  if (!next) {
    console.error("Error: release bundle cannot fit in one Semaphore artifact");
    process.exit(1);
  }
  selected.add(next.name);
}

for (const name of selected) {
  if (name === manifestName) {
    console.error(`Error: ${manifestName} is larger than one Semaphore artifact`);
    process.exit(1);
  }
  if (!safeName.test(name)) {
    console.error(`Error: large release file name is not safe: ${name}`);
    process.exit(1);
  }
  process.stdout.write(`${name}\n`);
}
JS
}

hash_file() {
  node - "$1" <<'JS'
const fs = require("fs");
const crypto = require("crypto");

const hash = crypto.createHash("sha256");
const input = fs.createReadStream(process.argv[2]);
input.on("error", (error) => {
  console.error(`Error: ${error.message}`);
  process.exit(1);
});
input.on("data", (chunk) => hash.update(chunk));
input.on("end", () => process.stdout.write(hash.digest("hex")));
JS
}

split_release_file() {
  node - "$1" "$2" "$3" "$4" "${SEMAPHORE_FILE_LIMIT_BYTES}" <<'JS'
const fs = require("fs");
const path = require("path");

const source = process.argv[2];
const destDir = process.argv[3];
const max = Number(process.argv[4]);
const partPrefix = process.argv[5];
const semaphoreLimit = Number(process.argv[6]);
const input = fs.openSync(source, "r");
const size = fs.fstatSync(input).size;
const chunk = Buffer.alloc(8 * 1024 * 1024);
const names = [];
let offset = 0;
let index = 0;

try {
  while (offset < size) {
    if (index > 999) {
      console.error("Error: release file needs more than 1000 artifact parts");
      process.exit(1);
    }
    const partName = `${partPrefix}.part${String(index).padStart(3, "0")}`;
    const partPath = path.join(destDir, partName);
    const output = fs.openSync(partPath, "wx");
    let remaining = Math.min(max, size - offset);
    try {
      while (remaining > 0) {
        const read = fs.readSync(input, chunk, 0, Math.min(chunk.length, remaining), offset);
        if (read <= 0) {
          console.error("Error: unexpected end of release file");
          process.exit(1);
        }
        fs.writeSync(output, chunk, 0, read);
        offset += read;
        remaining -= read;
      }
    } finally {
      fs.closeSync(output);
    }
    const partSize = fs.statSync(partPath).size;
    if (partSize > max || partSize >= semaphoreLimit) {
      console.error(`Error: artifact part is not under the Semaphore file limit: ${partName}`);
      process.exit(1);
    }
    names.push(partName);
    index += 1;
  }
} finally {
  fs.closeSync(input);
}

process.stdout.write(`${names.join("\n")}\n`);
JS
}

write_parts_manifest() {
  node - "$1" "$2" <<'JS'
const fs = require("fs");

const lines = fs.readFileSync(process.argv[2], "utf8").split("\n").filter(Boolean);
const files = lines.map((line) => {
  const [filePath, sha256, partsField] = line.split("\t");
  return {
    path: filePath,
    sha256,
    parts: partsField.split(","),
  };
});
fs.writeFileSync(process.argv[3], `${JSON.stringify({ files })}\n`);
JS
}

describe_parts_manifest() {
  node - "$1" "$2" "$3" "${PARTS_MANIFEST_NAME}" <<'JS'
const fs = require("fs");

const component = process.argv[3];
const version = process.argv[4];
const manifestName = process.argv[5];
const safeName = /^[A-Za-z0-9._-]+$/;
const shaPattern = /^[a-f0-9]{64}$/;
let manifest;
try {
  manifest = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
} catch (error) {
  console.error("Error: artifact parts manifest is not valid JSON");
  process.exit(1);
}
if (!manifest || !Array.isArray(manifest.files) || manifest.files.length === 0) {
  console.error("Error: artifact parts manifest does not list files");
  process.exit(1);
}

const seenPaths = new Set();
const seenParts = new Set();
for (const file of manifest.files) {
  if (!file || typeof file.path !== "string" || typeof file.sha256 !== "string" || !Array.isArray(file.parts)) {
    console.error("Error: artifact parts manifest entry is invalid");
    process.exit(1);
  }
  if (!safeName.test(file.path) || file.path === manifestName || seenPaths.has(file.path)) {
    console.error(`Error: artifact parts manifest path is not safe: ${file.path}`);
    process.exit(1);
  }
  if (!shaPattern.test(file.sha256) || file.parts.length === 0) {
    console.error(`Error: artifact parts manifest checksum is invalid: ${file.path}`);
    process.exit(1);
  }
  seenPaths.add(file.path);
  const partNames = file.parts.map((part, index) => {
    const expected = `${component}-${version}--${file.path}.part${String(index).padStart(3, "0")}`;
    if (part !== expected || seenParts.has(part) || !safeName.test(part)) {
      console.error(`Error: artifact part name is not safe: ${part}`);
      process.exit(1);
    }
    seenParts.add(part);
    return part;
  });
  process.stdout.write(`${file.path}\t${file.sha256}\t${partNames.join(",")}\n`);
}
JS
}

join_parts() {
  node - "$@" <<'JS'
const fs = require("fs");
const crypto = require("crypto");

const dest = process.argv[2];
const parts = process.argv.slice(3);
const semaphoreLimit = 2147483648;
if (parts.length === 0) {
  console.error("Error: artifact parts list is empty");
  process.exit(1);
}

const hash = crypto.createHash("sha256");
const output = fs.createWriteStream(dest, { flags: "wx" });
let index = 0;
let failed = false;

function fail(message) {
  if (failed) {
    return;
  }
  failed = true;
  console.error(message);
  output.destroy();
  try {
    fs.unlinkSync(dest);
  } catch (error) {
    if (error.code !== "ENOENT") {
      console.error(`Error: ${error.message}`);
    }
  }
  process.exit(1);
}

function writeNext() {
  if (failed) {
    return;
  }
  if (index >= parts.length) {
    output.end();
    return;
  }
  const part = parts[index];
  index += 1;
  let stat;
  try {
    stat = fs.lstatSync(part);
  } catch (error) {
    fail(`Error: artifact part is missing: ${part}`);
    return;
  }
  if (!stat.isFile()) {
    fail(`Error: artifact part is not a regular file: ${part}`);
    return;
  }
  if (stat.size >= semaphoreLimit) {
    fail(`Error: artifact part is not under 2 GiB: ${part}`);
    return;
  }
  const input = fs.createReadStream(part);
  input.on("error", (error) => fail(`Error: ${error.message}`));
  input.on("data", (chunk) => {
    if (failed) {
      return;
    }
    hash.update(chunk);
    if (!output.write(chunk)) {
      input.pause();
      output.once("drain", () => input.resume());
    }
  });
  input.on("end", writeNext);
}

output.on("error", (error) => fail(`Error: ${error.message}`));
output.on("finish", () => process.stdout.write(hash.digest("hex")));
writeNext();
JS
}

discard_release_dir() {
  rm -rf "${RELEASE_DIR}"
}

push_bundle() {
  if [ ! -d "${RELEASE_DIR}" ]; then
    echo "Error: release directory ${RELEASE_DIR} does not exist" >&2
    exit 1
  fi
  if [ -e "${BUNDLE_PATH}" ]; then
    echo "Error: release bundle ${BUNDLE_NAME} already exists" >&2
    exit 1
  fi

  local limit bundle_limit work_dir large_file
  local -a large_files
  limit="$(max_part_bytes)"
  bundle_limit="$(bundle_limit_bytes)"
  work_dir="$(mktemp -d)"
  trap 'rm -rf "'"${work_dir}"'"' EXIT
  list_large_release_files "${limit}" "${bundle_limit}" >"${work_dir}/large-files"
  large_files=()
  while IFS= read -r large_file; do
    if [ -n "${large_file}" ]; then
      large_files+=("${large_file}")
    fi
  done <"${work_dir}/large-files"

  if [ "${#large_files[@]}" -eq 0 ]; then
    tar -C "${RELEASE_PARENT}" -czf "${BUNDLE_PATH}" "${VERSION}"
    artifact push workflow "${BUNDLE_NAME}"
    return 0
  fi

  local name sha parts_csv part_name
  local -a part_names
  : >"${work_dir}/files.tsv"
  for name in "${large_files[@]}"; do
    sha="$(hash_file "${RELEASE_DIR}/${name}")"
    split_release_file \
      "${RELEASE_DIR}/${name}" \
      "${work_dir}" \
      "${limit}" \
      "${COMPONENT}-${VERSION}--${name}" \
      >"${work_dir}/${name}.names"
    parts_csv=""
    while IFS= read -r part_name; do
      if [ -z "${part_name}" ]; then
        continue
      fi
      if [ -n "${parts_csv}" ]; then
        parts_csv="${parts_csv},${part_name}"
      else
        parts_csv="${part_name}"
      fi
    done <"${work_dir}/${name}.names"
    if [ -z "${parts_csv}" ]; then
      echo "Error: ${name} produced no artifact parts" >&2
      exit 1
    fi
    printf '%s\t%s\t%s\n' "${name}" "${sha}" "${parts_csv}" >>"${work_dir}/files.tsv"
    echo "Split ${name} into workflow artifacts of at most ${limit} bytes"
  done

  write_parts_manifest "${work_dir}/files.tsv" "${RELEASE_DIR}/${PARTS_MANIFEST_NAME}"

  local -a exclude_args
  exclude_args=()
  for name in "${large_files[@]}"; do
    exclude_args+=(--exclude "${name}")
  done
  tar -C "${RELEASE_PARENT}" "${exclude_args[@]}" -czf "${BUNDLE_PATH}" "${VERSION}"

  local bundled_path
  tar -tzf "${BUNDLE_PATH}" >"${work_dir}/bundle-paths"
  while IFS= read -r bundled_path; do
    for name in "${large_files[@]}"; do
      if [ "${bundled_path}" = "${VERSION}/${name}" ]; then
        echo "Error: ${name} is still inside the release bundle" >&2
        exit 1
      fi
    done
  done <"${work_dir}/bundle-paths"

  artifact push workflow "${BUNDLE_NAME}"
  while IFS= read -r parts_csv; do
    parts_csv="${parts_csv##*$'\t'}"
    part_names=()
    IFS=',' read -r -a part_names <<< "${parts_csv}"
    for part_name in "${part_names[@]}"; do
      # The path argument is the artifact name. Publish pulls the same
      # name from a different machine, so it cannot be this temp path.
      (
        cd "${work_dir}"
        artifact push workflow "${part_name}"
      )
      rm -f "${work_dir}/${part_name}"
    done
  done <"${work_dir}/files.tsv"
  rm -f "${RELEASE_DIR}/${PARTS_MANIFEST_NAME}"
}

pull_bundle() {
  if [ -e "${RELEASE_DIR}" ]; then
    echo "Error: release directory ${RELEASE_DIR} already exists" >&2
    exit 1
  fi

  artifact pull workflow "${BUNDLE_NAME}"

  local listing_dir
  listing_dir="$(mktemp -d)"
  trap 'rm -rf "'"${listing_dir}"'"' EXIT

  if ! tar -tzf "${BUNDLE_PATH}" >"${listing_dir}/paths"; then
    echo "Error: release bundle is not a valid gzip archive" >&2
    exit 1
  fi
  if ! tar -tvzf "${BUNDLE_PATH}" >"${listing_dir}/entries"; then
    echo "Error: release bundle metadata cannot be read" >&2
    exit 1
  fi

  local entry
  while IFS= read -r entry; do
    case "${entry}" in
      "${VERSION}" | "${VERSION}/" | "${VERSION}/"*)
        if [[ "/${entry}/" == *"/../"* ]]; then
          echo "Error: release bundle contains unsafe path ${entry}" >&2
          exit 1
        fi
        ;;
      *)
        echo "Error: release bundle contains unexpected path ${entry}" >&2
        exit 1
        ;;
    esac
  done <"${listing_dir}/paths"

  while IFS= read -r entry; do
    case "${entry:0:1}" in
      - | d) ;;
      *)
        echo "Error: release bundle contains a link or special file" >&2
        exit 1
        ;;
    esac
  done <"${listing_dir}/entries"

  mkdir -p "${RELEASE_PARENT}"
  tar -C "${RELEASE_PARENT}" -xzf "${BUNDLE_PATH}"
  restore_split_files "${listing_dir}"
}

restore_split_files() {
  local listing_dir="$1"
  local manifest="${RELEASE_DIR}/${PARTS_MANIFEST_NAME}"
  if [ ! -e "${manifest}" ]; then
    return 0
  fi
  if [ -L "${manifest}" ] || [ ! -f "${manifest}" ]; then
    echo "Error: artifact parts manifest is not a regular file" >&2
    discard_release_dir
    exit 1
  fi

  local described="${listing_dir}/parts.tsv"
  if ! describe_parts_manifest "${manifest}" "${COMPONENT}" "${VERSION}" >"${described}"; then
    discard_release_dir
    exit 1
  fi

  local relpath sha parts_csv part_name actual
  local -a part_names part_paths
  while IFS=$'\t' read -r relpath sha parts_csv || [ -n "${relpath}" ]; do
    if [ -z "${relpath}" ]; then
      continue
    fi
    if [ -e "${RELEASE_DIR}/${relpath}" ] || [ -L "${RELEASE_DIR}/${relpath}" ]; then
      echo "Error: ${relpath} is in the release bundle and in artifact parts" >&2
      discard_release_dir
      exit 1
    fi
    part_names=()
    part_paths=()
    IFS=',' read -r -a part_names <<< "${parts_csv}"
    for part_name in "${part_names[@]}"; do
      if ! (
        cd "${listing_dir}"
        artifact pull workflow "${part_name}"
      ); then
        echo "Error: cannot pull artifact part ${part_name}" >&2
        discard_release_dir
        exit 1
      fi
      part_paths+=("${listing_dir}/${part_name}")
    done
    if ! actual="$(join_parts "${listing_dir}/restored-${relpath}" "${part_paths[@]}")"; then
      discard_release_dir
      exit 1
    fi
    if [ "${actual}" != "${sha}" ]; then
      echo "Error: ${relpath} checksum does not match the artifact parts manifest" >&2
      discard_release_dir
      exit 1
    fi
    mv "${listing_dir}/restored-${relpath}" "${RELEASE_DIR}/${relpath}"
    for part_name in "${part_paths[@]}"; do
      rm -f "${part_name}"
    done
  done <"${described}"
}

if [ "${ACTION}" = "push" ]; then
  push_bundle
else
  pull_bundle
fi

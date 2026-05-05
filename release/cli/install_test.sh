#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
INSTALLER="${SCRIPT_DIR}/install.sh"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM HUP

FAKE_BIN="${TMP_DIR}/bin"
mkdir -p "$FAKE_BIN"

cat >"${FAKE_BIN}/uname" <<'EOF'
#!/usr/bin/env sh
case "${1:-}" in
  -s) printf "%s\n" "${FAKE_UNAME_S}" ;;
  -m) printf "%s\n" "${FAKE_UNAME_M}" ;;
  *) exec /usr/bin/uname "$@" ;;
esac
EOF
chmod +x "${FAKE_BIN}/uname"

assert_contains() {
  haystack="$1"
  needle="$2"
  if ! printf "%s" "$haystack" | grep -F "$needle" >/dev/null 2>&1; then
    echo "Expected output to contain: $needle" >&2
    echo "Actual output:" >&2
    printf "%s\n" "$haystack" >&2
    exit 1
  fi
}

run_success_case() {
  uname_s="$1"
  uname_m="$2"
  expected_platform="$3"
  install_dir="${4:-$TMP_DIR/home/.local/bin}"

  output="$(
    FAKE_UNAME_S="$uname_s" \
      FAKE_UNAME_M="$uname_m" \
      PATH="${FAKE_BIN}:$PATH" \
      SUPERPLANE_CLI_INSTALL_DRY_RUN=1 \
      INSTALL_DIR="$install_dir" \
      HOME="$TMP_DIR/home" \
      "$INSTALLER" 2>&1
  )"

  assert_contains "$output" "platform=${expected_platform}"
  assert_contains "$output" "url=https://install.superplane.com/superplane-cli-${expected_platform}"
  assert_contains "$output" "target=${install_dir}/superplane"
}

run_failure_case() {
  uname_s="$1"
  uname_m="$2"
  expected_message="$3"

  set +e
  output="$(
    FAKE_UNAME_S="$uname_s" \
      FAKE_UNAME_M="$uname_m" \
      PATH="${FAKE_BIN}:$PATH" \
      SUPERPLANE_CLI_INSTALL_DRY_RUN=1 \
      HOME="$TMP_DIR/home" \
      "$INSTALLER" 2>&1
  )"
  code=$?
  set -e

  if [ "$code" -eq 0 ]; then
    echo "Expected installer to fail for ${uname_s}/${uname_m}" >&2
    exit 1
  fi

  assert_contains "$output" "$expected_message"
}

run_success_case Darwin arm64 darwin-arm64
run_success_case Darwin x86_64 darwin-amd64
run_success_case Linux x86_64 linux-amd64
run_success_case Linux aarch64 linux-arm64
run_success_case Linux amd64 linux-amd64 "${TMP_DIR}/custom/bin"

run_failure_case FreeBSD x86_64 "Unsupported OS: FreeBSD"
run_failure_case Linux riscv64 "Unsupported architecture: riscv64"

echo "install.sh dry-run tests passed"

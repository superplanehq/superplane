#!/usr/bin/env bash
#
# Scan git history for committed secrets with gitleaks.
# Downloads a pinned binary when gitleaks is not on PATH.

set -euo pipefail

cd "$(dirname "$0")/.."

version="${GITLEAKS_VERSION:-8.28.0}"
echo "==> Committed secrets (gitleaks $version)"
echo "    Working-tree scan (--no-git). Does not print secret values (--redact)."
echo "    Allowlist: .gitleaks.toml (docs, fixtures, dummy compose keys)."

bin=""
if command -v gitleaks >/dev/null 2>&1; then
	bin=$(command -v gitleaks)
	echo "OK    using $bin"
else
	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	arch=$(uname -m)
	case "$arch" in
	x86_64 | amd64) arch="x64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*)
		echo "FAIL  unsupported arch $arch"
		exit 1
		;;
	esac
	case "$os" in
	linux | darwin) ;;
	*)
		echo "FAIL  unsupported OS $os"
		exit 1
		;;
	esac
	asset="gitleaks_${version}_${os}_${arch}.tar.gz"
	url="https://github.com/gitleaks/gitleaks/releases/download/v${version}/${asset}"
	workdir=$(mktemp -d)
	echo "    download $asset"
	if ! curl -sSfL "$url" | tar -xz -C "$workdir" gitleaks; then
		echo "FAIL  could not download gitleaks from $url"
		exit 1
	fi
	bin="$workdir/gitleaks"
	chmod +x "$bin"
fi

# Scan tracked files only so a local gitignored .env does not fail the job.
scan_root=$(mktemp -d)
python3 - "$scan_root" <<'PY'
import shutil
import subprocess
import sys
from pathlib import Path

dest_root = Path(sys.argv[1])
files = subprocess.check_output(["git", "ls-files"], text=True).splitlines()
for rel in files:
    src = Path(rel)
    if not src.is_file():
        continue
    dest = dest_root / rel
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dest)
print(f"OK    scanning {len(files)} tracked files")
PY

if ! "$bin" detect \
	--source "$scan_root" \
	--no-git \
	--redact \
	--config .gitleaks.toml \
	--exit-code 1; then
	echo "==> Committed secrets: FAIL"
	echo "    Remove the secret or add a documented path to .gitleaks.toml."
	exit 1
fi

echo "==> Committed secrets: PASS"

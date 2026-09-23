#!/usr/bin/env bash
# Format-check and validate the install stacks next to this directory.
# Does not call AWS or GCP. terraform init only downloads providers.

set -euo pipefail

terraform_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$terraform_root"

if ! command -v terraform >/dev/null 2>&1; then
  echo "terraform is not on PATH. From release/terraform, run make check." >&2
  exit 1
fi

export TF_IN_AUTOMATION=1

echo "==> terraform fmt"
terraform fmt -check -recursive -diff .

found=0
for dir in */; do
  if ! compgen -G "${dir}*.tf" >/dev/null; then
    continue
  fi
  found=1
  echo "==> terraform validate (${dir%/})"
  terraform -chdir="$dir" init -backend=false -input=false -no-color
  terraform -chdir="$dir" validate -no-color
done

if [[ "$found" -eq 0 ]]; then
  echo "No Terraform stacks found under ${terraform_root}." >&2
  exit 1
fi

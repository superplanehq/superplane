#!/usr/bin/env bash

set -euo pipefail

base_url="$1"

if [ -z "${base_url}" ]; then
  echo "base-url is required" >&2
  exit 1
fi

cat > /home/app/Caddyfile <<EOF
${base_url} {
  tls internal
  reverse_proxy 127.0.0.1:8000
}
EOF

sudo caddy reload --config /home/app/Caddyfile

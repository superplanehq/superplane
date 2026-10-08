#!/usr/bin/env bash

set -euxo pipefail

# The runner systemd unit runs as ubuntu. GCE Ubuntu images do not create
# that account.
if ! id ubuntu >/dev/null 2>&1; then
  useradd --create-home --shell /bin/bash ubuntu
fi

# Fleet Manager sends the runner bootstrap as user-data metadata.
cloud-init --version

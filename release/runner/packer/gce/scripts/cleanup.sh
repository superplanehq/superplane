#!/usr/bin/env bash

set -euxo pipefail

# New instances must run cloud-init again so they execute their own
# user-data.
cloud-init clean --logs --seed

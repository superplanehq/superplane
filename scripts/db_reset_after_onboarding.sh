#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Local-only. Wipes SuperPlane runtime data in superplane_dev so the
# environment looks like onboarding just finished: org, workspace, GitHub,
# apps, lines, and intakes stay. Tasks, runs, usage, subscription, and
# credits reset. Intake then reseeds from GitHub.
# This does not change Polar sandbox customers or subscriptions.

DB_NAME="${1:-superplane_dev}"

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "db.reset.after.onboarding only runs against superplane_dev." >&2
  exit 1
fi

export DB_NAME

exec go run ./scripts/db_reset_after_onboarding

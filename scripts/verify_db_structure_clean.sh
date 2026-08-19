#!/bin/bash

# Verify that the database structure files are clean, meaning
# they contain only the expected schema after running all existing
# migrations.
#
# This script is run in CI to ensure that no changes to the database
# structure are introduced without proper migration. Or that the
# structure.sql files are not out of date.
#

function red() {
  echo -e "\033[0;31m$1\033[0m"
}

function diff() {
  # Ignore the diff header lines
  # Only show actual schema changes
  #
  # Note: the volatile "Dumped by pg_dump version"/"Dumped from database
  # version" comment lines are stripped when db/structure.sql is generated
  # (see scripts/db_migrate.sh), so they should never show up here. If they
  # do reappear, that means the generation-time fix regressed and this
  # check should legitimately fail.
  git diff --unified=0 -- db/structure.sql | grep -E '^\+|^\-' | grep -v "/db/structure.sql"
}

lineCount=$(diff | wc -l)

if [[ $lineCount -gt 0 ]]; then
  diff

  red ""
  red "Unexpected changes detected in database structure files!"
  red ""
  red "Please ensure that the database structure files are up to date and commit the changes."
  red ""
  red "You can update the structure.sql file by running:"
  red "    make db.migrate.all"
  red ""
  red "And then commit the updated db/structure.sql file."
  red ""

  exit 1
else
  exit 0
fi

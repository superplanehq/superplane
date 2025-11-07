#!/bin/bash

# Verify that the database structure.sql file is clean, meaning
# it contains only the expected scheme after running all existing 
# migrations.
#
# This script is run in CI to ensure that no changes to the database
# structure are introduced without proper migration. Or that the
# structure.sql file is not out of date.
#

function red() {
  echo -e "\033[0;31m$1\033[0m"
}

git diff --exit-code -- db/structure.sql

if [ $? != 0 ]; then
  red ""
  red "Unexpected changes detected in db/structure.sql!"
  red ""
  red "Please ensure that the database structure is up to date and commit the changes."
  red ""
  red "You can update the structure.sql file by running:"
  red "    make db.migrate.all"
  red ""
  red "And then commit the updated db/structure.sql file."
  red ""

  exit 1
fi
  exit 0
fi

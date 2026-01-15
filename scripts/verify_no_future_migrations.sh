#!/bin/bash

# Verify that no database migration files have timestamps from the future.
#
# This script is run in CI to ensure that migrations created by automated
# tools or agents don't accidentally have future timestamps, which could
# cause issues with migration ordering.
#

function red() {
  echo -e "\033[0;31m$1\033[0m"
}

function yellow() {
  echo -e "\033[0;33m$1\033[0m"
}

function green() {
  echo -e "\033[0;32m$1\033[0m"
}

# Get current timestamp in the same format as migration files (YYYYMMDDHHMMSS)
current_timestamp=$(date -u +%Y%m%d%H%M%S)

yellow "Checking for migrations with future timestamps..."
yellow "Current timestamp (UTC): $current_timestamp"
yellow ""

# Find all migration files and extract their timestamps
migration_dir="db/migrations"
future_migrations=()

for file in "$migration_dir"/*.sql; do
  if [ -f "$file" ]; then
    # Extract just the filename
    filename=$(basename "$file")
    
    # Extract the timestamp (first 14 characters: YYYYMMDDHHMMSS)
    timestamp="${filename:0:14}"
    
    # Check if this is a valid timestamp (14 digits)
    if [[ $timestamp =~ ^[0-9]{14}$ ]]; then
      # Compare timestamps as numbers
      if [ "$timestamp" -gt "$current_timestamp" ]; then
        future_migrations+=("$filename (timestamp: $timestamp)")
      fi
    fi
  fi
done

# Report results
if [ ${#future_migrations[@]} -gt 0 ]; then
  red ""
  red "ERROR: Found migration(s) with timestamps from the future!"
  red ""
  red "The following migration files have timestamps that are in the future:"
  red ""
  
  for migration in "${future_migrations[@]}"; do
    red "  - $migration"
  done
  
  red ""
  red "This can happen when migrations are created by automated tools or agents"
  red "with incorrect system time. Please recreate these migrations with the"
  red "correct timestamp using:"
  red "    make db.migration.create NAME=<descriptive-name>"
  red ""
  
  exit 1
else
  green "✓ All migrations have valid timestamps (not from the future)"
  exit 0
fi

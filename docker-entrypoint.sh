#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ -z $DB_PASSWORD ];         then echo "DB username not set" && exit 1; fi
if [ -z $DB_HOST ];             then echo "DB host not set"     && exit 1; fi
if [ -z $DB_PORT ];             then echo "DB port not set"     && exit 1; fi
if [ -z $DB_USERNAME ];         then echo "DB username not set" && exit 1; fi
if [ -z $DB_NAME ];             then echo "DB name not set"     && exit 1; fi
if [ -z $APPLICATION_NAME ];    then echo "Application name not set"     && exit 1; fi

[ "${POSTGRES_DB_SSL}" = "true" ] && export PGSSLMODE=require || export PGSSLMODE=disable

echo "Creating DB..."
PGPASSWORD=${DB_PASSWORD} createdb -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USERNAME} ${DB_NAME} || true

echo "Migrating DB..."
DB_URL="postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${PGSSLMODE}"
migrate -source file:///app/db/migrations -database ${DB_URL} up

echo "Starting server..."
/app/build/superplane

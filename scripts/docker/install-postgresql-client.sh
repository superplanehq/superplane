#!/bin/sh

set -e
set -o pipefail

echo "Installing PostgreSQL Client 17"

apt-get update -y
apt-get install --no-install-recommends -y ca-certificates curl gnupg

echo "deb http://apt.postgresql.org/pub/repos/apt noble-pgdg main" > /etc/apt/sources.list.d/pgdg.list
curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg

apt-get update -y
apt-get install --no-install-recommends -y postgresql-client-17
apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*
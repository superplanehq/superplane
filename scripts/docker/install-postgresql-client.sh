#!/bin/sh

set -e
set -o pipefail

echo "Installing PostgreSQL Client 17"

apt-get update -y
apt-get install --no-install-recommends -y ca-certificates unzip curl gnupg lsb-release libc-bin libc6

sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg

apt-get update -y
apt-get install --no-install-recommends -y postgresql-client-17
apt-get clean && rm -f /var/lib/apt/lists/*_*
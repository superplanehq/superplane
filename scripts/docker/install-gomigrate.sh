#!/bin/sh

set -e
set -o pipefail

echo "Installing Go Migrate v4.18.2"

ARCH=$(dpkg --print-architecture)

if [ "$ARCH" = "arm64" ]; then
  MIGRATE_ARCH="arm64"
else
  MIGRATE_ARCH="amd64"
fi

# Download to a file first. A failed download then stops the script
# before tar reads a partial or non-gzip response.
curl --retry 8 --retry-delay 5 --retry-max-time 180 --retry-all-errors -fsSL \
  "https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-${MIGRATE_ARCH}.tar.gz" \
  -o migrate.tar.gz
tar xzvf migrate.tar.gz
rm -f migrate.tar.gz

mv ./migrate /usr/bin/migrate
chmod +x /usr/bin/migrate
rm -f ./LICENSE ./README.md
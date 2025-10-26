#!/bin/sh

set -e
set -o pipefail

echo "Installing Go Migrate v4.18.2"

ARCH=$(dpkg --print-architecture)

if [ "$ARCH" = "arm64" ]; then
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-arm64.tar.gz | tar xvz;
else
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | tar xvz;
fi

mv /tmp/migrate /usr/bin/migrate
chmod +x /usr/bin/migrate
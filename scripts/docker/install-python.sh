#!/bin/sh

set -e
set -o pipefail

echo "Installing Python 3.12 from Ubuntu repositories"

apt-get update -y
apt-get install -y --no-install-recommends python3 python3-venv python3-dev

ln -sf /usr/bin/python3 /usr/local/bin/python3
ln -sf /usr/bin/python3 /usr/local/bin/python

apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* /tmp/*

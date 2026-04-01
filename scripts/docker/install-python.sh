#!/bin/sh

set -e
set -o pipefail

echo "Installing Python 3.13 from deadsnakes PPA"

apt-get update -y
apt-get install -y --no-install-recommends software-properties-common

add-apt-repository -y ppa:deadsnakes/ppa
apt-get update -y
apt-get install -y --no-install-recommends python3.13 python3.13-venv python3.13-dev

ln -sf /usr/bin/python3.13 /usr/local/bin/python3
ln -sf /usr/bin/python3.13 /usr/local/bin/python

apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* /tmp/*

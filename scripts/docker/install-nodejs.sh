#!/bin/sh

set -e
set -o pipefail

echo "Installing Node.js 22.x from NodeSource repository"

apt-get update -y
apt-get install --no-install-recommends -y curl gnupg

mkdir -p /etc/apt/keyrings
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list

apt-get update -y
apt-get install -y nodejs
apt-get clean && rm -f /var/lib/apt/lists/*_*

node -v
npm -v
#!/usr/bin/env bash

set -euxo pipefail
export DEBIAN_FRONTEND=noninteractive

: "${CLAUDE_CODE_VERSION:?CLAUDE_CODE_VERSION is required}"
: "${OPENCODE_VERSION:?OPENCODE_VERSION is required}"
: "${CODEX_VERSION:?CODEX_VERSION is required}"
: "${PLAYWRIGHT_VERSION:?PLAYWRIGHT_VERSION is required}"

architecture="$(dpkg --print-architecture)"
case "${architecture}" in
  amd64 | arm64) ;;
  *)
    echo "Unsupported architecture: ${architecture}" >&2
    exit 1
    ;;
esac

apt-get update -qy
apt-get install -qy --no-install-recommends \
  bash \
  build-essential \
  ca-certificates \
  cmake \
  coreutils \
  curl \
  ffmpeg \
  git \
  gnupg \
  jq \
  libgomp1 \
  lsb-release \
  make \
  openssl \
  python3 \
  python3-pip \
  python3-venv \
  ripgrep \
  rsync \
  tar \
  unzip \
  wget \
  zip

# Ubuntu installs fd as fdfind. Runner tasks expect fd to be on PATH.
apt-get install -qy --no-install-recommends fd-find
ln -sf "$(command -v fdfind)" /usr/local/bin/fd

yq_architecture="${architecture}"
yq_version="v4.53.3"
curl --fail --location --silent --show-error \
  "https://github.com/mikefarah/yq/releases/download/${yq_version}/yq_linux_${yq_architecture}" \
  --output /usr/local/bin/yq
chmod 0755 /usr/local/bin/yq

install -d -m 0755 /etc/apt/keyrings
curl --fail --location --silent --show-error \
  https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  --output /etc/apt/keyrings/github-cli.gpg
chmod a+r /etc/apt/keyrings/github-cli.gpg
echo "deb [arch=${architecture} signed-by=/etc/apt/keyrings/github-cli.gpg] https://cli.github.com/packages stable main" \
  > /etc/apt/sources.list.d/github-cli.list

curl --fail --location --silent --show-error \
  https://download.docker.com/linux/ubuntu/gpg |
  gpg --dearmor --yes --output /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=${architecture} signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
  > /etc/apt/sources.list.d/docker.list

apt-get update -qy
apt-get install -qy --no-install-recommends \
  containerd.io \
  docker-buildx-plugin \
  docker-ce \
  docker-ce-cli \
  docker-compose-plugin \
  gh
systemctl enable docker
usermod -aG docker ubuntu

curl --fail --location --silent --show-error \
  https://deb.nodesource.com/setup_22.x \
  --output /tmp/nodesource-setup.sh
bash /tmp/nodesource-setup.sh
apt-get install -qy --no-install-recommends nodejs

npm install --global \
  "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
  "opencode-ai@${OPENCODE_VERSION}" \
  "@openai/codex@${CODEX_VERSION}" \
  "playwright@${PLAYWRIGHT_VERSION}"

export PLAYWRIGHT_BROWSERS_PATH=/opt/ms-playwright
playwright install --with-deps chromium
chmod -R a+rX "${PLAYWRIGHT_BROWSERS_PATH}"

# Playwright and the global npm installs leave large caches. Remove them before
# the AWS CLI archive is unpacked to keep the image bake below its disk limit.
npm cache clean --force
apt-get clean
rm -rf /var/lib/apt/lists/*

aws_architecture="x86_64"
if [ "${architecture}" = "arm64" ]; then
  aws_architecture="aarch64"
fi
curl --fail --location --silent --show-error \
  "https://awscli.amazonaws.com/awscli-exe-linux-${aws_architecture}.zip" \
  --output /tmp/awscliv2.zip
unzip -q /tmp/awscliv2.zip -d /tmp
/tmp/aws/install

curl --fail --location --silent --show-error \
  "https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/${architecture}/latest/amazon-cloudwatch-agent.deb" \
  --output /tmp/amazon-cloudwatch-agent.deb
dpkg -i /tmp/amazon-cloudwatch-agent.deb

git --version
gh --version
jq --version
yq --version
rg --version
fd --version
make --version
docker --version
docker compose version
node --version
npm --version
claude --version
opencode --version
codex --version
playwright --version
aws --version

ffmpeg -version >/dev/null
ffprobe -version >/dev/null
PLAYWRIGHT_BROWSERS_PATH=/opt/ms-playwright \
  playwright screenshot about:blank /tmp/playwright-smoke.png
test -s /tmp/playwright-smoke.png

install -d -m 0755 /etc/systemd/system/superplane-runner.service.d
cat > /etc/systemd/system/superplane-runner.service.d/10-runner-ami.conf <<'EOF'
[Service]
Environment=PLAYWRIGHT_BROWSERS_PATH=/opt/ms-playwright
EOF

apt-get clean
rm -rf \
  /tmp/amazon-cloudwatch-agent.deb \
  /tmp/aws \
  /tmp/awscliv2.zip \
  /tmp/nodesource-setup.sh \
  /tmp/playwright-smoke.png \
  /var/lib/apt/lists/*

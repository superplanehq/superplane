#!/bin/bash
# packer/scripts/install.sh
# Bake all static dependencies into the AMI.
# The runner binary and per-instance config are installed at launch via userdata.
set -euxo pipefail
export DEBIAN_FRONTEND=noninteractive

#########################################################
# Packages
#########################################################

apt-get update -qy
apt-get install -qy \
  bash \
  ca-certificates \
  curl \
  wget \
  unzip \
  zip \
  rsync \
  build-essential \
  gnupg \
  lsb-release \
  git \
  make \
  jq \
  ripgrep \
  fd-find \
  python3 \
  python3-pip \
  python3-venv

ARCH=$(dpkg --print-architecture)  # amd64 or arm64

# Ubuntu packages fd as fdfind; agents and docs expect `fd`.
ln -sf "$(command -v fdfind)" /usr/local/bin/fd

# yq (mikefarah) — pin version; apt's yq is a different tool.
YQ_VERSION=v4.53.3
YQ_ARCH=$([ "$ARCH" = "arm64" ] && echo "arm64" || echo "amd64")
curl -fsSL "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${YQ_ARCH}" \
  -o /usr/local/bin/yq
chmod 755 /usr/local/bin/yq

# GitHub CLI (official apt repo — Ubuntu's gh package lags / can be broken)
mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  -o /etc/apt/keyrings/githubcli-archive-keyring.gpg
chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] \
  https://cli.github.com/packages stable main" \
  > /etc/apt/sources.list.d/github-cli.list
apt-get update -qy
apt-get install -qy gh

git --version
gh --version
jq --version
yq --version
rg --version
fd --version
make --version
# Avoid `cmd | head` under pipefail — SIGPIPE yields exit 141.
command -v zip
command -v rsync
command -v wget
zip -v >/dev/null
rsync --version >/dev/null
wget --version >/dev/null

# Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update -qy
apt-get install -qy docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
usermod -aG docker ubuntu

# Node.js 22 (NodeSource) — needed for javascript_script host-mode tasks.
curl -fsSL https://deb.nodesource.com/setup_22.x -o /tmp/nodesource_setup.sh
bash /tmp/nodesource_setup.sh
apt-get install -qy nodejs

# Coding agent CLIs (host-mode tasks; binaries land on PATH for ubuntu).
# Pin versions — bump deliberately when rebuilding AMIs.
CLAUDE_CODE_VERSION=2.1.212
OPENCODE_VERSION=1.18.3
CODEX_VERSION=0.144.5
npm install -g "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}"
npm install -g "opencode-ai@${OPENCODE_VERSION}"
npm install -g "@openai/codex@${CODEX_VERSION}"
claude --version
opencode --version
codex --version

# AWS CLI v2
AWS_CLI_ARCH=$([ "$ARCH" = "arm64" ] && echo "aarch64" || echo "x86_64")
curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${AWS_CLI_ARCH}.zip" \
  -o /tmp/awscliv2.zip
unzip -q /tmp/awscliv2.zip -d /tmp
/tmp/aws/install --update
rm -rf /tmp/aws /tmp/awscliv2.zip

# CloudWatch agent
CWA_ARCH="${ARCH/amd64/amd64}"  # ubuntu package uses amd64/arm64 directly
curl -fsSL "https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/${CWA_ARCH}/latest/amazon-cloudwatch-agent.deb" \
  -o /tmp/cwa.deb
dpkg -i /tmp/cwa.deb
rm /tmp/cwa.deb

#########################################################
# Systemd service (unit file only — runner binary + EnvironmentFile written by userdata)
#########################################################

# Create a placeholder EnvironmentFile so the unit can be enabled now.
# The real values are written by userdata.sh.tmpl at instance launch.
cat > /etc/default/superplane-runner <<'ENVEOF'
# Populated by userdata at instance launch.
TASK_BROKER_URL=
RUNNER_FLEET_ID=
RUNNER_ID=
RUNNER_SHELL=/bin/bash
RUNNER_HEALTH_ADDR=0.0.0.0:9090
AUTH_TOKEN=
RUNNER_TERMINATE_AFTER_EACH_TASK=true
ENVEOF
chmod 644 /etc/default/superplane-runner

cat > /etc/systemd/system/superplane-runner.service <<'UNITEOF'
[Unit]
Description=Superplane runner (host process; host-mode tasks run on Ubuntu)
After=network-online.target docker.service
Wants=docker.service

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
EnvironmentFile=/etc/default/superplane-runner
PrivateUsers=no
RestrictNamespaces=no
MemoryDenyWriteExecute=no
RestrictRealtime=no
RestrictSUIDSGID=no
LockPersonality=no
Restart=no
ExecStart=/usr/local/bin/runner
StandardOutput=append:/var/log/superplane-runner.log
StandardError=append:/var/log/superplane-runner.log

[Install]
WantedBy=multi-user.target
UNITEOF

touch /var/log/superplane-runner.log
chown ubuntu:ubuntu /var/log/superplane-runner.log
chmod 644 /var/log/superplane-runner.log

systemctl daemon-reload
# Enable but don't start — userdata must populate EnvironmentFile first.
systemctl enable superplane-runner.service

#########################################################
# Clean up
#########################################################

apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/nodesource_setup.sh

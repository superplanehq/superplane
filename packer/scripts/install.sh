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
  unzip \
  build-essential \
  gnupg \
  lsb-release

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

# AWS CLI v2
ARCH=$(dpkg --print-architecture)  # amd64 or arm64
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

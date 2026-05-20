#!/usr/bin/env bash
# Terminate all active superplane-managed EC2 runner instances via the fleet-manager host.
# Fleet-manager will reconcile and provision fresh ones with the latest runner binary.
#
# Usage (from Semaphore or locally):
#   SSH_KEY=~/.ssh/deploy.pem bash scripts/recycle-runners.sh
set -euo pipefail

SSH_KEY="${SSH_KEY:-$HOME/.ssh/deploy.pem}"
FLEET_HOST="44.220.93.243"

echo "Fetching active managed runner instances from fleet-manager host..."

IDS=$(ssh -o StrictHostKeyChecking=accept-new -i "$SSH_KEY" "ubuntu@$FLEET_HOST" bash << 'REMOTE'
set -euo pipefail
TOKEN=$(curl -sf -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
CREDS=$(curl -sf -H "X-aws-ec2-metadata-token: $TOKEN" \
  "http://169.254.169.254/latest/meta-data/iam/security-credentials/fleet-manager-ec2-role")

export AWS_ACCESS_KEY_ID=$(echo "$CREDS"    | python3 -c "import sys,json; print(json.load(sys.stdin)['AccessKeyId'])")
export AWS_SECRET_ACCESS_KEY=$(echo "$CREDS" | python3 -c "import sys,json; print(json.load(sys.stdin)['SecretAccessKey'])")
export AWS_SESSION_TOKEN=$(echo "$CREDS"     | python3 -c "import sys,json; print(json.load(sys.stdin)['Token'])")
export AWS_DEFAULT_REGION=us-east-1

python3 - << 'PYEOF'
import boto3, os
ec2 = boto3.client("ec2", region_name="us-east-1",
    aws_access_key_id=os.environ["AWS_ACCESS_KEY_ID"],
    aws_secret_access_key=os.environ["AWS_SECRET_ACCESS_KEY"],
    aws_session_token=os.environ["AWS_SESSION_TOKEN"])
r = ec2.describe_instances(Filters=[
    {"Name": "tag:superplane_managed_runner", "Values": ["true"]},
    {"Name": "instance-state-name",           "Values": ["pending", "running"]},
])
ids = [i["InstanceId"] for res in r["Reservations"] for i in res["Instances"]]
if ids:
    ec2.terminate_instances(InstanceIds=ids)
print(" ".join(ids))
PYEOF
REMOTE
)

if [ -n "$IDS" ]; then
  echo "Terminated runner instances: $IDS"
else
  echo "No active runner instances found — nothing to terminate."
fi
echo "Fleet-manager will provision fresh instances on next reconcile (up to 60s)."

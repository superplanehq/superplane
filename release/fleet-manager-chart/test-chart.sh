#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/helm"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

render_provider() {
  local provider="$1"
  helm template fleet-manager "${CHART_DIR}" \
    --namespace superplane \
    --values "${SCRIPT_DIR}/testdata/${provider}.yaml" \
    >"${TMP_DIR}/${provider}.yaml"
}

helm lint "${CHART_DIR}" --values "${SCRIPT_DIR}/testdata/aws.yaml"

helm package "${CHART_DIR}" \
  --version 1.2.3 \
  --app-version v1.2.3 \
  --destination "${TMP_DIR}" \
  >/dev/null
helm template fleet-manager "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" \
  --namespace superplane \
  --values "${SCRIPT_DIR}/testdata/aws.yaml" \
  >"${TMP_DIR}/packaged.yaml"
grep -q 'image: "ghcr.io/superplanehq/fleet-manager:v1.2.3"' "${TMP_DIR}/packaged.yaml"

for provider in aws gcp azure; do
  render_provider "${provider}"
  grep -q 'helm.sh/hook: post-install,post-upgrade' "${TMP_DIR}/${provider}.yaml"
  grep -q 'automountServiceAccountToken: false' "${TMP_DIR}/${provider}.yaml"
  grep -q 'readOnlyRootFilesystem: true' "${TMP_DIR}/${provider}.yaml"
  grep -q 'checksum/config:' "${TMP_DIR}/${provider}.yaml"
  grep -q "provider: ${provider}" "${TMP_DIR}/${provider}.yaml"
  ruby -ryaml -e '
    documents = YAML.load_stream(File.read(ARGV.fetch(0))).compact
    secrets = documents.select { |document| document["kind"] == "Secret" }
    config_maps = documents.select { |document| document["kind"] == "ConfigMap" }
    deployment = documents.find { |document| document["kind"] == "Deployment" }
    service_account = documents.find { |document| document["kind"] == "ServiceAccount" }
    startup_job = documents.find { |document| document["kind"] == "Job" }
    abort "expected one configuration Secret" unless secrets.length == 1
    abort "wrong configuration Secret name" unless secrets.first.dig("metadata", "name") == "fleet-manager-config"
    abort "configuration ConfigMap still exists" unless config_maps.empty?
    abort "missing Fleet Manager Deployment" unless deployment
    abort "missing Fleet Manager ServiceAccount" unless service_account
    abort "missing startup Job" unless startup_job
    abort "startup Job mounts a service-account token" unless startup_job.dig("spec", "template", "spec", "automountServiceAccountToken") == false
    abort "Fleet Manager disables workload identity tokens" if deployment.dig("spec", "template", "spec", "automountServiceAccountToken") == false
    abort "service account disables workload identity tokens" if service_account["automountServiceAccountToken"] == false
    config = YAML.safe_load(secrets.first.dig("stringData", "fleet-manager.yaml"))
    abort "configuration token is missing" unless config["installationAdminToken"] == "fleet-manager-personal-token"
    config.fetch("fleets").each do |fleet|
      abort "runtime fleet is missing runnerVersion" unless fleet.key?("runnerVersion")
      abort "runtime fleet is missing spec" unless fleet.key?("spec")
    end
    [deployment, startup_job].each do |workload|
      pod_spec = workload.dig("spec", "template", "spec")
      container = pod_spec.fetch("containers").first
      env_names = container.fetch("env").map { |entry| entry.fetch("name") }
      abort "workload uses INSTALLATION_ADMIN_TOKEN" if env_names.include?("INSTALLATION_ADMIN_TOKEN")
      abort "workload does not set FLEET_MANAGER_CONFIG_FILE" unless env_names.include?("FLEET_MANAGER_CONFIG_FILE")
      secret_name = pod_spec.fetch("volumes").first.dig("secret", "secretName")
      abort "workload does not mount fleet-manager-config" unless secret_name == "fleet-manager-config"
    end
    job_container = startup_job.dig("spec", "template", "spec", "containers").first
    deployment_container = deployment.dig("spec", "template", "spec", "containers").first
    abort "startup Job uses a different image" unless job_container["image"] == deployment_container["image"]
    abort "startup Job does not run sync-fleets" unless job_container["args"] == ["sync-fleets"]
  ' "${TMP_DIR}/${provider}.yaml"
done

grep -q 'eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/fleet-manager' "${TMP_DIR}/aws.yaml"
grep -q 'iam.gke.io/gcp-service-account: fleet-manager@example.iam.gserviceaccount.com' "${TMP_DIR}/gcp.yaml"
grep -q 'azure.workload.identity/client-id: 00000000-0000-0000-0000-000000000001' "${TMP_DIR}/azure.yaml"
grep -q 'azure.workload.identity/use: "true"' "${TMP_DIR}/azure.yaml"

if helm template fleet-manager "${CHART_DIR}" \
  --set manager.id=invalid.id \
  --set superplane.url=https://superplane.example.com \
  --set installationAdminToken=token \
  --set fleets[0].id=invalid.id \
  --set fleets[0].runnerVersion=v0.1.0 \
  --set fleets[0].provider=aws \
  --set fleets[0].spec.operatingSystem=linux \
  --set fleets[0].spec.architecture=amd64 \
  --set fleets[0].spec.cpuMillicores=2000 \
  --set fleets[0].spec.memoryMb=8192 \
  --set fleets[0].spec.diskGb=30 \
  --set fleets[0].aws.region=us-east-1 \
  --set fleets[0].aws.ami=ami-test \
  --set fleets[0].aws.architecture=amd64 \
  --set fleets[0].aws.subnetIds[0]=subnet-test \
  --set fleets[0].aws.securityGroupIds[0]=sg-test \
  >/dev/null 2>&1; then
  echo "Invalid fleet IDs must fail schema validation" >&2
  exit 1
fi

echo "Fleet Manager chart tests passed."

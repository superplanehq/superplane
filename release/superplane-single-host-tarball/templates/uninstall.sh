#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./uninstall.sh [--purge] [--force] [--yes]

Stop SuperPlane and remove its containers and network.
Keep data volumes and configuration files by default.

  --purge  Delete the installation's Docker data volumes. Back up data first.
  --force  Permit interruption of runner tasks. Required when runners are running.
  --yes    Skip interactive confirmation. This does not imply --force or --purge.

Configuration files and OIDC keys remain in the installation directory.
Run ./install.sh to start the installation again.
EOF
}

fail() { echo "Error: $*" >&2; exit 1; }

purge=no
force=no
confirmed=no
for argument in "$@"; do
  case "${argument}" in
    --purge) purge=yes ;;
    --force) force=yes ;;
    --yes) confirmed=yes ;;
    --help | -h) usage; exit 0 ;;
    *) usage >&2; fail "Unknown option: ${argument}" ;;
  esac
done

install_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
cd "${install_dir}"
[[ -f docker-compose.yml ]] || fail "docker-compose.yml is missing from the installation directory."
command -v docker >/dev/null || fail "Docker is required."
docker compose version >/dev/null
docker info >/dev/null

# The bundle declares its project explicitly. Ignore ambient Compose overrides.
project="$(awk '/^name:/ && NF == 2 { print $2 }' docker-compose.yml)"
[[ "${project}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || fail "The Compose project name is missing or invalid."
compose=(docker compose --project-directory "${install_dir}" -p "${project}" -f "${install_dir}/docker-compose.yml")
network="${project}_default"
network_id=""

project_containers() {
  docker ps -aq --no-trunc --filter "label=com.docker.compose.project=${project}"
}

project_volumes() {
  docker volume ls -q --filter "label=com.docker.compose.project=${project}"
}

verify_volumes() {
  local id volume labels owner key mounted declared
  declared="$("${compose[@]}" config --volumes)"
  mounted=""
  while read -r id; do
    [[ -n "${id}" ]] || continue
    mounted+="$(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{"\n"}}{{end}}{{end}}' "${id}")"$'\n'
  done <<< "$(project_containers)"
  volumes="$(printf '%s\n%s\n' "$(project_volumes)" "${mounted}" | sed '/^$/d' | sort -u)"
  while read -r volume; do
    [[ -n "${volume}" ]] || continue
    labels="$(docker volume inspect --format '{{with .Labels}}{{index . "com.docker.compose.project"}} {{index . "com.docker.compose.volume"}}{{end}}' "${volume}")"
    [[ -n "${labels}" ]] ||
      fail "Volume ${volume} does not have verified ownership for this installation. Nothing was removed."
    read -r owner key <<< "${labels}"
    [[ "${owner}" == "${project}" && -n "${key}" ]] &&
      grep -Fxq -- "${key}" <<< "${declared}" ||
      fail "Volume ${volume} does not have verified ownership for this installation. Nothing was removed."
  done <<< "${volumes}"
}

verify_containers() {
  local id directory files ids
  ids="$(project_containers)"
  while read -r id; do
    [[ -n "${id}" ]] || continue
    directory="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "${id}")"
    files="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project.config_files"}}' "${id}")"
    [[ "${directory}" == "${install_dir}" && "${files}" == "${install_dir}/docker-compose.yml" ]] ||
      fail "Project ${project} contains a container from another installation: ${id}. Nothing was removed."
  done <<< "${ids}"
}

verify_network() {
  local found labels
  found="$(docker network ls -q --no-trunc --filter "name=^${network}$")"
  [[ -n "${found}" ]] || return 0
  labels="$(docker network inspect --format '{{index .Labels "com.docker.compose.project"}} {{index .Labels "com.docker.compose.network"}}' "${found}")"
  [[ "${labels}" == "${project} default" ]] || fail "Network ${network} does not belong to this Compose project."
  if [[ -n "${network_id}" && "${network_id}" != "${found}" ]]; then
    fail "The installation network changed. Run uninstall.sh again after you check its ownership."
  fi
  network_id="${found}"
}

runner_containers() {
  [[ -n "${network_id}" ]] || return 0
  docker ps -aq --no-trunc --filter "network=${network_id}" --filter label=superplane.managed-runner=true
}

verify_runner() {
  local id="$1" identity runner_id fleet name
  identity="$(docker inspect --format '{{index .Config.Labels "superplane.runner-id"}} {{index .Config.Labels "superplane.fleet-id"}} {{.Name}}' "${id}")"
  read -r runner_id fleet name <<< "${identity}"
  [[ "${runner_id}" =~ ^[a-zA-Z0-9-]+$ && "${fleet}" =~ ^[a-zA-Z0-9_-]+$ && "${name}" == "/superplane-runner-${runner_id}" ]] ||
    fail "Runner ownership is incomplete for container ${id}. Refusing to remove it."
}

load_runner_api_token() {
  local file key
  runner_api_token=""
  for file in superplane.env fleet-manager.env; do
    [[ -f "${file}" ]] || continue
    for key in FLEET_MANAGER_API_TOKEN INSTALLATION_ADMIN_TOKEN; do
      runner_api_token="$(awk -v key="${key}" 'index($0, key "=") == 1 { sub(/^[^=]*=/, ""); print }' "${file}")"
      [[ -z "${runner_api_token}" ]] || break
    done
    [[ -z "${runner_api_token}" ]] || break
  done
  [[ "${runner_api_token}" =~ ^[a-zA-Z0-9._~+/-]+=*$ ]] ||
    fail "No valid runner API token found. Restore the configured token in superplane.env or fleet-manager.env before preserving removal."
}

retire_runner() {
  local id="$1" deadline="$2" identity runner_id fleet response status state remaining timeout request_query
  local method=DELETE query='?states=pending&states=idle&states=terminated'
  identity="$(docker inspect --format '{{index .Config.Labels "superplane.runner-id"}} {{index .Config.Labels "superplane.fleet-id"}}' "${id}")"
  read -r runner_id fleet <<< "${identity}"
  [[ "${force}" != yes ]] || query+='&states=busy'

  while true; do
    remaining=$((deadline - SECONDS))
    [[ "${remaining}" -gt 0 ]] ||
      fail "Runner ${runner_id} did not terminate within 60 seconds. No runner containers were removed. Run ./install.sh to resume, then check its task before you retry."
    timeout=10
    [[ "${remaining}" -ge "${timeout}" ]] || timeout="${remaining}"
    request_query=""
    [[ "${method}" != DELETE ]] || request_query="${query}"
    if ! response="$(printf 'Authorization: Bearer %s\n' "${runner_api_token}" |
      "${compose[@]}" exec -T app curl --silent --show-error --connect-timeout 5 --max-time "${timeout}" \
        --header @- --request "${method}" --write-out $'\n%{http_code}' \
        "http://127.0.0.1:8000/admin/api/installation/fleets/${fleet}/runners/${runner_id}${request_query}")"; then
      fail "Cannot reach the app API to retire runner ${runner_id}. No runner containers were removed. Run ./install.sh to resume before you retry."
    fi
    status="${response##*$'\n'}"
    [[ "${status}" == 200 ]] ||
      fail "The app API did not retire runner ${runner_id} (HTTP ${status}). A busy runner requires --force and must finish cancellation. No runner containers were removed. Run ./install.sh to resume before you retry."
    # Require one exact state field. An ambiguous response must not authorize
    # physical deletion while the database still counts the runner as available.
    state="$(printf '%s' "${response%$'\n'*}" |
      grep -oE '"state"[[:space:]]*:[[:space:]]*"[^"]*"' |
      sed -E 's/^"state"[[:space:]]*:[[:space:]]*"([^"]*)"$/\1/' || true)"
    case "${state}" in
      terminated) return 0 ;;
      busy)
        [[ "${force}" == yes ]] || fail "Runner ${runner_id} is busy. Use --force to permit cancellation, or ./install.sh to resume."
        method=GET
        ;;
      pending | idle) method=DELETE ;;
      *) fail "The app API returned an unrecognized runner state. No runner containers were removed. Run ./install.sh to resume before you retry." ;;
    esac
    remaining=$((deadline - SECONDS))
    [[ "${remaining}" -le 0 ]] || sleep "$((remaining < 2 ? remaining : 2))"
  done
}

verify_containers
verify_network
containers="$(project_containers)"
runners="$(runner_containers)"
observed_runners="${runners}"
volumes="$(project_volumes)"

# Compose volumes and networks do not record the installation path.
# A live Compose container supplies that ownership evidence. Never infer it
# from the globally shared managed-runner label or a project name alone.
if [[ -z "${containers}" && -n "${network_id}" ]]; then
  fail "Network ownership cannot be verified without this installation's Compose containers. Run ./install.sh first."
fi
if [[ "${purge}" == yes && -z "${containers}" && -n "${volumes}" ]]; then
  fail "Volume ownership cannot be verified. Run ./install.sh before you use --purge."
fi
if [[ -z "${containers}" && -z "${network_id}" ]]; then
  echo "No project containers or network found. No changes made."
  exit 0
fi
[[ "${purge}" != yes ]] || verify_volumes
while read -r id; do
  [[ -n "${id}" ]] || continue
  verify_runner "${id}"
  if [[ "${force}" != yes && "$(docker inspect --format '{{.State.Running}}' "${id}")" == true ]]; then
    fail "Runners are running. Even an idle runner can accept work. Stop new work, then use --force to permit task interruption."
  fi
done <<< "${runners}"

echo "Installation: ${install_dir}"
echo "Compose project: ${project}"
if [[ "${purge}" == yes ]]; then
  echo "This deletes the installation's Docker data volumes. Configuration files and OIDC keys remain."
else
  echo "Data volumes, configuration files, and OIDC keys will remain."
fi
if [[ "${force}" == yes ]]; then
  echo "Runner tasks can be interrupted."
fi
if [[ "${confirmed}" != yes ]]; then
  [[ -t 0 ]] || fail "Interactive confirmation is unavailable. Use --yes after you review the options."
  read -r -p "Continue? [y/N] " answer
  [[ "${answer}" == y || "${answer}" == Y ]] || exit 0
fi

verify_containers
verify_network
[[ "${purge}" != yes ]] || verify_volumes
# Stop the manager first so it cannot replace runners during removal. Keep the
# app available until preserved runner records no longer count toward capacity.
"${compose[@]}" stop fleet-manager
runners="$(runner_containers)"
observed_runners="$(printf '%s\n%s\n' "${observed_runners}" "${runners}" | sed '/^$/d' | sort -u)"
while read -r id; do
  [[ -n "${id}" ]] || continue
  verify_runner "${id}"
  if [[ "${force}" != yes && "$(docker inspect --format '{{.State.Running}}' "${id}")" == true ]]; then
    fail "A runner started during confirmation. Use --force to permit task interruption, or ./install.sh to resume."
  fi
done <<< "${runners}"
if [[ "${purge}" != yes && -n "${runners}" ]]; then
  load_runner_api_token
  retirement_deadline=$((SECONDS + 60))
  while read -r id; do
    [[ -n "${id}" ]] || continue
    retire_runner "${id}" "${retirement_deadline}"
  done <<< "${runners}"
  unset runner_api_token
fi
# The database is discarded on purge; otherwise every runner is now terminated.
# Stop new assignments before removing the verified Docker containers.
"${compose[@]}" stop app
while read -r id; do
  [[ -n "${id}" ]] || continue
  verify_runner "${id}"
  docker rm -f "${id}"
done <<< "${runners}"

# Compose down -v also deletes pre-existing volumes without ownership labels.
# Remove only the exact volumes verified above, after their containers are gone.
"${compose[@]}" down --remove-orphans
if [[ "${purge}" == yes ]]; then
  while read -r volume; do
    [[ -n "${volume}" ]] || continue
    docker volume rm "${volume}"
  done <<< "${volumes}"
fi

containers="$(project_containers)"
runners="$(runner_containers)"
remaining_network="$(docker network ls -q --filter "name=^${network}$")"
[[ -z "${containers}" ]] || fail "Some Compose containers remain. Check docker compose ps --all."
[[ -z "${runners}" ]] || fail "Some managed runners remain."
while read -r id; do
  [[ -n "${id}" ]] || continue
  [[ -z "$(docker ps -aq --no-trunc --filter "id=${id}")" ]] ||
    fail "Previously observed runner ${id} remains. Check its network membership before removal."
done <<< "${observed_runners}"
[[ -z "${remaining_network}" ]] || fail "The installation network remains in use."
if [[ "${purge}" == yes ]]; then
  volumes="$(project_volumes)"
  [[ -z "${volumes}" ]] || fail "Some project volumes remain. Check their ownership before removal."
fi
echo "SuperPlane containers and network were removed."
[[ "${purge}" != yes ]] || echo "Docker data volumes were removed."
echo "Configuration files and OIDC keys remain in ${install_dir}."

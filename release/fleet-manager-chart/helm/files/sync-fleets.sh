set -eu

api_base="${SUPERPLANE_URL%/}/admin/api/installation/fleets"

request() {
  method="$1"
  url="$2"
  payload="${3:-}"
  attempt=1

  while [ "${attempt}" -le 6 ]; do
    if [ -n "${payload}" ]; then
      if ! status="$(
        printf '%s' "${payload}" | base64 -d | curl \
          --silent \
          --show-error \
          --connect-timeout 10 \
          --max-time 90 \
          --output /tmp/fleet-response \
          --write-out '%{http_code}' \
          --request "${method}" \
          --header "Authorization: Bearer ${INSTALLATION_ADMIN_TOKEN}" \
          --header 'Content-Type: application/json' \
          --data-binary @- \
          "${url}"
      )"; then
        status=000
      fi
    else
      if ! status="$(
        curl \
          --silent \
          --show-error \
          --connect-timeout 10 \
          --max-time 90 \
          --output /tmp/fleet-response \
          --write-out '%{http_code}' \
          --request "${method}" \
          --header "Authorization: Bearer ${INSTALLATION_ADMIN_TOKEN}" \
          "${url}"
      )"; then
        status=000
      fi
    fi

    case "${status}" in
      000|408|425|429|500|502|503|504)
        if [ "${attempt}" -lt 6 ]; then
          sleep "$((attempt * 2))"
          attempt="$((attempt + 1))"
          continue
        fi
        ;;
    esac

    printf '%s' "${status}"
    return
  done
}

update_fleet() {
  fleet_id="$1"
  update_payload="$2"
  status="$(request PATCH "${api_base}/${fleet_id}" "${update_payload}")"
  if [ "${status}" != 200 ]; then
    echo "Could not update fleet ${fleet_id}: HTTP ${status}" >&2
    return 1
  fi
}

sync_fleet() {
  fleet_id="$1"
  create_payload="$2"
  update_payload="$3"
  status="$(request GET "${api_base}/${fleet_id}")"

  case "${status}" in
    200)
      update_fleet "${fleet_id}" "${update_payload}"
      ;;
    404)
      status="$(request POST "${api_base}" "${create_payload}")"
      case "${status}" in
        200|201)
          ;;
        409)
          update_fleet "${fleet_id}" "${update_payload}"
          ;;
        *)
          echo "Could not create fleet ${fleet_id}: HTTP ${status}" >&2
          return 1
          ;;
      esac
      ;;
    *)
      echo "Could not read fleet ${fleet_id}: HTTP ${status}" >&2
      return 1
      ;;
  esac
}

if [ "$(( $# % 3 ))" -ne 0 ]; then
  echo "Fleet arguments must be ID, create payload, and update payload triples" >&2
  exit 1
fi

while [ "$#" -gt 0 ]; do
  sync_fleet "$1" "$2" "$3"
  shift 3
done

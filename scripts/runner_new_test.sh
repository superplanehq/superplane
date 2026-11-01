#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary="$(mktemp -d)"
trap 'rm -rf "${temporary}"' EXIT

cat >"${temporary}/docker" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
case " $* " in
  *" ps "*) printf 'running\n' ;;
  *" go run "*)
    [[ " $* " == *" ./cmd/cli/main.go "* ]]
    [[ " $* " == *" --installation "* ]]
    [[ " $* " == *" --output json "* ]]
    [[ " $* " == *" SUPERPLANE_API_TOKEN=test-token "* ]]
    [[ " $* " == *" SUPERPLANE_API_URL=http://localhost:8000 "* ]]
    if [[ " $* " == *" fleets list "* ]]; then
      printf '{"fleets":[{"id":"e1-large-amd64"}]}\n'
      exit
    fi
    printf 'create\n' >>"${STUB_LOG}"
    case "${STUB_FAILURE}" in
      create) exit 1 ;;
      malformed) printf '{"runner":{}}\n' ;;
      *) printf '{"runner":{"id":"direct-runner"},"registrationToken":"registration-token"}\n' ;;
    esac
    ;;
  *" stop fleet-manager "*) printf 'stop\n' >>"${STUB_LOG}" ;;
  *" up -d --no-build --no-deps fleet-manager "*) printf 'restore\n' >>"${STUB_LOG}" ;;
  *" run --rm "*)
    [[ " $* " == *" --url http://runner-network:8000 "* ]]
    [[ " $* " == *" --registration-token registration-token "* ]]
    [[ " $* " == *" --name superplane-runner-direct-runner "* ]]
    printf 'run\n' >>"${STUB_LOG}"
    [[ "${STUB_FAILURE}" != run ]]
    ;;
esac
STUB
chmod +x "${temporary}/docker"

for failure in none create malformed run; do
  export STUB_LOG="${temporary}/${failure}.log" STUB_FAILURE="${failure}"
  status=0
  PATH="${temporary}:${PATH}" INSTALLATION_ADMIN_TOKEN=test-token \
    LOCAL_RUNNER_API_URL=http://runner-network:8000 \
    bash "${repo_root}/scripts/runner-new" >"${temporary}/output" 2>&1 || status=$?
  if [[ "${failure}" == none ]]; then
    [[ "${status}" == 0 ]]
    expected=$'stop\ncreate\nrun\nrestore'
  else
    [[ "${status}" != 0 ]]
    expected=$'stop\ncreate\nrestore'
    if [[ "${failure}" == run ]]; then expected=$'stop\ncreate\nrun\nrestore'; fi
  fi
  [[ "$(<"${STUB_LOG}")" == "${expected}" ]]
done
printf 'runner-new: structured registration and Fleet Manager restoration passed\n'

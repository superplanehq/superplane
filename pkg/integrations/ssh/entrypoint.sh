#!/usr/bin/env bash
set -euo pipefail

USERNAME="${SSH_USERNAME:-testuser}"

# Ensure user exists (in case you override SSH_USERNAME)
if ! id -u "${USERNAME}" >/dev/null 2>&1; then
  useradd -m -s /bin/bash "${USERNAME}"
fi

HOME_DIR="$(getent passwd "${USERNAME}" | cut -d: -f6)"
SSH_DIR="${HOME_DIR}/.ssh"
AUTH_KEYS="${SSH_DIR}/authorized_keys"

mkdir -p "${SSH_DIR}"
chmod 700 "${SSH_DIR}"
touch "${AUTH_KEYS}"
chmod 600 "${AUTH_KEYS}"
chown -R "${USERNAME}:${USERNAME}" "${SSH_DIR}"

# Option A: provide public key via env var (recommended)
if [[ -n "${SSH_PUBKEY:-}" ]]; then
  echo "${SSH_PUBKEY}" > "${AUTH_KEYS}"
  chown "${USERNAME}:${USERNAME}" "${AUTH_KEYS}"
fi

# Option B: if you mount authorized_keys into /mounted_authorized_keys
if [[ -f /mounted_authorized_keys ]]; then
  cat /mounted_authorized_keys > "${AUTH_KEYS}"
  chown "${USERNAME}:${USERNAME}" "${AUTH_KEYS}"
fi

# Generate host keys if missing
ssh-keygen -A

echo "Starting sshd for user=${USERNAME} ..."
exec /usr/sbin/sshd -D -e

# Cursor Cloud Environment

These notes apply only to Docker-in-Docker cloud VMs (for example Cursor
Cloud). Skip them on a normal workstation with Docker already running.

- Run all `make` commands from the repository root.
- The Docker daemon must be started manually:
  `sudo dockerd &>/tmp/dockerd.log &` — wait ~3-4 seconds before issuing Docker
  commands, then make the socket accessible with
  `sudo chmod 666 /var/run/docker.sock`.
- Docker needs the `fuse-overlayfs` storage driver and `iptables-legacy` for
  nested-container support.
- The `app` container starts with `sleep infinity`; you must explicitly run
  `make dev.server` to start the API + UI stack.

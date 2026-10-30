# GitHub Enterprise Cloud

This page covers SuperPlane settings for GitHub Enterprise Cloud (`ghe.com`).
GitHub Enterprise Server is not available yet.

The GitHub integration stays `github`. An empty host is github.com.

## Enable the feature

GitHub Enterprise Cloud needs a SuperPlane Enterprise license that grants
`github_enterprise`. SuperPlane then accepts a GitHub Enterprise Cloud
hostname during setup.

## Hostname allowlist

Set `SUPERPLANE_GITHUB_ENTERPRISE_ALLOWED_HOSTS` to a comma-separated list of
hostnames or `*.example.ghe.com` wildcards.

Helm value: `githubEnterprise.allowedHosts`.

An empty list allows any hostname that SuperPlane accepts. `github.com` is
always allowed.

## IP allow list

If the GitHub organization uses an IP allow list, add the SuperPlane egress
addresses. SuperPlane Cloud will publish that list when it is available.
Self-host operators use the egress addresses of their own cluster.

## Private network access

GitHub Enterprise Server on a private network will need
`AllowPrivateNetworkAccess` on the SuperPlane HTTP context. Do not enable
that for GitHub Enterprise Cloud on `ghe.com`.

## CA bundle

For a future GitHub Enterprise Server that uses a private CA, create a
Kubernetes Secret with a CA bundle. Set Helm:

- `githubEnterprise.extraCACertsSecret` to the Secret name
- `githubEnterprise.extraCACertsKey` to the key, default `ca-certificates.crt`

SuperPlane mounts the file as `SSL_CERT_FILE` on the API, workers, and runner.

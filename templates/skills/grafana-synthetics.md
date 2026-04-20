# Grafana Synthetics

Use this skill when building or operating Grafana Synthetic Monitoring checks through the SuperPlane `grafana` integration.

Focus on the currently supported HTTP synthetic check components:

- `grafana.createHttpSyntheticCheck`
- `grafana.getHttpSyntheticCheck`
- `grafana.updateHttpSyntheticCheck`
- `grafana.deleteHttpSyntheticCheck`

Implementation notes:

- Synthetic Monitoring uses its own base URL and access token. Do not assume the standard Grafana API token is sufficient.
- Resource fields should use resource names such as `syntheticCheck`, not `syntheticCheckId`.
- `getHttpSyntheticCheck` should return both normalized configuration and best-effort operational metrics so workflow details can show more than raw check config.
- Frontend details should follow the existing Dash0 synthetic check presentation: target, method, schedule, probes, enabled state, timestamps, and a check URL when available.

Configuration is grouped in the UI (similar to Dash0 synthetic checks). Top-level groups:

- `job`, `labels`
- `request` — `target`, `method`, `headers`, `body`, `noFollowRedirects`, `basicAuth`, `bearerToken`, `tls`
- `schedule` — `enabled`, `frequency`, `timeout`, `probes` (probe picker label: **Locations**)
- `validation` (optional) — `failIfSSL`, `failIfNotSSL`, `validStatusCodes`, body/header regex fields
- `alerts`

Older canvases may still store the same keys as a flat map; the backend accepts both.

Validation expectations:

- `request.target` (or legacy flat `target`) must be a valid HTTP or HTTPS URL.
- `schedule.probes` (or legacy flat `probes`) must contain at least one selected probe for create and update.
- `syntheticCheck` is required for get, update, and delete.
- `updateHttpSyntheticCheck` uses togglable sections: only enabled groups are merged on top of the check loaded from Grafana; omitted groups keep existing values.

Testing expectations:

- Regenerate component docs with `make gen.components.docs`.
- Run frontend lint budget and mapper specs.
- Run Grafana integration backend tests.
- Build both app and UI in the standard Docker-backed development environment.

# Daytona Get Preview URL Skill

Use this guidance when planning or configuring `daytona.getPreviewUrl`.

## Purpose

`daytona.getPreviewUrl` generates a preview URL for an HTTP service running in a Daytona sandbox.

## Required Configuration

- `sandbox` (required): sandbox identifier to generate URL for.
- `port` (optional): target port (defaults are component-defined).
- `signed` (optional): whether to generate a signed URL.
- `expiresInSeconds` (optional): expiration for signed URLs.

## Planning Rules

When generating workflow operations that include `daytona.getPreviewUrl`:

1. Always set `configuration.sandbox` from the upstream sandbox-producing node.
2. Set `configuration.port` only as a raw JSON number (for example `3000`), never as a string or handlebars expression.
3. If the exact port is unknown, omit `configuration.port` instead of guessing a dynamic expression.
4. Keep `signed` and `expiresInSeconds` defaults unless the user requests specific sharing or security behavior.
5. Place this node after server-setup command steps so the preview URL points to a running service.

## Output Semantics

- The preview link is available at `data.url`.
- For non-signed URLs, consumers may also need `data.token`.

## Mistakes To Avoid

- Missing `sandbox`.
- Setting `port` as a string value or handlebars expression (causes type validation errors).
- Generating preview URL before a service is running on the target port.
- Setting contradictory `signed`/`expiresInSeconds` options without user intent.

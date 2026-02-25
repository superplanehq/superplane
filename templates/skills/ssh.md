# SSH Component Skill

Use this guidance when planning or configuring the `ssh` component.

## Purpose

The `ssh` component runs a command on a remote host over SSH and routes output by command result:

- `success`: exit code `0`
- `failed`: non-zero exit code or connection failure

## Required Configuration

- `host` (required): SSH hostname or IP.
- `username` (required): SSH username.
- `command` (required): command to execute.
- `timeout` (required): timeout in seconds (must be at least `1`).
- `port` (optional): defaults to `22` when omitted.
- `workingDirectory` (optional): directory to `cd` into before command execution.

Authentication is configured under `authentication` (required):

- `authentication.authMethod` (required): `sshKey` or `password`.
- If `authMethod` is `sshKey`:
  - `authentication.privateKey` (required): secret key reference.
  - `authentication.passphrase` (optional): secret key reference.
- If `authMethod` is `password`:
  - `authentication.password` (required): secret key reference.

## Optional Retry Configuration

Use `connectionRetry` only when needed:

- `connectionRetry.enabled` (optional, default `false`)
- `connectionRetry.retries` (optional): must be `0` or greater when enabled
- `connectionRetry.intervalSeconds` (optional): must be at least `1` when enabled

## Planning Rules

When generating workflow operations that include `ssh`:

1. Always set `host`, `username`, `command`, `timeout`, and `authentication`.
2. Always set `authentication.authMethod`, then set only the credential fields for that method.
3. Do not mix both password and key authentication fields in the same configuration.
4. Keep `port` in valid SSH range (`1`-`65535`) when explicitly set.
5. Route normal completion from `success` and error handling/fallback logic from `failed`.
6. Do not invent extra output channels for this component.

## Expression Context

The `command` field supports expressions, for example:

- `echo {{$["Build"].version}}`
- `deploy --env={{root().data.environment}}`
- `bash -lc "retry={{previous().attempt}}; ./run.sh"`

## Good Configuration Example

- `host: "app.example.com"`
- `port: 22`
- `username: "ubuntu"`
- `command: "sudo systemctl restart api"`
- `timeout: 90`
- `authentication:`
  - `authMethod: "sshKey"`
  - `privateKey: { secret: "prod-ssh", key: "private_key" }`

## Mistakes To Avoid

- Missing `authentication.authMethod`.
- Using `sshKey` auth without `authentication.privateKey`.
- Using `password` auth without `authentication.password`.
- Setting `timeout` to `0` or negative values.
- Routing downstream logic without considering `failed` outcomes.

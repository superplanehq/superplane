# Workspace agent resources

> Status: Implemented for MCP connections. Skills remain a catalog shell.
> Audience: Product and engineering

This playbook is the source for workspace agent resources. One catalog
serves MCP connections now and skills later. One settings page and one
runner attach path serve both kinds.

## Locked decisions

1. **One workspace catalog.** Store rows in `factory_agent_resources`.
   Kinds are `mcp_server` and `skill`. One attach function injects enabled
   rows. Do not add a second table or settings page for skills later.
2. **One settings page named Agent resources.** The page lives under
   Workspace settings. Permission is `factories:update`. Tabs are
   **Connections** (MCP) and **Skills**. v1 implements Connections. The
   Skills tab ships as a shell with an empty state. Skills later fill that
   tab. They do not add a new route family.
3. **MCP v1 auth is headers or OAuth.** SuperPlane completes OAuth in the
   browser. The runner receives only a Bearer access token. Do not write
   refresh tokens to the runner.
4. **Inject at task build**, not in canvas YAML. Merge workspace MCP beside
   the first-party `planning_session_mcp.js` server. The name `superplane`
   is reserved.
5. **Skill rows are packages**, not one markdown file. The source is a
   GitHub repository, ref, and optional path. That shape supports
   third-party skills such as ui-ux-pro-max. Inline markdown is optional
   later.
6. **Storybook is the visual spec.** Every page state in this playbook has
   a named story before the page is done.

## Goal

A workspace admin adds MCP connections once. Every factory runner step,
including task refinement, can use those connections. The same catalog and
page will hold skills later without a second product surface.

## What exists today

Do not reinvent these pieces:

- Factory runner tasks already attach a first-party stdio MCP server for
  task refinement (`planning_session_mcp.js`).
- Organization integrations are product apps with setup wizards and
  webhooks. They are not a generic MCP URL catalog.
- Organization secrets already store header values. Runner
  `environmentFrom` already resolves those secrets.
- GitLab and Jira already complete browser OAuth, encrypt tokens, and
  refresh them in SuperPlane.
- Canvas YAML and `environmentFrom` bind secrets per node. That path
  misses existing automations and the refinement node unless every
  template is rewritten.

## Product rules

| Topic | Rule |
| --- | --- |
| Catalog | One table. Callers pass `factory_id` and filter by `kind`. |
| Kinds | `mcp_server` in v1. `skill` is reserved and unused at runtime. |
| Name | Slug, unique per factory. The name `superplane` is reserved. |
| MCP transport | HTTP / streamable HTTP only. No stdio or `npx` in v1. |
| Header auth | Optional. Map header names to organization secret name and key. |
| OAuth | SuperPlane is the MCP OAuth client. Inject Bearer at task start. |
| Cap | At most 20 enabled MCP servers per workspace. |
| Settings page | Workspace / Agent resources. Tabs: Connections, Skills. |
| Route | `.../settings/workspace/agent-resources`. Skills use `?tab=skills`. |
| Inject | `AttachWorkspaceAgentResources` at broker-task build. |
| Storybook | `Factories/Pages/Settings/Agent resources` lists every state. |

## Domain model

```
factory (workspace)
  └── factory_agent_resources
        kind: mcp_server | skill
        name, enabled, config JSONB
        └── factory_agent_resource_secrets (encrypted OAuth tokens)
              └── AttachWorkspaceAgentResources
                    └── Claude / Codex / OpenCode run.js
```

MCP `config` for OAuth:

```json
{
  "transport": "http",
  "url": "https://api.mobbin.com/mcp",
  "auth": "oauth"
}
```

MCP `config` for headers:

```json
{
  "transport": "http",
  "url": "https://mcp.example.com/mcp",
  "auth": "headers",
  "headers": [
    { "name": "Authorization", "secretName": "vendor-mcp", "secretKey": "token" }
  ]
}
```

Store no tokens in `config`. Encrypt OAuth refresh and access tokens in
`factory_agent_resource_secrets`. Resolve header values from organization
secrets at task build.

Validate MCP URLs as `https`. Reject loopback, link-local, and private
hosts.

Skill `config` later (do not persist or attach in v1):

```json
{
  "source": "github",
  "repository": "nextlevelbuilder/ui-ux-pro-max-skill",
  "ref": "<tag-or-sha>",
  "path": "."
}
```

OAuth connection status on the row:

- `not_connected` — saved, Connect still required
- `connected` — refresh token present
- `needs_reconnect` — refresh failed
- `vendor_rejected` — discovery, DCR, or allowlist failed

## Settings page

Follow factory settings chrome. Page title: **Agent resources**.

```
Workspace settings
  Agent resources
    [ Connections ] [ Skills ]
    helper text
    list OR empty state
    primary action: Add connection | Add skill
```

Helper text: Agents on every run in this workspace can use these
resources. One signed-in account is shared by every agent.

Connections row: name, URL, auth (Header or Sign-in), status
(Connected / Not connected / Reconnect), enable switch, menu (Edit,
Disconnect, Delete).

Skills row (shell and later product): name, source (`owner/repo@ref`),
status (Ready / Failed to fetch), enable switch. Add skill form: GitHub
repository, ref, optional subpath.

Empty Connections: No MCP connections yet. Add a connection so agents can
use it on every run.

Empty Skills (v1): Skills are not available yet. SuperPlane will load
skill packages from GitHub on this tab.

Copy uses SuperPlane, workspace, connection, skill, and agent as stable
nouns. Do not put MCP-only words in the nav label.

## OAuth

SuperPlane completes MCP OAuth in the admin browser. The fleet runner is
headless and cannot open a login window.

1. Probe the MCP URL. Read RFC 9728 protected-resource metadata.
2. Discover the authorization server (RFC 8414 or OpenID Connect).
3. Obtain a client id with Client ID Metadata Documents, or Dynamic
   Client Registration when the server advertises it.
4. Run authorization-code plus PKCE. Redirect to
   `/api/v1/mcp-oauth/callback`.
5. Encrypt the refresh token. Record who connected and when.
6. At task build, refresh the access token and inject
   `Authorization: Bearer`.

Send RFC 8707 `resource=<canonical MCP URL>` on authorize and token
requests.

If the vendor allowlists only Claude, Cursor, or Codex, fail Connect with
a clear error. Keep header auth for vendors that also ship an API key.

v1 refreshes the access token at task start only. A short token can expire
during a long run. Do not add a SuperPlane MCP proxy in v1.

## Runtime inject

Each runner component already calls `ResolveEnvironment` and
`AttachPlanningSessionEnv`. Add `AttachWorkspaceAgentResources`.

1. Resolve the factory from the canvas (`workflows.factory_id`). No-op
   when the canvas is not factory-owned.
2. Load enabled `mcp_server` rows.
3. Resolve header secrets or mint an OAuth access token.
4. Ship `workspace_mcp.json`. Set `SUPERPLANE_WORKSPACE_MCP_CONFIG`.
5. Each `run.js` merges those remote servers with the SuperPlane stdio
   MCP when planning is on.

| CLI | Merge |
| --- | --- |
| Claude Code | `mcp.runtime.json` with `{ type: "http", url, headers }`. Write this file on every run when workspace MCP exists. Planning allowlists include `mcp__<name>`. |
| Codex | TOML `mcp_servers.<name>` url and headers. |
| OpenCode | `config.mcp.<name>` as `type: "remote"`. |

Custom HTTP MCP tools can change external systems during refinement. The
repo stays read-only. State that in helper text.

## Skills follow-on

v1 does not install skills. The catalog kind, Skills tab shell, and
Storybook `SkillsEmpty` / `SkillsGitHub` stories lock the layout.

When skills land:

1. Snapshot the GitHub tree at add time into SuperPlane blob storage.
   Pin the snapshot by SHA.
2. Unpack onto the runner under a shared skills root. Copy into Claude
   `.claude/skills/<name>/` and Codex/OpenCode `.agents/skills/<name>/`.
3. Do not rewrite canvas YAML.
4. Do not run vendor installers such as `uipro` or `npx skills add`.

The fleet runner image must include `python3` for skills that run search
scripts. Public repositories first. Private GitHub skills need the
workspace VCS token later.

## Out of scope

- Stdio / `npx` MCP on the runner
- SuperPlane MCP proxy or mid-run token refresh
- Canvas AI chat custom tools
- Anthropic Managed Agent vault MCP
- Per-step opt-out and organization-wide sharing
- Interactive MCP Apps UIs
- First-party vendor connector registration
- Skill package install and snapshot

## Test plan

- [ ] Create a header MCP connection and list it on Agent resources.
- [ ] Enable the connection and confirm a runner task ships `workspace_mcp.json`.
- [ ] Confirm Claude, Codex, and OpenCode merge the remote server.
- [ ] Confirm planning sessions still attach the SuperPlane stdio MCP.
- [ ] Start OAuth Connect and complete the callback with a mock auth server.
- [ ] Confirm a failed vendor allowlist shows Reconnect and the error text.
- [ ] Confirm private MCP URLs and `http` URLs are rejected.
- [ ] Open Storybook `Factories/Pages/Settings/Agent resources`.
- [ ] Review every named story in the list below.

## Storybook

File: `web_src/src/pages/factories/pages/settings/FactorySettingsAgentResources.stories.tsx`

Title: `Factories/Pages/Settings/Agent resources`

Connections tab:

- `Empty`
- `HeaderAuth`
- `OAuthNotConnected`
- `OAuthConnected`
- `OAuthNeedsReconnect`
- `OAuthVendorRejected`
- `Mixed`
- `AddConnectionDialog`

Skills tab:

- `SkillsEmpty`
- `SkillsGitHub` (mocked `nextlevelbuilder/ui-ux-pro-max-skill`)

Keep the page reachable from `Settings.stories.tsx` chrome through the
sidebar.

## Maintenance notes

Shipped v1 surfaces:

- Settings page: `web_src/src/pages/factories/pages/settings/FactorySettingsAgentResourcesPage.tsx`
- Route: `/{org}/workspaces/{key}/settings/workspace/agent-resources`
- Stories: `web_src/src/pages/factories/pages/settings/FactorySettingsAgentResources.stories.tsx`
- Catalog RPCs: `List/Create/Update/DeleteFactoryAgentResource` in `protos/factories.proto`
- OAuth: `StartFactoryAgentResourceOAuth`, `/api/v1/mcp-oauth/callback`
- Inject: `pkg/components/runner/workspace_agent_resources.go`

When you add `kind=skill` runtime:

1. Update this playbook. Keep locked decisions 1, 2, and 5.
2. Fill the Skills tab. Do not add a second settings page.
3. Extend `AttachWorkspaceAgentResources` only. Do not add a second
   inject path.
4. Add stories for fetch failure and ready skill rows. Keep `SkillsEmpty`.
5. Keep proto field numbers contiguous after `make pb.gen`.
6. Keep GitHub `repository`, `ref`, and `path` as the skill source.
   Do not add `npx skills` install.

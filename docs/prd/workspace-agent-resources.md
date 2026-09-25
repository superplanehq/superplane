# Workspace agent resources

> Status: Implemented for MCP connections and inline SKILL.md skills.
> Audience: Product and engineering

This playbook is the source for workspace MCP servers and skills. One
catalog table stores both kinds. Settings, flags, and automation
sections are separate. One attach function still injects enabled rows.

## Locked decisions

1. **One workspace catalog.** Store rows in `factory_agent_resources`.
   Kinds are `mcp_server` and `skill`. One attach function injects enabled
   rows. Do not add a second table.
2. **Separate MCP and Skills settings pages.** The pages live under
   Workspace settings. Permission is `factories:update`. Experimental
   flags are `workspace_mcp` and `workspace_skills`. The old
   `workspace/agent-resources` route redirects to the new pages.
3. **MCP v1 auth is headers or OAuth.** SuperPlane completes OAuth in the
   browser. The runner receives only a Bearer access token. Do not write
   refresh tokens to the runner.
4. **Inject at task build**, not in canvas YAML. Merge workspace MCP beside
   the first-party `planning_session_mcp.js` server. The name `superplane`
   is reserved.
5. **Skills v1 is one SKILL.md file.** The admin pastes markdown on a
   full-page editor. SuperPlane stores `{ "source": "inline", "markdown": "..." }`
   and writes that file onto the runner. GitHub packages (`repository`,
   `ref`, `path`) stay the follow-on source for third-party skills such as
   ui-ux-pro-max. Do not add `npx skills` install.
6. **Storybook is the visual spec.** Every page state in this playbook has
   a named story before the page is done.

## Goal

A workspace admin adds MCP servers and inline skills once. Every
factory runner step, including task refinement, can use those resources.
GitHub skill packages fill the same catalog later without a second
table.

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
| Kinds | `mcp_server` and `skill`. |
| Name | Slug, unique per factory. The name `superplane` is reserved. |
| MCP transport | HTTP / streamable HTTP only. No stdio or `npx` in v1. |
| Header auth | Optional. Map header names to organization secret name and key. |
| OAuth | SuperPlane is the MCP OAuth client. Inject Bearer at task start. |
| Cap | At most 20 enabled MCP servers per workspace. |
| Skills v1 | Inline SKILL.md only. Extra files and scripts are not included. |
| Flags | `workspace_mcp` and `workspace_skills`. |
| Settings pages | Workspace / MCP servers and Workspace / Skills. |
| Routes | `.../settings/workspace/mcp` and `.../settings/workspace/skills`. Skill create and edit use `/new` and `/:resourceId`. |
| Tools | Workspace `disabledTools` plus node `disabledAgentResourceTools`. New tools stay on. |
| Inject | `AttachWorkspaceAgentResources` at broker-task build. Attaches each kind only when its flag is on. |
| Storybook | `Factories/Pages/Settings/MCP and skills` lists every page state. |

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
  ],
  "disabledTools": ["create_issue"]
}
```

Store no tokens in `config`. Encrypt OAuth refresh and access tokens in
`factory_agent_resource_secrets`. Resolve header values from organization
secrets at task build.

Validate MCP URLs as `https`. Reject loopback, link-local, and private
hosts.

Skill `config` for inline SKILL.md:

```json
{
  "source": "inline",
  "markdown": "---\nname: review-copy\ndescription: Review UI copy.\n---\n\nWrite STE copy."
}
```

Skill `config` later (GitHub packages; do not persist in v1 create):

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

## Settings pages

Follow factory settings chrome. Page titles: **MCP servers** and **Skills**.

```
Workspace settings
  MCP servers
    catalog picker or Custom
    list OR empty state
    primary action: Add MCP server
  Skills
    list OR empty state
    primary action: Add skill
    full-page editor at /skills/new and /skills/:id
```

MCP server row: green status dot when connected, name, URL, auth
(Header or Sign-in), status (Connected / Not connected / Reconnect),
enable switch, expandable tools, menu (Edit, Disconnect, Delete).

Tool rows: name and Read or Write. Sort by name, read first, or write
first. Do not show the description. A tool that is off here is off for
every automation.

Skills row: name, source (`SKILL.md` for inline), enable switch, menu
(Edit, Delete). The editor has a name field, a `/command` recommendation,
and a full-page markdown editor.

Empty MCP servers: No MCP servers yet. Add an MCP server so agents can
use it on every run.

Empty Skills: No skills yet. Add a SKILL.md so agents can use it on
every run.

Copy uses SuperPlane, workspace, MCP server, skill, and agent as stable
nouns.

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
2. Load enabled `mcp_server` and `skill` rows.
3. Resolve header secrets or mint an OAuth access token.
4. Ship `workspace_mcp.json` under `SUPERPLANE_TASK_DIR`. Set
   `SUPERPLANE_WORKSPACE_MCP_CONFIG` to `$SUPERPLANE_TASK_DIR/workspace_mcp.json`.
5. Each `run.js` merges those remote servers with the SuperPlane stdio
   MCP when planning is on.
6. Write each inline skill to `.claude/skills/<name>/SKILL.md` and
   `.agents/skills/<name>/SKILL.md` under `SUPERPLANE_TASK_DIR`. Prompt
   steps copy those files into the agent working directory before the
   CLI starts. If the repository already has that skill directory, keep
   the project files and skip the workspace copy.

| CLI | Merge |
| --- | --- |
| Claude Code | `mcp.runtime.json` with `{ type: "http", url, headers }`. Write this file on every run when workspace MCP exists. Planning allowlists include `mcp__<name>`. Pass disabled tools as `mcp__<server>__<tool>` on `--disallowedTools`. Skills load from `.claude/skills`. |
| Codex | TOML `mcp_servers.<name>` url, headers, and `disabled_tools`. Skills load from `.agents/skills`. |
| OpenCode | `config.mcp.<name>` as `type: "remote"`. Deny disabled tools with `permission.<server>_<tool>`. Skills load from `.agents/skills`. |

Custom HTTP MCP tools can change external systems during refinement. The
repo stays read-only. State that in helper text.

## Skills follow-on

GitHub packages are not in v1 create. Keep Storybook `SkillsGitHub` as
the later layout.

When GitHub packages land:

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
- Skill scripts, references, and extra files beside SKILL.md

## Test plan

- [ ] Create a header MCP connection and list it on MCP servers.
- [ ] Enable the connection and confirm a runner task ships `workspace_mcp.json`.
- [ ] Confirm Claude, Codex, and OpenCode merge the remote server.
- [ ] Confirm disabled tools appear in `workspace_mcp.json` and in Claude, Codex, and OpenCode denylists.
- [ ] Confirm planning sessions still attach the SuperPlane stdio MCP.
- [ ] Start OAuth Connect and complete the callback with a mock auth server.
- [ ] Confirm a failed vendor allowlist shows Reconnect and the error text.
- [ ] Confirm private MCP URLs and `http` URLs are rejected.
- [ ] Create an inline skill on the full-page editor and list it on Skills.
- [ ] Confirm a runner task ships `.claude/skills/<name>/SKILL.md` and `.agents/skills/<name>/SKILL.md`.
- [ ] Confirm empty markdown and GitHub skill sources are rejected.
- [ ] Open Storybook `Factories/Pages/Settings/MCP and skills`.
- [ ] Review every named story in the list below.

## Storybook

File: `web_src/src/pages/factories/pages/settings/FactorySettingsAgentResources.stories.tsx`

Title: `Factories/Pages/Settings/MCP and skills`

MCP servers page:

- `MCPEmpty`
- `MCPCatalog`
- `HeaderAuth`
- `OAuthNotConnected`
- `ConnectedGreenDot`
- `OAuthNeedsReconnect`
- `OAuthVendorRejected`
- `Mixed`

Skills page:

- `SkillsEmpty`
- `SkillsInline`
- `SkillsGitHub` (mocked `nextlevelbuilder/ui-ux-pro-max-skill`)
- `SkillEditor`

Also keep `Factories/Pages/Settings/MCP catalog`, `MCP status`, and `MCP tools` for the picker, green dot, tool sort, and `8/12` count.

Keep the pages reachable from `Settings.stories.tsx` chrome through the
sidebar.

## Maintenance notes

Shipped surfaces:

- MCP page: `web_src/src/pages/factories/pages/settings/FactorySettingsMCPPage.tsx`
- Skills page: `web_src/src/pages/factories/pages/settings/FactorySettingsSkillsPage.tsx`
- Skill editor: `web_src/src/pages/factories/pages/settings/FactorySettingsSkillEditorPage.tsx`
- Routes: `/{org}/workspaces/{key}/settings/workspace/mcp` and `.../skills`
- Stories: `web_src/src/pages/factories/pages/settings/FactorySettingsAgentResources.stories.tsx`
- Catalog RPCs: `List/Create/Update/DeleteFactoryAgentResource` in `protos/factories.proto`
- OAuth: `StartFactoryAgentResourceOAuth`, `/api/v1/mcp-oauth/callback`
- Inject: `pkg/components/runner/workspace_agent_resources.go`

When you add GitHub `kind=skill` packages:

1. Update this playbook. Keep locked decisions 1, 3, and 5.
2. Keep the Skills page. Do not add a second catalog table.
3. Extend `AttachWorkspaceAgentResources` only. Do not add a second
   inject path.
4. Add stories for fetch failure and ready GitHub skill rows. Keep
   `SkillsEmpty` and `SkillsInline`.
5. Keep proto field numbers contiguous after `make pb.gen`.
6. Keep GitHub `repository`, `ref`, and `path` as the package source.
   Do not add `npx skills` install.

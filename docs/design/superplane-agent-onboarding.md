# SuperPlane agent as the default onboarding runner

Status: Draft

Date: 2026-08-27

## Problem

New-workspace automations are authored as **Run Claude Code**. Setup can
rewrite those nodes when the user has a key or hosted credit. The default
story still asks the user to connect Anthropic, OpenAI, or OpenRouter.

The product default is SuperPlane agent. SuperPlane agent is hosted
OpenRouter. The UI must not name OpenRouter on that default path. A user
can still bring a key, but that choice must stay optional and collapsed.

## Goal

After the agent step, a new workspace:

- Uses **Run OpenRouter Agent** with SuperPlane-hosted credentials by
  default.
- Explains that SuperPlane runs the agent. It does not name OpenRouter.
- Lets the user expand **Use your own key** and connect Claude, OpenAI,
  or OpenRouter.
- Rewrites the same automation nodes when the user selects a key.

Existing workspaces do not change.

## Agent step

The default choice is selected: SuperPlane agent.

Copy:

- Headline: name the SuperPlane agent. Do not say "Connect agent" as the
  only story.
- Body: SuperPlane will run the agent on this workspace. Work still
  starts only after approval on a ticket.
- Do not write "OpenRouter", "hosted OpenRouter", or provider product
  names on the default card.

A muted control, **Use your own key**, expands three connect rows:

- Claude (Anthropic)
- OpenAI
- OpenRouter

Those rows keep the current connect flow. Selecting one row makes that
provider the agent plan. Clearing the key, or choosing SuperPlane agent
again, returns to the default.

If hosted credit is empty and the user does not connect a key, they
cannot finish. Empty-credit copy lives in the expanded BYOK path, not as
the main story.

A Claude, OpenAI, or OpenRouter key that already exists on the
organization does not steal the default. The user must expand BYOK and
select that provider.

## Agent plan

Keep the existing finish-setup rewrite. Change how the plan is chosen.

Default plan:

- Component: `runnerOpenRouter`
- Credentials: `{ source: "hosted" }`
- Model: first allowlisted OpenRouter model that matches the current
  hosted-model picker. Prefer a Sonnet id when the allowlist has one.
  Fall back to the first allowlisted OpenRouter model.

BYOK plan:

- Claude → `runnerClaudeCode` and the Claude installation
- OpenAI → `runnerCodex` and the OpenAI installation
- OpenRouter → `runnerOpenRouter` and the OpenRouter installation

`resolveOnboardingAgent` must stop preferring a connected provider over
hosted SuperPlane agent. Connected providers apply only after an explicit
BYOK selection.

Intake analysis uses the same plan. Backend `intakeAgentSpecs` keeps
Claude, Codex, and OpenRouter as valid runners. Hosted fallback prefers
OpenRouter first.

## Templates

Author the bundled canvases as SuperPlane agent:

- `web_src/src/pages/home/factories/line-apps/planning.canvas.yaml`
- `web_src/src/pages/home/factories/line-apps/implementation.canvas.yaml`
- `web_src/src/pages/home/factories/line-apps/pr.canvas.yaml`
- `web_src/src/pages/home/factories/software-factory/canvas.yaml`

Each agent node:

- `component: runnerOpenRouter`
- `credentials.source: hosted`
- `model:` an OpenRouter model id, default `anthropic/claude-sonnet-4-6`

Keep the existing bash and prompt steps. Only the runner, credentials,
and model change.

Storybook and first-run intake fixtures that hardcode `runnerClaudeCode`
as the generated analysis node follow the same default.

## Rewrite

`rewriteOnboardingAgentNodes` today matches only `runnerClaudeCode`.
After the templates change, it must match the authored agent nodes
(`runnerOpenRouter`) and any leftover Claude Code or Codex nodes.

Finish setup still calls `agentRewriteFromPlan`. The default path writes
hosted OpenRouter onto nodes that already use that component. A BYOK
path swaps the component and the installation.

Do not change canvases on workspaces that already finished setup.

## Copy rules

- Product name: SuperPlane.
- Default path: SuperPlane agent.
- BYOK path may name Claude, OpenAI, and OpenRouter.
- ASD-STE100 for user-facing strings. No contractions. No slang.

## Tests

- Agent step: SuperPlane agent is selected. BYOK rows stay hidden until
  expand. Finish is enabled when hosted credit remains.
- Agent plan: no BYOK selection yields hosted OpenRouter. A connected
  Claude key does not change that plan.
- Agent plan: an expanded Claude, OpenAI, or OpenRouter selection yields
  that runner and installation.
- Materialize: default rewrite keeps `runnerOpenRouter` and hosted
  credentials. BYOK rewrite swaps to Claude Code, Codex, or OpenRouter
  with the installation name.
- Intake: hosted fallback prefers OpenRouter. A setup-recorded BYOK
  agent still scores with that runner.
- Copy: default strings do not contain "OpenRouter".

## Non-goals

- Do not migrate existing factory canvases.
- Do not add a new runner component.
- Do not expose OpenRouter as the default product name.
- Do not change GitHub, repository, or ticket steps.

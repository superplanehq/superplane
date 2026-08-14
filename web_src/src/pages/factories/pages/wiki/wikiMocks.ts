export type WikiDocument = {
  id: string;
  path: string;
  title: string;
  content: string;
};

export type WikiTreeFile = {
  kind: "file";
  name: string;
  document: WikiDocument;
};

export type WikiTreeFolder = {
  kind: "folder";
  name: string;
  children: WikiTreeNode[];
};

export type WikiTreeNode = WikiTreeFile | WikiTreeFolder;

function compareNodes(left: WikiTreeNode, right: WikiTreeNode): number {
  if (left.kind !== right.kind) return left.kind === "file" ? -1 : 1;
  return left.name.localeCompare(right.name);
}

export function buildWikiTree(documents: WikiDocument[]): WikiTreeNode[] {
  type MutableFolder = {
    kind: "folder";
    name: string;
    children: Map<string, MutableNode>;
  };
  type MutableNode = WikiTreeFile | MutableFolder;

  const root = new Map<string, MutableNode>();

  for (const document of documents) {
    const parts = document.path.split("/").filter(Boolean);
    if (parts.length === 0) continue;

    let current = root;
    for (let index = 0; index < parts.length; index += 1) {
      const part = parts[index];
      const isFile = index === parts.length - 1;
      if (isFile) {
        current.set(part, { kind: "file", name: part, document });
        continue;
      }
      const existing = current.get(part);
      if (existing?.kind === "folder") {
        current = existing.children;
        continue;
      }
      const folder: MutableFolder = { kind: "folder", name: part, children: new Map() };
      current.set(part, folder);
      current = folder.children;
    }
  }

  function freeze(nodes: Map<string, MutableNode>): WikiTreeNode[] {
    return Array.from(nodes.values())
      .map((node) =>
        node.kind === "folder" ? { kind: "folder" as const, name: node.name, children: freeze(node.children) } : node,
      )
      .sort(compareNodes);
  }

  return freeze(root);
}

export function wikiFolderPaths(nodes: WikiTreeNode[], prefix = ""): string[] {
  const paths: string[] = [];
  for (const node of nodes) {
    if (node.kind !== "folder") continue;
    const path = prefix ? `${prefix}/${node.name}` : node.name;
    paths.push(path, ...wikiFolderPaths(node.children, path));
  }
  return paths;
}

export const WIKI_DOCUMENTS_DEFAULT: WikiDocument[] = [
  {
    id: "overview",
    path: "overview.md",
    title: "Overview",
    content: `# SuperPlane

SuperPlane is an open source automation engine for AI-driven engineering.

It orchestrates workflows across Git, LLMs, CI/CD, observability, incident tools, and infrastructure — with durable execution, approvals, and an operational UI.

## What it does

- Version apps in git (\`canvas.yaml\`, \`console.yaml\`) with guardrails for agents and humans
- Run event-driven, multi-step workflows that outgrow a single script or CI job
- Give each app a console dashboard backed by live memory, runs, and executions

## Status

Beta. Self-host the core engine or use [SuperPlane Cloud](https://app.superplane.com). Docs: [docs.superplane.com](https://docs.superplane.com).

## Example use cases

- PR preview environments
- Policy-gated production deploy
- Progressive delivery (10% → 50% → 100%)
- Multi-repo release trains
- First-five-minutes incident triage
`,
  },
  {
    id: "team",
    path: "team.md",
    title: "Team",
    content: `# Team

## Roles in the product

- **Operators** — design canvases, approve gated steps, run consoles day to day
- **App agents** — per-app assistants that help design workflows and debug runs
- **External coding agents** — CLI and [skills](https://github.com/superplanehq/skills) under the same RBAC

## Ownership

Apps are git-backed deployable units. Humans own approvals and policy; agents operate inside those guardrails.

## Community

- GitHub: [superplanehq/superplane](https://github.com/superplanehq/superplane)
- Discord: [discord.superplane.com](https://discord.superplane.com)
`,
  },
  {
    id: "feat-apps-canvases",
    path: "features/apps-and-canvases.md",
    title: "Apps and canvases",
    content: `# Apps and canvases

An **app** is a deployable unit: workflow graph, console UI, app-scoped memory, and deterministic execution. It is versioned in git via \`canvas.yaml\` and \`console.yaml\`.

A **canvas** is a graph of steps and dependencies. One canvas can express multiple workflows and run them concurrently.

## How runs start

Incoming events match triggers and start runs with the event payload as input. Steps are components (triggers or actions), built-in or integration-backed.
`,
  },
  {
    id: "feat-components",
    path: "features/components-and-integrations.md",
    title: "Components and integrations",
    content: `# Components and integrations

Each canvas node is a **component**: a trigger or an action that performs one task (deploy, open an incident, notify, wait, require approval, call an LLM, etc.).

Integrations ship triggers and actions for tools teams already use:

- AI & LLM — Claude, Cursor, OpenAI, …
- Version control & CI/CD — GitHub, GitLab, Semaphore, CircleCI, …
- Cloud & infra — AWS, GCP, Azure, Cloudflare, …
- Observability — Datadog, Grafana, Sentry, Prometheus, …
- Incident & comms — PagerDuty, Rootly, Slack, Teams, …

Full catalog: [docs.superplane.com/components](https://docs.superplane.com/components/).
`,
  },
  {
    id: "feat-console",
    path: "features/console.md",
    title: "Console",
    content: `# Console

Each app can define an operational **console**: a dynamic grid of panels in \`console.yaml\`.

Typical panel jobs:

- KPIs, tables, and charts from memory and run data
- Runbooks and pinned canvas nodes
- Workflow controls for operators

Consoles are the day-2 surface for watching and steering durable workflows without leaving SuperPlane.
`,
  },
  {
    id: "feat-memory",
    path: "features/memory.md",
    title: "Memory",
    content: `# Memory

**Memory** is app-scoped (and canvas-aware) JSON storage that persists across runs.

Use it to hold shared state workflows and consoles need between executions — evidence packs, ship-set readiness, rollout percentages, operator notes — without bolting on an external store for every app.
`,
  },
  {
    id: "feat-agents",
    path: "features/agents-and-operators.md",
    title: "Agents and operators",
    content: `# Agents and operators

## Built-in app agent

Each app includes an agent that helps design workflows and debug runs inside the same RBAC as human operators.

## External agents

CLI and SuperPlane skills let coding agents interact with apps programmatically. All paths share access control, secrets, and approval gates.

## Operators

Humans approve policy gates, inspect consoles, and intervene when a durable run needs a decision.
`,
  },
  {
    id: "feat-runs",
    path: "features/runs.md",
    title: "Runs and durable execution",
    content: `# Runs and durable execution

A **run** is one execution of a workflow path on a canvas. SuperPlane tracks runs, run items, and payloads across restarts.

## Why it matters

- Failed steps can resume without custom retry glue
- Approvals and waits are first-class steps, not out-of-band tickets
- Execution stays deterministic so humans and AI share the same guardrails

Event → trigger match → run → durable step progression.
`,
  },
  {
    id: "arch-backend",
    path: "architecture/backend.md",
    title: "Backend",
    content: `# Backend

Go service exposing a **gRPC** API with a REST/OpenAPI gateway.

## Layout

- \`cmd/\` — server, workers, CLI entrypoints
- \`pkg/grpc/actions\` — gRPC API implementation
- \`pkg/models\` — database models (explicit \`*gorm.DB\` / tx first)
- \`pkg/workers\` — background workers
- \`protos/\` — API definitions; codegen via \`make pb.gen\`
- \`db/\` — structure and migrations

## Infra

PostgreSQL for state, RabbitMQ for messaging. Dev and many installs run via Docker Compose.
`,
  },
  {
    id: "arch-frontend",
    path: "architecture/frontend.md",
    title: "Frontend",
    content: `# Frontend

TypeScript + React app in \`web_src/\`, built with Vite.

- Generated API client: \`web_src/src/api-client/\` (from \`make pb.gen\`)
- Integration UI mappers: \`web_src/src/pages/app/mappers/<integration>/\`
- Shared non-React helpers live in \`web_src/src/lib/\` (not \`utils/\`)

Local UI after \`make dev.server\`: [http://localhost:8000](http://localhost:8000). See \`web_src/AGENTS.md\` for UI-specific conventions.
`,
  },
  {
    id: "arch-execution",
    path: "architecture/execution-and-workers.md",
    title: "Execution and workers",
    content: `# Execution and workers

Durable execution is carried by Go workers under \`pkg/workers\`, started from \`cmd/server\`.

Workers consume RabbitMQ messages, advance run items, call integration actions, and persist progress in PostgreSQL so runs survive process restarts.

When adding a worker: register startup in \`cmd/server/main.go\` and update compose env as needed.
`,
  },
  {
    id: "arch-integrations",
    path: "architecture/integrations.md",
    title: "Integrations package layout",
    content: `# Integrations package layout

Integration implementations live under \`pkg/integrations/<integration>/\`.

UI configuration mappers for the same provider live under \`web_src/src/pages/app/mappers/<integration>/\`.

Authoring guidance:

- [docs/contributing/component-implementations.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/component-implementations.md)
- [docs/contributing/component-design.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/component-design.md)
`,
  },
  {
    id: "dev-local",
    path: "development/local-setup.md",
    title: "Local setup",
    content: `# Local setup

Dev is Docker-based. You need Docker Compose; Go and Node run inside containers.

\`\`\`bash
make dev.up      # app, db, rabbitmq (first build ~3–5 min)
make dev.setup   # npm, Go modules, protos, migrate DB
make dev.server  # API (air) + Vite; UI on :8000
\`\`\`

Health: \`http://localhost:8000/health\`. First UI load prompts owner setup (\`OWNER_SETUP_ENABLED=yes\`).

## Demo container (no source checkout)

\`\`\`bash
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
\`\`\`
`,
  },
  {
    id: "dev-testing",
    path: "development/testing.md",
    title: "Testing",
    content: `# Testing

\`\`\`bash
make test                                          # Go unit/integration
make test PKG_TEST_PACKAGES=./pkg/workers          # targeted
make test.e2e                                      # end-to-end
E2E_TEST_PACKAGES=./test/e2e/workflows make test.e2e
\`\`\`

After Go changes: \`make format.go\`, then \`make lint && make check.build.app\`.

After JS/TS changes: \`make format.js\`, then \`make check.build.ui\`.

E2E authoring notes live in \`docs/contributing/e2e-tests.md\`.
`,
  },
  {
    id: "dev-contributing",
    path: "development/contributing.md",
    title: "Contributing",
    content: `# Contributing

- Read root \`AGENTS.md\` and \`CONTRIBUTING.md\`; UI work also reads \`web_src/AGENTS.md\`
- PR titles: Conventional Commits with \`feat:\`, \`fix:\`, \`chore:\`, or \`docs:\`
- Commits need DCO sign-off (\`git commit -s\`)
- Never hand-edit generated clients, OpenAPI, or migration files — use \`make pb.gen\` and \`make db.migration.create\`
- Application name in user-facing text: **SuperPlane** (not "Superplane")

License: Apache 2.0.
`,
  },
];

/** Second corpus shown after a simulated “Refresh knowledge”. */
export const WIKI_DOCUMENTS_REFRESHED: WikiDocument[] = [
  {
    id: "overview-r",
    path: "overview.md",
    title: "Overview",
    content: `# SuperPlane

Refreshed from [superplanehq/superplane](https://github.com/superplanehq/superplane).

Open source automation engine for AI-driven engineering: git-backed apps, durable canvas runs, consoles, and integrations across Git, LLM, CI/CD, observability, and incident tools.

Self-host or use Cloud at [app.superplane.com](https://app.superplane.com). Docs at [docs.superplane.com](https://docs.superplane.com).
`,
  },
  {
    id: "team-r",
    path: "team.md",
    title: "Team",
    content: `# Team

Regenerated from CONTRIBUTING and community links.

Maintainers accept focused PRs with DCO sign-off. Operators and agents share RBAC; secrets stay encrypted. Join Discord for questions: [discord.superplane.com](https://discord.superplane.com).
`,
  },
  {
    id: "feat-apps-canvases-r",
    path: "features/apps-and-canvases.md",
    title: "Apps and canvases",
    content: `# Apps and canvases

Updated: apps remain the git-versioned unit (\`canvas.yaml\` + \`console.yaml\`). Canvases are dependency graphs; one canvas may host concurrent workflows with shared memory and approvals.
`,
  },
  {
    id: "feat-components-r",
    path: "features/components-and-integrations.md",
    title: "Components and integrations",
    content: `# Components and integrations

Catalog spans AI, VCS/CI, cloud, observability, incident, and messaging providers. Missing a provider? Open an issue on the SuperPlane repo. Implementation packages: \`pkg/integrations/<name>/\`.
`,
  },
  {
    id: "feat-console-r",
    path: "features/console.md",
    title: "Console",
    content: `# Console

Console grids bind to memory, runs, and executions. Prefer operational panels (KPIs, runbooks, pinned nodes, controls) over one-off dashboards outside the app.
`,
  },
  {
    id: "feat-memory-r",
    path: "features/memory.md",
    title: "Memory",
    content: `# Memory

App-scoped JSON that survives across runs. Consoles and canvas steps read/write the same store so progressive delivery state and incident evidence packs stay in-product.
`,
  },
  {
    id: "feat-agents-r",
    path: "features/agents-and-operators.md",
    title: "Agents and operators",
    content: `# Agents and operators

Per-app agent for design and run debug; CLI/skills for external coding agents. Same RBAC and approval gates on every path.
`,
  },
  {
    id: "feat-runs-r",
    path: "features/runs.md",
    title: "Runs and durable execution",
    content: `# Runs and durable execution

Runs, run items, and payloads are persisted; workers resume failed steps. Approvals and waits are canvas nodes, not sidecar process.
`,
  },
  {
    id: "arch-backend-r",
    path: "architecture/backend.md",
    title: "Backend",
    content: `# Backend

Go + gRPC (+ OpenAPI gateway). Key packages: \`pkg/grpc/actions\`, \`pkg/models\`, \`pkg/workers\`, \`pkg/integrations\`. State in PostgreSQL; messaging via RabbitMQ.
`,
  },
  {
    id: "arch-frontend-r",
    path: "architecture/frontend.md",
    title: "Frontend",
    content: `# Frontend

\`web_src/\` Vite + React. Do not hand-edit \`web_src/src/api-client/\` — regenerate with \`make pb.gen\`. UI conventions: \`web_src/AGENTS.md\`.
`,
  },
  {
    id: "arch-repo-r",
    path: "architecture/repo-layout.md",
    title: "Repo layout",
    content: `# Repo layout

\`\`\`
cmd/        Go entrypoints
pkg/        gRPC, models, workers, integrations
web_src/    React/Vite UI
protos/     API definitions
db/         migrations
docs/       contributing docs
test/       backend + e2e
\`\`\`

Makefile is the task entrypoint for setup, codegen, lint, and tests.
`,
  },
  {
    id: "dev-local-r",
    path: "development/local-setup.md",
    title: "Local setup",
    content: `# Local setup

\`make dev.up\` → \`make dev.setup\` → \`make dev.server\`.

Re-run \`dev.setup\` when protos, Go modules, or frontend deps change. Corrupt Go module cache: \`make dev.clean.go.cache\` then \`make dev.setup.go\`.
`,
  },
  {
    id: "dev-testing-r",
    path: "development/testing.md",
    title: "Testing",
    content: `# Testing

\`make test\` for Go; \`make test.e2e\` for end-to-end. Point \`PKG_TEST_PACKAGES\` / \`E2E_TEST_PACKAGES\` at a package for a faster loop. CI runs on Semaphore.
`,
  },
  {
    id: "dev-contributing-r",
    path: "development/contributing.md",
    title: "Contributing",
    content: `# Contributing

Use \`make db.migration.create NAME=<name>\` for schema changes (never hand-write migrations). After protos: \`make pb.gen\` and \`make check.proto.field.numbers\`. PR prefix must be \`feat:\` / \`fix:\` / \`chore:\` / \`docs:\` with DCO.
`,
  },
];

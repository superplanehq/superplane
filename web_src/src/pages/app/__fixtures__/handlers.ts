import { materializeConsoleSpec } from "../lib/workflow-spec-files";

import { buildStorybookAgentChat, buildStorybookAgentMessages, STORYBOOK_AGENT_CHAT_ID } from "./agentChatResponses";
import defaultRaw from "./canvasAppResponses.json";
import softwareFactoryHowItWorks from "./repository/softwareFactory.howItWorks.md?raw";
import softwareFactoryReadme from "./repository/softwareFactory.README.md?raw";

// Shape of a captured fixture. Endpoint bodies are stored verbatim as the
// live API returned them (after trimming GitHub webhook payloads and
// dropping unused integration definitions), so they can be replayed straight
// back without reshaping.
//
// Every field is optional so a caller can seed just the endpoints their
// story exercises — the route table falls back to a benign empty response
// for anything that's missing.
export interface CanvasAppFixture {
  canvasId: string;
  organizationId: string;
  versionId?: string;
  publishedRunId?: string;
  rootEventId?: string;
  triggers?: unknown;
  actions?: unknown;
  widgets?: unknown;
  integrations?: unknown;
  /** GET /api/v1/canvases/{canvasId} */
  canvas?: { canvas?: { spec?: unknown; metadata?: { name?: string } } };
  /** GET /api/v1/canvases/{canvasId}/versions[?limit=50] */
  versions?: { versions?: Array<{ metadata?: Record<string, unknown> }> };
  /** GET /api/v1/canvases/{canvasId}/versions?limit=1 */
  versionsLatest?: { versions?: Array<{ metadata?: Record<string, unknown> }> };
  /** GET /api/v1/canvases/{canvasId}/runs (all pages collapsed into one) */
  runs?: {
    runs?: Array<Record<string, unknown>>;
    totalCount?: number;
    hasNextPage?: boolean;
    lastTimestamp?: string;
  };
  /**
   * GET /api/v1/canvases/{canvasId}/runs/{runId}
   * Prefer `runDetailsById` when multiple runs need distinct describe payloads.
   */
  runDetail?: { run?: Record<string, unknown> };
  /** Per-run describe payloads keyed by run id. */
  runDetailsById?: Record<string, { run?: Record<string, unknown> }>;
  /**
   * GET /api/v1/canvases/{canvasId}/events/{eventId}/executions
   * Prefer `executionsByEventId` so each run's root event returns its own steps.
   */
  executions?: { executions?: unknown[] };
  /** Per-root-event execution lists keyed by root event id. */
  executionsByEventId?: Record<string, { executions?: unknown[] }>;
  /** GET /api/v1/canvases/{canvasId}/memory (real API returns `{items: []}`) */
  memory?: { items?: unknown[] };
  /** GET /api/v1/canvases/{canvasId}/repository/file?path=console.yaml */
  consoleYaml?: string;
  /**
   * Extra repository file bodies keyed by path (e.g. `README.md`).
   * `console.yaml` still prefers `consoleYaml` when both are set.
   */
  repositoryFileContents?: Record<string, string>;
  /**
   * Paths returned by GET .../repository/files. Defaults to the standard
   * app-repo trio (`README.md`, `canvas.yaml`, `console.yaml`) plus any
   * keys from `repositoryFileContents`.
   */
  repositoryFilePaths?: string[];
  /** GET /api/v1/agents/canvases/{canvasId}/chat */
  agentChat?: { chat?: Record<string, unknown> };
  /** GET /api/v1/agents/chats/{chatId}/messages */
  agentMessages?: { messages?: Array<Record<string, unknown>>; hasMore?: boolean };
}

const DEFAULT_REPOSITORY_FILE_PATHS = ["README.md", "canvas.yaml", "console.yaml"] as const;

const capturedFixture = defaultRaw as CanvasAppFixture;

// Live Canvas / Software Factory console: Create a task (inline) beside a
// 3-column PR pipeline board. `useCanvasConsole` rejects an empty file.
const defaultConsoleYaml =
  capturedFixture.consoleYaml ??
  materializeConsoleSpec({
    canvasId: capturedFixture.canvasId,
    canvasName: capturedFixture.canvas?.canvas?.metadata?.name ?? "Software Factory",
    panels: [
      {
        id: "submit-task",
        type: "nodes",
        content: {
          title: "Create a task",
          nodes: [
            {
              node: "create-task-start",
              formMode: "inline",
              showFieldLabels: false,
              showNodeLabel: false,
              showRun: true,
              submitLabel: "Work on it",
              triggerName: "Create Task",
            },
          ],
        },
      },
      {
        id: "how-it-works",
        type: "markdown",
        content: {
          title: "How it works",
          body: softwareFactoryHowItWorks,
          variables: [],
        },
      },
      {
        id: "pipeline-board",
        type: "board",
        content: {
          title: "PR pipeline",
          dataSource: {
            kind: "runs",
            limit: 100,
            triggers: ["on-issue-labeled-trigger", "component-node-4m9qti"],
          },
          render: {
            kind: "board",
            groupBy: `{{ status == "passed" ? "Done" :
   status == "failed" || status == "cancelled" ? "Failed" :
   $["Mark PR Ready"].state != null ? "Human review" :
   "In progress" }}`,
            lanes: [
              { value: "In progress", color: "blue" },
              { value: "Human review", color: "yellow", label: "Human review" },
              { value: "Failed", color: "red" },
              { value: "Done", color: "green" },
            ],
            otherLane: false,
            sort: { field: "updatedAt", order: "desc" },
            where: [{ field: '$["Open Draft PR"].state', op: "exists" }],
            emptyMessage: "No factory pull requests yet. Submit a task to start one.",
            card: {
              titleField: `{{ $["Open Draft PR"].data.title != null
   ? $["Open Draft PR"].data.title
   : payload.data.issue.title }}`,
              fields: [
                {
                  field: '$["Open Draft PR"].data.number',
                  format: "link",
                  href: '{{ $["Open Draft PR"].data.html_url }}',
                  label: "PR",
                },
                {
                  field: "payload.data.issue.number",
                  format: "link",
                  href: "{{ payload.data.issue.html_url }}",
                  label: "Issue",
                },
                { field: "durationMs", format: "duration", label: "Elapsed" },
                { field: "updatedAt", format: "relative", label: "Updated" },
              ],
            },
          },
        },
      },
    ],
    layout: [
      // Left column: prompt + how-it-works (same width). Board matches stacked height.
      { i: "submit-task", x: 0, y: 0, w: 3, h: 6, minW: 2, minH: 4 },
      { i: "how-it-works", x: 0, y: 6, w: 3, h: 7, minW: 2, minH: 4 },
      { i: "pipeline-board", x: 3, y: 0, w: 9, h: 13, minW: 6, minH: 6 },
    ],
  });

const defaultFixture = {
  ...capturedFixture,
  consoleYaml: defaultConsoleYaml,
  repositoryFileContents: {
    "README.md": softwareFactoryReadme,
    ...capturedFixture.repositoryFileContents,
  },
} satisfies CanvasAppFixture;

export const canvasAppIds = {
  organizationId: defaultFixture.organizationId,
  canvasId: defaultFixture.canvasId,
  versionId: defaultFixture.versionId,
  publishedRunId: defaultFixture.publishedRunId,
  rootEventId: defaultFixture.rootEventId,
};

// A synthetic user with broad permissions so PermissionsProvider grants every
// canAct() check AppPage makes. We never capture the real user (email/token);
// only the permission strings matter for rendering.
function buildMeUser(orgId: string) {
  return {
    id: "storybook-user",
    name: "Storybook User",
    email: "storybook@superplane.dev",
    organizationId: orgId,
    hasToken: true,
    roles: ["org_admin"],
    groups: [],
    permissions: ["canvases", "integrations", "secrets", "groups", "users", "roles", "organization", "agents"].flatMap(
      (resource) => ["read", "create", "update", "delete"].map((action) => ({ resource, action })),
    ),
  };
}

type FixtureResult = { json: unknown } | { text: string } | null;

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

const CANVAS = "/api/v1/canvases/[^/]+";

interface Route {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, url: URL, method: string, body: unknown) => FixtureResult;
}

function buildRoutes(fixture: CanvasAppFixture): Route[] {
  const orgId = fixture.organizationId;
  const meUser = buildMeUser(orgId);

  return [
    { pattern: re("/api/v1/me"), resolve: () => ({ json: { user: meUser } }) },
    { pattern: re("/api/v1/triggers"), resolve: () => ({ json: fixture.triggers ?? { triggers: [] } }) },
    { pattern: re("/api/v1/actions"), resolve: () => ({ json: fixture.actions ?? { actions: [] } }) },
    { pattern: re("/api/v1/widgets"), resolve: () => ({ json: fixture.widgets ?? { widgets: [] } }) },
    { pattern: re("/api/v1/integrations"), resolve: () => ({ json: fixture.integrations ?? { integrations: [] } }) },
    { pattern: re("/api/v1/api-keys"), resolve: () => ({ json: { apiKeys: [] } }) },

    // Draft-version listing must stay empty (no open drafts); every other version
    // query returns the captured history. `?limit=1` resolves against the
    // `versionsLatest` slot when the fixture provides one; otherwise it falls
    // back to slicing the general versions list so a caller can seed both from
    // a single field.
    {
      pattern: re(`${CANVAS}/versions`),
      resolve: (_m, url) => {
        if (url.searchParams.get("state") === "STATE_DRAFT") {
          return { json: { versions: [], totalCount: 0, hasNextPage: false } };
        }
        const limit = Number.parseInt(url.searchParams.get("limit") ?? "", 10);
        if (limit === 1) {
          if (fixture.versionsLatest) return { json: fixture.versionsLatest };
          const versionsFallback = fixture.versions?.versions?.slice(0, 1) ?? [];
          return { json: { versions: versionsFallback, totalCount: versionsFallback.length, hasNextPage: false } };
        }
        return { json: fixture.versions ?? { versions: [], totalCount: 0, hasNextPage: false } };
      },
    },
    // Single-version detail reuses the live canvas spec (we only captured metadata
    // for the version list, which is all the versions sidebar needs).
    {
      pattern: re(`${CANVAS}/versions/([^/]+)`),
      resolve: (m) => ({
        json: {
          version: {
            metadata: { ...(fixture.versions?.versions?.[0]?.metadata ?? {}), id: m[1] },
            spec: fixture.canvas?.canvas?.spec ?? {},
          },
        },
      }),
    },

    // Run detail (`/runs/:runId`) is a distinct path from the list (`/runs`).
    {
      pattern: re(`${CANVAS}/runs/([^/]+)`),
      resolve: (m) => {
        const runId = m[1];
        const byId = fixture.runDetailsById?.[runId];
        if (byId) return { json: byId };
        const listed = listRuns(fixture).find((run) => run.id === runId);
        if (listed) return { json: { run: listed } };
        return { json: fixture.runDetail ?? {} };
      },
    },
    // For paginated `?before=…` requests we return an empty page so React
    // Query's infinite-scroll stops after the first batch. Widgets that ask
    // for larger sets configure their own `limit`; the console tab captures
    // enough runs on the first page to satisfy the visible panels.
    //
    // Honor `states` / `results` filters (exploded form style) so the running-
    // runs badge query (`states=STATE_STARTED`) does not see every run.
    {
      pattern: re(`${CANVAS}/runs`),
      resolve: (_m, url) => {
        if (url.searchParams.get("before")) {
          return { json: { runs: [], totalCount: 0, hasNextPage: false } };
        }
        const runs = filterRuns(listRuns(fixture), url);
        return {
          json: {
            runs,
            totalCount: runs.length,
            hasNextPage: false,
            lastTimestamp:
              typeof runs[runs.length - 1]?.createdAt === "string" ? runs[runs.length - 1]?.createdAt : undefined,
          },
        };
      },
    },
    {
      pattern: re(`${CANVAS}/events/([^/]+)/executions`),
      resolve: (m) => {
        const eventId = m[1];
        const byEvent = fixture.executionsByEventId?.[eventId];
        if (byEvent) return { json: byEvent };
        // Legacy single-capture fallback used by older fixtures / RunInspection.
        if (fixture.rootEventId && eventId === fixture.rootEventId) {
          return { json: fixture.executions ?? { executions: [] } };
        }
        return { json: { executions: [] } };
      },
    },
    // Real API shape is `{items: []}`; some legacy fixtures used `{memory: []}`
    // which no widget ever read successfully — normalize on `items` here.
    { pattern: re(`${CANVAS}/memory`), resolve: () => ({ json: fixture.memory ?? { items: [] } }) },
    // Files tab needs a ready repository before it will render the tree.
    // Without this, `useCanvasRepository` returns `undefined` and TanStack
    // Query surfaces `["canvases","repository",…] data is undefined`.
    {
      pattern: re(`${CANVAS}/repository/files`),
      resolve: () => ({
        json: {
          files: resolveRepositoryFilePaths(fixture).map((path) => ({ path })),
        },
      }),
    },
    {
      pattern: re(`${CANVAS}/repository/file`),
      resolve: (_m, url) => ({ text: resolveRepositoryFileContent(fixture, url.searchParams.get("path")) }),
    },
    {
      pattern: re(`${CANVAS}/repository`),
      resolve: () => ({
        json: {
          repository: {
            metadata: { canvasId: fixture.canvasId },
            status: { state: "STATE_READY", headSha: "storybook-fixture-head" },
          },
        },
      }),
    },
    { pattern: re(CANVAS), resolve: () => ({ json: fixture.canvas ?? { canvas: {} } }) },
    { pattern: re("/api/v1/canvases"), resolve: () => ({ json: { canvases: [], totalCount: 0, hasNextPage: false } }) },

    {
      pattern: re("/api/v1/agents/canvases/([^/]+)/chat/reset"),
      resolve: (_m, _url, method) => {
        if (method !== "POST") return null;
        return { json: fixture.agentChat ?? buildStorybookAgentChat(fixture.canvasId) };
      },
    },
    {
      pattern: re("/api/v1/agents/canvases/([^/]+)/chat"),
      resolve: () => ({ json: fixture.agentChat ?? buildStorybookAgentChat(fixture.canvasId) }),
    },
    {
      pattern: re("/api/v1/agents/chats/([^/]+)/messages"),
      resolve: (_m, _url, method, body) => {
        if (method === "POST") {
          const content =
            body && typeof body === "object" && "content" in body
              ? String((body as { content?: unknown }).content ?? "")
              : "";
          return {
            json: {
              message: {
                id: `storybook-sent-${Date.now()}`,
                role: "user",
                content,
                createdAt: new Date().toISOString(),
              },
            },
          };
        }
        return { json: fixture.agentMessages ?? buildStorybookAgentMessages() };
      },
    },
    {
      pattern: re("/api/v1/agents/chats/([^/]+)/interrupt"),
      resolve: (_m, _url, method) => (method === "POST" ? { json: {} } : null),
    },
    {
      pattern: re("/api/v1/agents/chats/([^/]+)/outcome"),
      resolve: (_m, _url, method) => (method === "POST" ? { json: {} } : null),
    },

    { pattern: re("/api/v1/organizations/[^/]+/integrations"), resolve: () => ({ json: { integrations: [] } }) },
    { pattern: re("/api/v1/organizations/[^/]+/usage"), resolve: () => ({ json: {} }) },
    { pattern: re("/api/v1/organizations/[^/]+/invite-link"), resolve: () => ({ json: {} }) },
    {
      pattern: re("/api/v1/organizations/[^/]+"),
      resolve: () => ({
        json: { organization: { metadata: { id: orgId, name: "SuperPlane" }, spec: {}, status: {} } },
      }),
    },

    // Non-versioned account endpoints hit outside the /api/v1 tree.
    {
      pattern: re("/account/experimental-features"),
      resolve: () => ({
        json: {
          features: [
            {
              id: "claude_managed_agents",
              label: "Managed agents",
              description: "Canvas agent chat",
              released: true,
            },
          ],
        },
      }),
    },
    { pattern: re("/account"), resolve: () => ({ json: { id: meUser.id, email: meUser.email, name: meUser.name } }) },
  ];
}

/** Match a registered canvas-app fixture route. Returns `null` when none match (no catch-all). */
export function matchCanvasAppFixture(
  url: URL,
  fixture?: CanvasAppFixture,
  method = "GET",
  body: unknown = undefined,
): FixtureResult {
  const activeFixture = fixture ?? defaultFixture;
  for (const route of buildRoutes(activeFixture)) {
    const match = route.pattern.exec(url.pathname);
    if (match) {
      return route.resolve(match, url, method, body);
    }
  }
  return null;
}

export { STORYBOOK_AGENT_CHAT_ID };

function emptyCanvasApiCatchAll(url: URL): FixtureResult {
  // Safety net: any other API call degrades gracefully to an empty object
  // instead of escaping to the network.
  if (url.pathname.startsWith("/api") || url.pathname.startsWith("/admin/api")) {
    return { json: {} };
  }
  return null;
}

function fixtureResponse(resolved: NonNullable<FixtureResult>): Response {
  if ("text" in resolved) {
    return new Response(resolved.text, { status: 200, headers: { "content-type": "text/plain" } });
  }
  return new Response(JSON.stringify(resolved.json), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

/**
 * Builds a `fetch` implementation that serves the captured canvas fixture
 * entirely in-process, falling back to `fallback` for non-API requests.
 *
 * The fixture is injected so different stories can render different apps
 * without touching this module. When omitted it defaults to the Software
 * Factory capture used by the `LiveCanvas`/`RunInspection` stories.
 *
 * We deliberately avoid MSW here: MSW intercepts via a Service Worker, which is
 * silently unavailable in non-secure contexts (e.g. opening Storybook through a
 * LAN IP like http://192.168.x.x:6006 instead of http://localhost:6006). This
 * override has no such dependency, so the stories render deterministic fake
 * data no matter how Storybook is accessed.
 */
export function createFixtureFetch(fallback: typeof fetch, fixture: CanvasAppFixture = defaultFixture): typeof fetch {
  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const method = (
      init?.method ??
      (typeof input !== "string" && !(input instanceof URL) ? input.method : "GET") ??
      "GET"
    ).toUpperCase();
    const resolved = matchCanvasAppFixture(url, fixture, method, parseRequestBody(init)) ?? emptyCanvasApiCatchAll(url);
    if (!resolved) {
      return fallback(input, init);
    }
    return fixtureResponse(resolved);
  };
  return impl as typeof fetch;
}

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

function parseRequestBody(init?: RequestInit): unknown {
  if (!init?.body || typeof init.body !== "string") return undefined;
  try {
    return JSON.parse(init.body);
  } catch {
    return undefined;
  }
}

function listRuns(fixture: CanvasAppFixture): Array<Record<string, unknown>> {
  return (fixture.runs?.runs ?? []) as Array<Record<string, unknown>>;
}

function filterRuns(runs: Array<Record<string, unknown>>, url: URL): Array<Record<string, unknown>> {
  const states = url.searchParams.getAll("states");
  const results = url.searchParams.getAll("results");
  return runs.filter((run) => {
    if (states.length > 0 && !states.includes(String(run.state ?? ""))) {
      return false;
    }
    if (results.length > 0 && !results.includes(String(run.result ?? ""))) {
      return false;
    }
    return true;
  });
}

function resolveRepositoryFilePaths(fixture: CanvasAppFixture): string[] {
  if (fixture.repositoryFilePaths?.length) {
    return [...fixture.repositoryFilePaths];
  }

  const paths = new Set<string>(DEFAULT_REPOSITORY_FILE_PATHS);
  for (const path of Object.keys(fixture.repositoryFileContents ?? {})) {
    paths.add(path);
  }
  return Array.from(paths).sort();
}

function resolveRepositoryFileContent(fixture: CanvasAppFixture, path: string | null): string {
  if (!path) {
    return "";
  }

  if (path === "console.yaml" && typeof fixture.consoleYaml === "string") {
    return fixture.consoleYaml;
  }

  return fixture.repositoryFileContents?.[path] ?? "";
}

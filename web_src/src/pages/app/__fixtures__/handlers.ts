import raw from "./canvasAppResponses.json";

// Shape of the captured fixture (see canvasAppResponses.json). Endpoint bodies
// are stored verbatim as the live API returned them (after trimming GitHub
// webhook payloads and dropping unused integration definitions), so they can be
// replayed straight back without reshaping.
interface CanvasAppFixture {
  canvasId: string;
  organizationId: string;
  versionId: string;
  publishedRunId: string;
  rootEventId: string;
  triggers: unknown;
  actions: unknown;
  widgets: unknown;
  integrations: unknown;
  canvas: { canvas?: { spec?: unknown } };
  versions: { versions?: Array<{ metadata?: Record<string, unknown> }> };
  runs: unknown;
  runDetail: unknown;
  executions: unknown;
}

const fixture = raw as CanvasAppFixture;

type FixtureExecution = Record<string, unknown> & { nodeId?: string; state?: string; result?: string };
type FixtureRun = {
  state?: string;
  result?: string;
  createdAt?: string;
  updatedAt?: string;
  rootEvent?: { id?: string };
  executions?: FixtureExecution[];
};

// The capture is a couple of days old, so a still-running run's elapsed time
// (now - createdAt) would read as days. Re-anchor each running run's start to a
// realistic few-minutes-ago so the "Running for …" / duration reads sensibly.
const runningRunElapsedMs = [3 * 60_000 + 20_000, 11 * 60_000 + 5_000, 47_000];
let runningRunSeen = 0;
for (const run of (fixture.runs as { runs?: FixtureRun[] }).runs ?? []) {
  if (run.state !== "STATE_STARTED") continue;
  const elapsed = runningRunElapsedMs[runningRunSeen % runningRunElapsedMs.length];
  runningRunSeen += 1;
  run.createdAt = new Date(Date.now() - elapsed).toISOString();
  run.updatedAt = new Date().toISOString();
}

// The capture stores each run's own step executions inline (reflecting that
// run's real state: passed runs are all-passed, running runs have an in-flight
// step, failed runs carry their errored step). Those inline executions are lean
// though — only the shared top-level `executions` capture has display extras
// (outputs/configuration/metadata). We key executions by the run's rootEvent id
// so the run-inspection panel serves the *selected* run's steps (previously
// every run replayed the same shared capture, so they all looked identical),
// enriching each step with the matching node's display extras.
const displayExtrasByNode = new Map<string, Partial<FixtureExecution>>();
for (const exec of (fixture.executions as { executions?: FixtureExecution[] }).executions ?? []) {
  if (typeof exec.nodeId === "string") {
    displayExtrasByNode.set(exec.nodeId, {
      outputs: exec.outputs,
      configuration: exec.configuration,
      metadata: exec.metadata,
    });
  }
}

// The capture only ever carried a single errored step per run. Promote a second
// finished step to errored on one failed run so the multi-error banner (one
// banner per errored step, rather than a single "+N more" summary) is
// demonstrable in the AppPage stories.
const failedRunWithExtraStep = (fixture.runs as { runs?: FixtureRun[] }).runs?.find(
  (run) =>
    run.result === "RESULT_FAILED" &&
    (run.executions ?? []).filter((exec) => exec.result === "RESULT_FAILED").length === 1 &&
    (run.executions ?? []).some((exec) => exec.state === "STATE_FINISHED" && exec.result !== "RESULT_FAILED"),
);
const secondErrorStep = failedRunWithExtraStep?.executions?.find(
  (exec) => exec.state === "STATE_FINISHED" && exec.result !== "RESULT_FAILED",
);
if (secondErrorStep) {
  secondErrorStep.result = "RESULT_FAILED";
  secondErrorStep.resultReason = "RESULT_REASON_ERROR";
  secondErrorStep.resultMessage = "Step failed: exited with code 1";
}

const executionsByEventId = new Map<string, FixtureExecution[]>();
for (const run of (fixture.runs as { runs?: FixtureRun[] }).runs ?? []) {
  const eventId = run.rootEvent?.id;
  if (!eventId || !run.executions) {
    continue;
  }
  executionsByEventId.set(
    eventId,
    run.executions.map((exec) => ({
      ...(typeof exec.nodeId === "string" ? displayExtrasByNode.get(exec.nodeId) : undefined),
      ...exec,
    })),
  );
}

export const canvasAppIds = {
  organizationId: fixture.organizationId,
  canvasId: fixture.canvasId,
  versionId: fixture.versionId,
  publishedRunId: fixture.publishedRunId,
  rootEventId: fixture.rootEventId,
};

const ORG = fixture.organizationId;

// A synthetic user with broad permissions so PermissionsProvider grants every
// canAct() check AppPage makes. We never capture the real user (email/token);
// only the permission strings matter for rendering.
const meUser = {
  id: "storybook-user",
  name: "Storybook User",
  email: "storybook@superplane.dev",
  organizationId: ORG,
  hasToken: true,
  roles: ["org_admin"],
  groups: [],
  permissions: ["canvases", "integrations", "secrets", "groups", "users", "roles", "organization"].flatMap((resource) =>
    ["read", "create", "update", "delete"].map((action) => ({ resource, action })),
  ),
};

type FixtureResult = { json: unknown } | { text: string } | null;

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

const CANVAS = "/api/v1/canvases/[^/]+";

// Route table mapping an API path (anchored regex) to its fixture body. Every
// pattern is fully anchored, so the entries are mutually exclusive and order
// doesn't matter. Anything not listed falls through to the catch-all in
// `resolveFixture`.
const routes: Array<{ pattern: RegExp; resolve: (match: RegExpExecArray, url: URL) => FixtureResult }> = [
  { pattern: re("/api/v1/me"), resolve: () => ({ json: { user: meUser } }) },
  { pattern: re("/api/v1/triggers"), resolve: () => ({ json: fixture.triggers }) },
  { pattern: re("/api/v1/actions"), resolve: () => ({ json: fixture.actions }) },
  { pattern: re("/api/v1/widgets"), resolve: () => ({ json: fixture.widgets }) },
  { pattern: re("/api/v1/integrations"), resolve: () => ({ json: fixture.integrations }) },
  { pattern: re("/api/v1/service-accounts"), resolve: () => ({ json: { serviceAccounts: [] } }) },

  // Draft-version listing must stay empty (no open drafts); every other version
  // query returns the published history.
  {
    pattern: re(`${CANVAS}/versions`),
    resolve: (_m, url) =>
      url.searchParams.get("state") === "STATE_DRAFT"
        ? { json: { versions: [], totalCount: 0, hasNextPage: false } }
        : { json: fixture.versions },
  },
  // Single-version detail reuses the live canvas spec (we only captured metadata
  // for the version list, which is all the versions sidebar needs).
  {
    pattern: re(`${CANVAS}/versions/([^/]+)`),
    resolve: (m) => ({
      json: {
        version: {
          metadata: { ...(fixture.versions.versions?.[0]?.metadata ?? {}), id: m[1] },
          spec: fixture.canvas.canvas?.spec ?? {},
        },
      },
    }),
  },

  // Run detail (`/runs/:runId`) is a distinct path from the list (`/runs`).
  { pattern: re(`${CANVAS}/runs/([^/]+)`), resolve: () => ({ json: fixture.runDetail }) },
  { pattern: re(`${CANVAS}/runs`), resolve: () => ({ json: fixture.runs }) },
  {
    pattern: re(`${CANVAS}/events/([^/]+)/executions`),
    resolve: (m) => {
      const executions = executionsByEventId.get(m[1]);
      return { json: executions ? { executions } : fixture.executions };
    },
  },
  { pattern: re(`${CANVAS}/memory`), resolve: () => ({ json: { memory: [] } }) },
  // Repository files (canvas.yaml / console.yaml) return raw text; empty content
  // means "no console dashboard configured".
  { pattern: re(`${CANVAS}/repository/file`), resolve: () => ({ text: "" }) },
  { pattern: re(CANVAS), resolve: () => ({ json: fixture.canvas }) },
  { pattern: re("/api/v1/canvases"), resolve: () => ({ json: { canvases: [], totalCount: 0, hasNextPage: false } }) },

  { pattern: re("/api/v1/organizations/[^/]+/integrations"), resolve: () => ({ json: { integrations: [] } }) },
  { pattern: re("/api/v1/organizations/[^/]+/usage"), resolve: () => ({ json: {} }) },
  { pattern: re("/api/v1/organizations/[^/]+/invite-link"), resolve: () => ({ json: {} }) },
  {
    pattern: re("/api/v1/organizations/[^/]+"),
    resolve: () => ({ json: { organization: { metadata: { id: ORG, name: "SuperPlane" }, spec: {}, status: {} } } }),
  },

  // Non-versioned account endpoints hit outside the /api/v1 tree.
  { pattern: re("/account/experimental-features"), resolve: () => ({ json: { features: [] } }) },
  { pattern: re("/account"), resolve: () => ({ json: { id: meUser.id, email: meUser.email, name: meUser.name } }) },
];

// Maps an API request to its fixture body. Returns `null` for anything that
// isn't an API call (assets, HMR, etc.) so the caller falls back to the real
// network.
function resolveFixture(url: URL): FixtureResult {
  for (const route of routes) {
    const match = route.pattern.exec(url.pathname);
    if (match) {
      return route.resolve(match, url);
    }
  }
  // Safety net: any other API call degrades gracefully to an empty object
  // instead of escaping to the network.
  if (url.pathname.startsWith("/api") || url.pathname.startsWith("/admin/api")) {
    return { json: {} };
  }
  return null;
}

/**
 * Builds a `fetch` implementation that serves the captured canvas fixture
 * entirely in-process, falling back to `fallback` for non-API requests.
 *
 * We deliberately avoid MSW here: MSW intercepts via a Service Worker, which is
 * silently unavailable in non-secure contexts (e.g. opening Storybook through a
 * LAN IP like http://192.168.x.x:6006 instead of http://localhost:6006). This
 * override has no such dependency, so the AppPage stories render deterministic
 * fake data no matter how Storybook is accessed.
 */
export function createFixtureFetch(fallback: typeof fetch): typeof fetch {
  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const resolved = resolveFixture(url);
    if (!resolved) {
      return fallback(input, init);
    }
    if ("text" in resolved) {
      return new Response(resolved.text, { status: 200, headers: { "content-type": "text/plain" } });
    }
    return new Response(JSON.stringify(resolved.json), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  };
  return impl as typeof fetch;
}

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

import defaultRaw from "./canvasAppResponses.json";
import softwareFactoryFixture from "./console/softwareFactory.json";
import cleanCodeAssessmentReadme from "./repository/cleanCodeAssessment.README.md?raw";

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
  runs?: unknown;
  /** GET /api/v1/canvases/{canvasId}/runs/{runId} */
  runDetail?: unknown;
  /** GET /api/v1/canvases/{canvasId}/events/{eventId}/executions */
  executions?: unknown;
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
}

const DEFAULT_REPOSITORY_FILE_PATHS = ["README.md", "canvas.yaml", "console.yaml"] as const;

const capturedFixture = defaultRaw as CanvasAppFixture;

const softwareFactory = softwareFactoryFixture as CanvasAppFixture;

// Live Canvas stories need a real console.yaml: `useCanvasConsole` treats an
// empty/missing file as `undefined`, which TanStack Query rejects. Reuse the
// Software Factory console (spotlight + scorecards + tables) so the default
// AppPage story showcases the factory dashboard; rewrite metadata canvasId to
// match this capture so repository/console reads stay consistent.
const defaultConsoleYaml =
  capturedFixture.consoleYaml ??
  (softwareFactory.consoleYaml ?? "").replaceAll(softwareFactory.canvasId, capturedFixture.canvasId);

const defaultFixture = {
  ...capturedFixture,
  consoleYaml: defaultConsoleYaml,
  memory: capturedFixture.memory ?? softwareFactory.memory,
  repositoryFileContents: {
    "README.md": cleanCodeAssessmentReadme,
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
    permissions: ["canvases", "integrations", "secrets", "groups", "users", "roles", "organization"].flatMap(
      (resource) => ["read", "create", "update", "delete"].map((action) => ({ resource, action })),
    ),
  };
}

type FixtureResult = { json: unknown } | { text: string } | null;

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

const CANVAS = "/api/v1/canvases/[^/]+";

interface Route {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, url: URL) => FixtureResult;
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
    { pattern: re(`${CANVAS}/runs/([^/]+)`), resolve: () => ({ json: fixture.runDetail ?? {} }) },
    // For paginated `?before=…` requests we return an empty page so React
    // Query's infinite-scroll stops after the first batch. Widgets that ask
    // for larger sets configure their own `limit`; the console tab captures
    // enough runs on the first page to satisfy the visible panels.
    {
      pattern: re(`${CANVAS}/runs`),
      resolve: (_m, url) => {
        if (url.searchParams.get("before")) {
          return { json: { runs: [], totalCount: 0, hasNextPage: false } };
        }
        return { json: fixture.runs ?? { runs: [], totalCount: 0, hasNextPage: false } };
      },
    },
    {
      pattern: re(`${CANVAS}/events/([^/]+)/executions`),
      resolve: () => ({ json: fixture.executions ?? { executions: [] } }),
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
    { pattern: re("/account/experimental-features"), resolve: () => ({ json: { features: [] } }) },
    { pattern: re("/account"), resolve: () => ({ json: { id: meUser.id, email: meUser.email, name: meUser.name } }) },
  ];
}

// Maps an API request to its fixture body. Returns `null` for anything that
// isn't an API call (assets, HMR, etc.) so the caller falls back to the real
// network.
function resolveFixture(url: URL, routes: Route[]): FixtureResult {
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
 * The fixture is injected so different stories can render different apps
 * without touching this module. When omitted it defaults to the Clean Code
 * Assessment capture used by the `LiveCanvas`/`RunInspection` stories.
 *
 * We deliberately avoid MSW here: MSW intercepts via a Service Worker, which is
 * silently unavailable in non-secure contexts (e.g. opening Storybook through a
 * LAN IP like http://192.168.x.x:6006 instead of http://localhost:6006). This
 * override has no such dependency, so the stories render deterministic fake
 * data no matter how Storybook is accessed.
 */
export function createFixtureFetch(fallback: typeof fetch, fixture: CanvasAppFixture = defaultFixture): typeof fetch {
  const routes = buildRoutes(fixture);
  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const resolved = resolveFixture(url, routes);
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

import { defaultHomePageFixture, type HomePageFixture } from "./homePageResponses";

export type { HomePageFixture };

export const homePageIds = {
  organizationId: defaultHomePageFixture.organizationId,
};

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

export type FixtureResult = { json: unknown } | { text: string } | null;

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

interface Route {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, url: URL, method: string) => FixtureResult;
}

function buildRoutes(fixture: HomePageFixture): Route[] {
  const orgId = fixture.organizationId;
  const meUser = buildMeUser(orgId);

  return [
    { pattern: re("/api/v1/me"), resolve: () => ({ json: { user: meUser } }) },
    {
      pattern: re("/api/v1/canvases"),
      resolve: (_m, _url, method) => {
        if (method === "POST") {
          return {
            json: {
              canvas: {
                metadata: {
                  id: "storybook-new-canvas",
                  name: "new-app",
                  organizationId: orgId,
                },
              },
            },
          };
        }
        return {
          json: {
            canvases: fixture.canvases,
            totalCount: fixture.canvases.length,
            hasNextPage: false,
          },
        };
      },
    },
    {
      pattern: re("/api/v1/canvases/[^/]+/preference"),
      resolve: () => ({ json: {} }),
    },
    {
      pattern: re("/api/v1/canvases/[^/]+"),
      resolve: (_m, _url, method) => {
        if (method === "DELETE" || method === "PUT" || method === "PATCH") {
          return { json: {} };
        }
        // GET canvas detail belongs to the AppPage fixture in OrgWorkspaceHarness.
        // Returning null lets the combined fetch fall through instead of serving
        // an empty `{ canvas: {} }` that blanks the live graph.
        return null;
      },
    },
    {
      pattern: re("/api/v1/canvas-folders"),
      resolve: (_m, _url, method) => {
        if (method === "POST") {
          return {
            json: {
              folder: {
                metadata: { id: "storybook-new-folder" },
                spec: { title: "New Folder", backgroundColor: "blue", canvases: [] },
              },
            },
          };
        }
        return { json: { folders: fixture.folders } };
      },
    },
    {
      pattern: re("/api/v1/canvas-folders/[^/]+/position"),
      resolve: () => ({ json: {} }),
    },
    {
      pattern: re("/api/v1/canvas-folders/[^/]+"),
      resolve: () => ({ json: {} }),
    },
    { pattern: re("/api/v1/organizations/[^/]+/integrations"), resolve: () => ({ json: { integrations: [] } }) },
    { pattern: re("/api/v1/organizations/[^/]+/usage"), resolve: () => ({ json: {} }) },
    { pattern: re("/api/v1/organizations/[^/]+/invite-link"), resolve: () => ({ json: {} }) },
    {
      pattern: re("/api/v1/organizations/[^/]+"),
      resolve: () => ({
        json: {
          organization: {
            metadata: { id: orgId, name: fixture.organizationName },
            spec: {},
            status: {},
          },
        },
      }),
    },
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
    {
      pattern: re("/account"),
      resolve: () => ({
        json: {
          id: meUser.id,
          email: meUser.email,
          name: meUser.name,
          organization_id: orgId,
        },
      }),
    },
    // Catalog install from ZeroStatePage
    {
      pattern: re("/apps/install"),
      resolve: () => ({
        json: { canvasId: "storybook-installed-canvas", organizationId: orgId },
      }),
    },
  ];
}

/** Match a registered home fixture route. Returns `null` when none match (no catch-all). */
export function matchHomePageFixture(
  url: URL,
  method: string,
  fixture: HomePageFixture = defaultHomePageFixture,
): FixtureResult {
  for (const route of buildRoutes(fixture)) {
    const match = route.pattern.exec(url.pathname);
    if (match) {
      return route.resolve(match, url, method);
    }
  }
  return null;
}

function emptyHomeApiCatchAll(url: URL): FixtureResult {
  if (
    url.pathname.startsWith("/api") ||
    url.pathname.startsWith("/admin/api") ||
    url.pathname === "/account" ||
    url.pathname.startsWith("/apps/")
  ) {
    return { json: {} };
  }
  return null;
}

export function requestMethod(input: RequestInfo | URL, init?: RequestInit): string {
  return (
    init?.method ??
    (typeof input !== "string" && !(input instanceof URL) ? input.method : "GET") ??
    "GET"
  ).toUpperCase();
}

export function fixtureResponse(resolved: NonNullable<FixtureResult>): Response {
  if ("text" in resolved) {
    return new Response(resolved.text, { status: 200, headers: { "content-type": "text/plain" } });
  }
  return new Response(JSON.stringify(resolved.json), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

/**
 * Builds a `fetch` implementation that serves the homepage fixture entirely
 * in-process. Same rationale as AppPage's fixture fetch: avoid MSW's Service
 * Worker dependency so Storybook works off-localhost too.
 */
export function createHomeFixtureFetch(
  fallback: typeof fetch,
  fixture: HomePageFixture = defaultHomePageFixture,
): typeof fetch {
  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const resolved = matchHomePageFixture(url, requestMethod(input, init), fixture) ?? emptyHomeApiCatchAll(url);
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

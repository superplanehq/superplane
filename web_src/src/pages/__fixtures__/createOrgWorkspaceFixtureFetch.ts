import { matchCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import {
  fixtureResponse,
  matchFactorySetupFixture,
  matchHomePageFixture,
  requestMethod,
  type HomePageFixture,
  type StorybookOrgIntegration,
} from "@/pages/home/__fixtures__/handlers";
import { defaultHomePageFixture } from "@/pages/home/__fixtures__/homePageResponses";

function emptyOrgWorkspaceCatchAll(url: URL): { json: unknown } | null {
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

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

/**
 * Serves both homepage and canvas-app fixtures from one `fetch` override so
 * Storybook can navigate between HomePage and AppPage without hitting the network.
 * Home routes win on overlap (e.g. canvas list, org, account).
 */
export function createOrgWorkspaceFixtureFetch(
  fallback: typeof fetch,
  options?: {
    homeFixture?: HomePageFixture;
    appFixture?: CanvasAppFixture;
  },
): typeof fetch {
  const homeFixture = options?.homeFixture ?? defaultHomePageFixture;
  const appFixture = options?.appFixture;
  const orgIntegrations: StorybookOrgIntegration[] = [];

  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const method = requestMethod(input, init);
    const body = parseRequestBody(init);
    const resolved = await resolveOrgWorkspaceFixture({
      url,
      method,
      input,
      init,
      body,
      homeFixture,
      appFixture,
      orgIntegrations,
    });
    if (!resolved) {
      return fallback(input, init);
    }
    return fixtureResponse(resolved);
  };
  return impl as typeof fetch;
}

function parseRequestBody(init?: RequestInit): unknown {
  if (!init?.body || typeof init.body !== "string") return undefined;
  try {
    return JSON.parse(init.body);
  } catch {
    return undefined;
  }
}

async function resolveOrgWorkspaceFixture(args: {
  url: URL;
  method: string;
  input: RequestInfo | URL;
  init?: RequestInit;
  body: unknown;
  homeFixture: HomePageFixture;
  appFixture?: CanvasAppFixture;
  orgIntegrations: StorybookOrgIntegration[];
}) {
  const { url, method, input, init, body, homeFixture, appFixture, orgIntegrations } = args;
  // Omit `appFixture` when unset so matchCanvasAppFixture uses its Software Factory default.
  const homeResolved = matchHomePageFixture(url, method, homeFixture);
  const canvasResolved = matchCanvasAppFixture(url, appFixture, method, body);
  const factoryResolved = await matchFactorySetupFixture(url, method, input, init, orgIntegrations);
  // AppPageHarness always supplies `appFixture`, so canvas integrations win there.
  // HomePageHarness omits it, so factory GitHub/Claude stubs stay available for setup.
  if (url.pathname === "/api/v1/integrations" && method === "GET" && appFixture !== undefined) {
    return canvasResolved ?? factoryResolved ?? homeResolved ?? emptyOrgWorkspaceCatchAll(url);
  }
  return factoryResolved ?? homeResolved ?? canvasResolved ?? emptyOrgWorkspaceCatchAll(url);
}

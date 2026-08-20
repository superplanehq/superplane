import {
  buildStorybookAgentChat,
  createStorybookAgentMessageStore,
  matchStorybookAgentMessageRoute,
} from "@/pages/app/__fixtures__/agentChatResponses";
import { matchCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import {
  factoriesOrganizationUsersResponse,
  matchFactoryPageFixture,
  type FactoriesFixture,
} from "@/pages/factories/__fixtures__/handlers";
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
    factoriesFixture?: FactoriesFixture;
    /** Organization connections the story starts with (e.g. an installed GitHub). */
    orgIntegrations?: StorybookOrgIntegration[];
  },
): typeof fetch {
  const homeFixture = options?.homeFixture ?? defaultHomePageFixture;
  const appFixture = options?.appFixture;
  // Factories handlers mutate the fixture in place (create/dispatch/close/assign),
  // so each fetch impl owns a private deep-clone. Otherwise interactive stories
  // would permanently alter the module-level `defaultFactoriesFixture` (and every
  // fixture that shares nested arrays with it).
  const factoriesFixture = options?.factoriesFixture ? structuredClone(options.factoriesFixture) : undefined;
  // Connect flows push into this list, so each fetch impl owns a private copy.
  const orgIntegrations: StorybookOrgIntegration[] = structuredClone(options?.orgIntegrations ?? []);
  const agentMessages = createStorybookAgentMessageStore(appFixture?.agentMessages?.messages);

  const impl = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(requestUrl(input), globalThis.location?.href ?? "http://localhost");
    const method = requestMethod(input, init);
    const body = await readRequestJson(input, init);
    const agentRoute = matchStorybookAgentMessageRoute({
      pathname: url.pathname,
      method,
      body,
      agentMessages,
      resetChat: () => appFixture?.agentChat ?? buildStorybookAgentChat(appFixture?.canvasId ?? ""),
    });
    if (agentRoute) {
      return fixtureResponse(agentRoute);
    }

    const resolved = await resolveOrgWorkspaceFixture({
      url,
      method,
      input,
      init,
      body,
      homeFixture,
      appFixture,
      factoriesFixture,
      orgIntegrations,
    });
    if (!resolved) {
      return fallback(input, init);
    }
    return fixtureResponse(resolved);
  };
  return impl as typeof fetch;
}

async function readRequestJson(input: RequestInfo | URL, init?: RequestInit): Promise<unknown> {
  try {
    if (typeof init?.body === "string" && init.body.trim()) {
      return JSON.parse(init.body);
    }
    if (typeof input !== "string" && !(input instanceof URL)) {
      return await input.clone().json();
    }
  } catch {
    return undefined;
  }
  return undefined;
}

function resolveFactoryFixtures(
  url: URL,
  method: string,
  body: unknown,
  factoriesFixture: FactoriesFixture | undefined,
) {
  if (!factoriesFixture) {
    return { factoryPagesResolved: null, factoryUsersResolved: null };
  }
  const factoryPagesResolved = matchFactoryPageFixture(
    url,
    method,
    (body ?? null) as Record<string, unknown> | null,
    factoriesFixture,
  );
  const factoryUsersResolved =
    url.pathname === "/api/v1/users" && method === "GET" ? factoriesOrganizationUsersResponse() : null;
  return { factoryPagesResolved, factoryUsersResolved };
}

async function resolveOrgWorkspaceFixture(args: {
  url: URL;
  method: string;
  input: RequestInfo | URL;
  init?: RequestInit;
  body: unknown;
  homeFixture: HomePageFixture;
  appFixture?: CanvasAppFixture;
  factoriesFixture?: FactoriesFixture;
  orgIntegrations: StorybookOrgIntegration[];
}) {
  const { url, method, input, init, body, homeFixture, appFixture, factoriesFixture, orgIntegrations } = args;
  // Omit `appFixture` when unset so matchCanvasAppFixture uses its Software Factory default.
  const homeResolved = matchHomePageFixture(url, method, homeFixture);
  const canvasResolved = matchCanvasAppFixture(url, appFixture, method, body);
  const factorySetupResolved = await matchFactorySetupFixture(url, method, input, init, orgIntegrations);
  const { factoryPagesResolved, factoryUsersResolved } = resolveFactoryFixtures(url, method, body, factoriesFixture);
  // Canvas integrations win only for fixtures that declare them. Factory
  // stories carry a canvas fixture without integrations, so the GitHub/Claude
  // definitions workspace setup needs stay available.
  if (url.pathname === "/api/v1/integrations" && method === "GET" && appFixture?.integrations !== undefined) {
    return canvasResolved ?? factorySetupResolved ?? homeResolved ?? emptyOrgWorkspaceCatchAll(url);
  }
  return (
    factoryPagesResolved ??
    factoryUsersResolved ??
    factorySetupResolved ??
    homeResolved ??
    canvasResolved ??
    emptyOrgWorkspaceCatchAll(url)
  );
}

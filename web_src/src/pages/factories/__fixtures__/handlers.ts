import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  OPERATOR_USER,
  ORGANIZATION_USERS,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  type FactoriesFixture,
} from "./factoryPageResponses";
import type { FactoriesWorkOrder } from "@/api-client";
import { fixtureResponse, type FixtureResult, type StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";

export type { FactoriesFixture };

export const factoryPageIds = {
  organizationId: defaultFactoriesFixture.organizationId,
};

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

interface FactoriesRoute {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, method: string, body: Record<string, unknown> | null) => FixtureResult;
}

interface RequestBody {
  name?: unknown;
  title?: unknown;
  description?: unknown;
  assigneeIds?: unknown;
  assignee_ids?: unknown;
  lineName?: unknown;
  line_name?: unknown;
  result?: unknown;
  steps?: unknown;
}

function stringOrEmpty(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function stringArrayOrEmpty(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((entry): entry is string => typeof entry === "string") : [];
}

function findUsersByIds(ids: string[]) {
  const byId = new Map(ORGANIZATION_USERS.map((user) => [user.id, user]));
  return ids
    .map((id) => byId.get(id))
    .filter((user): user is (typeof ORGANIZATION_USERS)[number] => Boolean(user))
    .map((user) => ({ id: user.id, name: user.name }));
}

function ensureFactoryWorkOrders(fixture: FactoriesFixture, factoryId: string): FactoriesWorkOrder[] {
  const existing = fixture.workOrdersByFactoryId[factoryId];
  if (existing) {
    return existing;
  }
  const created: FactoriesWorkOrder[] = [];
  fixture.workOrdersByFactoryId[factoryId] = created;
  return created;
}

function factoriesCollectionRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/factories"),
    resolve: (_match, method, body) => {
      if (method !== "POST") {
        return { json: { factories: fixture.factories } };
      }
      const request = (body ?? {}) as RequestBody;
      const created = {
        id: `storybook-factory-${fixture.factories.length + 1}`,
        name: stringOrEmpty(request.name) || "New workspace",
        description: stringOrEmpty(request.description),
        lines: [],
      };
      fixture.factories.push(created);
      fixture.workOrdersByFactoryId[created.id] = [];
      fixture.appsByFactoryId[created.id] = [];
      return { json: { factory: created } };
    },
  };
}

function factoryDetailRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    {
      pattern: re("/api/v1/factories/([^/]+)"),
      resolve: (match, method, body) => {
        const factoryId = match[1];
        const factoryIndex = fixture.factories.findIndex((entry) => entry.id === factoryId);
        const factory = factoryIndex >= 0 ? fixture.factories[factoryIndex] : undefined;

        if (method === "PUT") {
          if (!factory) return { json: {} };
          const request = (body ?? {}) as RequestBody;
          if (typeof request.name === "string" && request.name.trim()) {
            factory.name = request.name.trim();
          }
          if (typeof request.description === "string") {
            factory.description = request.description;
          }
          return { json: { factory } };
        }

        if (method === "DELETE") {
          if (factoryIndex < 0) return { json: {} };
          fixture.factories.splice(factoryIndex, 1);
          delete fixture.workOrdersByFactoryId[factoryId];
          delete fixture.appsByFactoryId[factoryId];
          return { json: {} };
        }

        return factory ? { json: { factory } } : { json: {} };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/apps"),
      resolve: (match) => ({ json: { apps: fixture.appsByFactoryId[match[1]] ?? [] } }),
    },
  ];
}

function factoryLinesRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    {
      pattern: re("/api/v1/factories/([^/]+)/lines"),
      resolve: (match, method, body) => {
        if (method !== "POST") return null;
        const request = (body ?? {}) as RequestBody;
        const factory = fixture.factories.find((entry) => entry.id === match[1]);
        if (!factory) return { json: {} };
        const line = {
          id: `storybook-line-${(factory.lines?.length ?? 0) + 1}`,
          name: stringOrEmpty(request.name) || "new-line",
          steps: Array.isArray(request.steps) ? request.steps : [],
        };
        factory.lines = [...(factory.lines ?? []), line];
        return { json: { line } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/lines/([^/]+)"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        const request = (body ?? {}) as RequestBody;
        const factory = fixture.factories.find((entry) => entry.id === match[1]);
        const existing = factory?.lines?.find((entry) => entry.id === match[2]);
        if (!existing) return { json: {} };
        existing.name = stringOrEmpty(request.name) || existing.name;
        if (Array.isArray(request.steps)) {
          existing.steps = request.steps as typeof existing.steps;
        }
        return { json: { line: existing } };
      },
    },
  ];
}

function createWorkOrderFromRequest(request: RequestBody, orderCount: number): FactoriesWorkOrder {
  const nowIso = new Date().toISOString();
  return {
    id: `storybook-work-order-${orderCount + 1}`,
    title: stringOrEmpty(request.title) || "New work order",
    description: stringOrEmpty(request.description),
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    createdAt: nowIso,
    updatedAt: nowIso,
    createdBy: { user: { id: ORGANIZATION_USERS[0].id, name: ORGANIZATION_USERS[0].name } },
    assignees: findUsersByIds(stringArrayOrEmpty(request.assigneeIds ?? request.assignee_ids)),
    executions: [],
  };
}

function findOrder(fixture: FactoriesFixture, factoryId: string, orderId: string) {
  const orders = fixture.workOrdersByFactoryId[factoryId] ?? [];
  return orders.find((entry) => entry.id === orderId);
}

function dispatchOrder(fixture: FactoriesFixture, factoryId: string, orderId: string, request: RequestBody) {
  const order = findOrder(fixture, factoryId, orderId);
  if (!order) return { json: {} };
  const factory = fixture.factories.find((entry) => entry.id === factoryId);
  const lineName = stringOrEmpty(request.lineName ?? request.line_name);
  const line = factory?.lines?.find((entry) => entry.name === lineName) ?? factory?.lines?.[0];
  order.updatedAt = new Date().toISOString();
  order.executions = [
    ...(order.executions ?? []),
    {
      id: `dispatch-${Date.now()}`,
      line: line ? { id: line.id, name: line.name } : { id: "line-unknown", name: lineName },
      step: line?.steps?.[0]?.name ?? "start",
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
  ];
  return { json: { order } };
}

function workOrderRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    {
      pattern: re("/api/v1/factories/([^/]+)/orders"),
      resolve: (match, method, body) => {
        const orders = ensureFactoryWorkOrders(fixture, match[1]);
        if (method !== "POST") return { json: { orders } };
        const created = createWorkOrderFromRequest((body ?? {}) as RequestBody, orders.length);
        orders.unshift(created);
        return { json: { order: created } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)"),
      resolve: (match) => {
        const order = findOrder(fixture, match[1], match[2]);
        return order ? { json: { order } } : { json: {} };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/assignees"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        const request = (body ?? {}) as RequestBody;
        order.assignees = findUsersByIds(stringArrayOrEmpty(request.assigneeIds ?? request.assignee_ids));
        order.updatedAt = new Date().toISOString();
        return { json: { order } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/dispatch"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        return dispatchOrder(fixture, match[1], match[2], (body ?? {}) as RequestBody);
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/close"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        const request = (body ?? {}) as RequestBody;
        order.state = "STATE_CLOSED";
        order.result = stringOrEmpty(request.result) === "RESULT_REJECTED" ? "RESULT_REJECTED" : "RESULT_COMPLETED";
        order.updatedAt = new Date().toISOString();
        return { json: { order } };
      },
    },
  ];
}

/** Builds a resolvable factories route table for a fixture snapshot. */
function buildRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    factoriesCollectionRoute(fixture),
    ...factoryDetailRoutes(fixture),
    ...factoryLinesRoutes(fixture),
    ...workOrderRoutes(fixture),
  ];
}

/**
 * Resolves a factories API request against a fixture snapshot. Returns `null`
 * when no route matches so the caller can fall through to another matcher.
 */
export function matchFactoryPageFixture(
  url: URL,
  method: string,
  body: Record<string, unknown> | null,
  fixture: FactoriesFixture = defaultFactoriesFixture,
): FixtureResult {
  for (const route of buildRoutes(fixture)) {
    const match = route.pattern.exec(url.pathname);
    if (match) {
      return route.resolve(match, method, body);
    }
  }
  return null;
}

const STORYBOOK_USER_ROLES: Record<string, { roleName: string; roleDisplayName: string }> = {
  [STORYBOOK_ME_USER_ID]: { roleName: "org_owner", roleDisplayName: "Owner" },
  [REVIEWER_USER.id]: { roleName: "org_admin", roleDisplayName: "Admin" },
  [OPERATOR_USER.id]: { roleName: "org_member", roleDisplayName: "Member" },
};

const STORYBOOK_WORKSPACE_ROLES = [
  { metadata: { name: "org_owner" }, spec: { displayName: "Owner" } },
  { metadata: { name: "org_admin" }, spec: { displayName: "Admin" } },
  { metadata: { name: "org_member" }, spec: { displayName: "Member" } },
];

const STORYBOOK_WORKSPACE_SECRETS = [
  {
    metadata: { id: "secret-openai", name: "openai" },
    spec: { provider: "PROVIDER_LOCAL", local: { data: { OPENAI_API_KEY: "sk-storybook" } } },
  },
  {
    metadata: { id: "secret-github", name: "github-app" },
    spec: {
      provider: "PROVIDER_LOCAL",
      local: { data: { GITHUB_TOKEN: "ghs-storybook", APP_ID: "12345" } },
    },
  },
];

const STORYBOOK_INVITE_LINK = {
  id: "storybook-invite",
  organizationId: FACTORIES_ORGANIZATION_ID,
  token: "storybook-invite-token",
  enabled: true,
};

function catalogIntegration(name: string, label: string, description: string, options?: { legacySetupOnly?: boolean }) {
  return {
    name,
    label,
    icon: name,
    description,
    configuration: [],
    instructions: "",
    legacySetupOnly: options?.legacySetupOnly ?? true,
  };
}

/** Same catalog the organization Integrations page lists in the live app. */
export const STORYBOOK_WORKSPACE_INTEGRATION_CATALOG = [
  catalogIntegration("github", "GitHub", "Manage and react to changes in your GitHub repositories", {
    legacySetupOnly: false,
  }),
  catalogIntegration("gitlab", "GitLab", "Manage and react to changes in your GitLab repositories"),
  catalogIntegration("bitbucket", "Bitbucket", "Manage and react to changes in your Bitbucket repositories"),
  catalogIntegration("slack", "Slack", "Send and react to Slack messages and interactions"),
  catalogIntegration("linear", "Linear", "Manage and react to issues in Linear"),
  catalogIntegration("jira", "Jira", "Manage issues in Jira"),
  catalogIntegration("claude", "Claude", "Use Claude models in workflows"),
  catalogIntegration("openai", "OpenAI", "Generate text responses with OpenAI models"),
  catalogIntegration("cursor", "Cursor", "Build workflows with Cursor AI Agents and track usage"),
  catalogIntegration("semaphore", "Semaphore", "Run and react to your Semaphore workflows"),
  catalogIntegration("circleci", "CircleCI", "Run and react to CircleCI pipelines"),
  catalogIntegration("datadog", "Datadog", "Create events in Datadog"),
  catalogIntegration("sentry", "Sentry", "React to issue events and manage issues and metric alerts in Sentry"),
  catalogIntegration("pagerduty", "PagerDuty", "Manage and react to incidents in PagerDuty"),
  catalogIntegration("grafana", "Grafana", "React to Grafana alerts and annotations"),
  catalogIntegration("aws", "AWS", "Manage resources and execute AWS commands in workflows"),
  catalogIntegration("gcp", "Google Cloud", "Manage and use Google Cloud resources in your workflows"),
  catalogIntegration("azure", "Azure", "Manage Azure resources in workflows"),
  catalogIntegration("digitalocean", "DigitalOcean", "Manage DigitalOcean resources in workflows"),
  catalogIntegration("dockerhub", "DockerHub", "React to image pushes and manage Docker Hub repositories"),
  catalogIntegration("cloudflare", "Cloudflare", "Manage Cloudflare resources and react to events"),
  catalogIntegration("discord", "Discord", "Send and react to Discord messages"),
  catalogIntegration("teams", "Microsoft Teams", "Send and react to Microsoft Teams messages"),
  catalogIntegration("incident", "Incident.io", "Manage and react to incidents"),
  catalogIntegration("rootly", "Rootly", "Manage and react to incidents in Rootly"),
  catalogIntegration("firehydrant", "FireHydrant", "Manage and react to incidents in FireHydrant"),
  catalogIntegration("launchdarkly", "LaunchDarkly", "Manage feature flags in LaunchDarkly"),
  catalogIntegration("honeycomb", "Honeycomb", "React to Honeycomb triggers and burn alerts"),
  catalogIntegration("newrelic", "New Relic", "React to New Relic alerts"),
  catalogIntegration("elastic", "Elastic", "Search and react to Elastic events"),
  catalogIntegration("statuspage", "Statuspage", "Update Statuspage incidents and components"),
  catalogIntegration("sendgrid", "SendGrid", "Send email with SendGrid"),
  catalogIntegration("smtp", "SMTP", "Send email through an SMTP server"),
  catalogIntegration("perplexity", "Perplexity", "Use Perplexity models in workflows"),
  catalogIntegration("daytona", "Daytona", "Create and manage Daytona sandboxes"),
  catalogIntegration("dash0", "Dash0", "Query Dash0 and react to alerts"),
  catalogIntegration("coolify", "Coolify", "Manage Coolify applications and deployments"),
  catalogIntegration("cloudsmith", "Cloudsmith", "Manage Cloudsmith packages"),
  catalogIntegration("harness", "Harness", "Run and react to Harness pipelines"),
  catalogIntegration("hetzner", "Hetzner", "Manage Hetzner Cloud resources"),
  catalogIntegration("oci", "OCI", "Manage Oracle Cloud resources"),
  catalogIntegration("octopus", "Octopus Deploy", "Run and react to Octopus Deploy releases"),
  catalogIntegration("prometheus", "Prometheus", "Query Prometheus and react to alerts"),
  catalogIntegration("render", "Render", "Manage Render services and deploys"),
  catalogIntegration("servicenow", "ServiceNow", "Create and update ServiceNow records"),
  catalogIntegration("telegram", "Telegram", "Send and react to Telegram messages"),
  catalogIntegration("logfire", "Logfire", "React to Logfire alerts"),
  catalogIntegration("jfrogArtifactory", "JFrog Artifactory", "Manage JFrog Artifactory packages"),
];

export const STORYBOOK_CONNECTED_ORG_INTEGRATIONS: StorybookOrgIntegration[] = [
  {
    metadata: { id: "int-github-acme", name: "github", integrationName: "github" },
    status: { state: "ready" },
  },
  {
    metadata: { id: "int-claude-workspace", name: "claude", integrationName: "claude" },
    status: { state: "ready" },
  },
];

/** Response body for `GET /api/v1/users` (`useOrganizationUsers`). */
export function factoriesOrganizationUsersResponse(): FixtureResult {
  return {
    json: {
      users: ORGANIZATION_USERS.map((user) => {
        const role = STORYBOOK_USER_ROLES[user.id];
        return {
          metadata: { id: user.id, email: user.email },
          spec: { displayName: user.name },
          status: role
            ? {
                roles: [
                  {
                    roleName: role.roleName,
                    roleDisplayName: role.roleDisplayName,
                  },
                ],
              }
            : {},
        };
      }),
    },
  };
}

/**
 * Roles, secrets, and invite-link fixtures for Storybook workspace settings.
 * Factory page routes stay factory-scoped; these org APIs back the reused
 * Members / Integrations / Secrets pages.
 */
export function matchFactorySettingsFixture(url: URL, method: string): FixtureResult {
  if (url.pathname === "/api/v1/integrations" && method === "GET") {
    return { json: { integrations: STORYBOOK_WORKSPACE_INTEGRATION_CATALOG } };
  }

  if (/^\/api\/v1\/organizations\/[^/]+\/integrations$/.test(url.pathname) && method === "GET") {
    return { json: { integrations: STORYBOOK_CONNECTED_ORG_INTEGRATIONS } };
  }

  if (url.pathname === "/api/v1/roles" && method === "GET") {
    return { json: { roles: STORYBOOK_WORKSPACE_ROLES } };
  }

  if (url.pathname === "/api/v1/secrets" && method === "GET") {
    return { json: { secrets: STORYBOOK_WORKSPACE_SECRETS } };
  }

  const secretMatch = /^\/api\/v1\/secrets\/([^/]+)$/.exec(url.pathname);
  if (secretMatch && method === "GET") {
    const secret = STORYBOOK_WORKSPACE_SECRETS.find((entry) => entry.metadata.id === secretMatch[1]);
    return secret ? { json: { secret } } : { json: {} };
  }

  if (/^\/api\/v1\/organizations\/[^/]+\/invite-link$/.test(url.pathname)) {
    return { json: { inviteLink: STORYBOOK_INVITE_LINK } };
  }

  return null;
}

/**
 * Convenience for tests that want to exercise the matcher directly. Deep-clones
 * `defaultFactoriesFixture` so mutating calls do not leak between tests.
 */
export async function fetchFactoryPageFixture(
  path: string,
  options?: RequestInit,
  fixture: FactoriesFixture = structuredClone(defaultFactoriesFixture),
): Promise<Response> {
  const url = new URL(path, "http://localhost");
  const method = (options?.method ?? "GET").toUpperCase();
  const body =
    typeof options?.body === "string" && options.body.trim()
      ? (JSON.parse(options.body) as Record<string, unknown>)
      : null;
  const result =
    matchFactoryPageFixture(url, method, body, fixture) ??
    (url.pathname === "/api/v1/users" && method === "GET" ? factoriesOrganizationUsersResponse() : null) ??
    matchFactorySettingsFixture(url, method);
  if (!result) {
    return new Response(null, { status: 404 });
  }
  return fixtureResponse(result);
}

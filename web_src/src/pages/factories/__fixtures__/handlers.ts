import { defaultFactoriesFixture, ORGANIZATION_USERS, type FactoriesFixture } from "./factoryPageResponses";
import type { FactoriesWorkOrder } from "@/api-client";
import { fixtureResponse, type FixtureResult } from "@/pages/home/__fixtures__/handlers";
import { LINE_LIST_METRICS_BY_ID } from "../pages/lineListMetricsMockData";

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
      pattern: re("/api/v1/factories/([^/]+)/line-metrics"),
      resolve: (match) => {
        const factory = fixture.factories.find((entry) => entry.id === match[1]);
        const lines = (factory?.lines ?? []).flatMap((line) => {
          if (!line.id) {
            return [];
          }
          const metrics = LINE_LIST_METRICS_BY_ID[line.id];
          return [
            {
              lineId: line.id,
              ...(metrics ? { metrics } : {}),
            },
          ];
        });
        return { json: { lines } };
      },
    },
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
    number: String(200 + orderCount + 1),
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

/** Response body for `GET /api/v1/users` (`useOrganizationUsers`). */
export function factoriesOrganizationUsersResponse(): FixtureResult {
  return {
    json: {
      users: ORGANIZATION_USERS.map((user) => ({
        metadata: { id: user.id, email: user.email },
        spec: { displayName: user.name },
      })),
    },
  };
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
  const result = matchFactoryPageFixture(url, method, body, fixture);
  if (!result) {
    return new Response(null, { status: 404 });
  }
  return fixtureResponse(result);
}

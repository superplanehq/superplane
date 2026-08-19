import {
  defaultFactoriesFixture,
  ORGANIZATION_USERS,
  STORYBOOK_ME_USER_EMAIL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  type FactoriesFixture,
} from "./factoryPageResponses";
import { DEFAULT_ARTIFACTS_BY_ORDER_ID, DEFAULT_EVENTS_BY_ORDER_ID } from "./factoryPageEventFixtures";
import { DEFAULT_CHECKS_BY_ORDER_ID } from "./workOrderCheckFixtures";
import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { defaultNotificationSettings } from "@/lib/notificationSettings";
import { buildStorybookMeUser, fixtureResponse, type FixtureResult } from "@/pages/home/__fixtures__/handlers";
import { automationNameForLineStep } from "../lib/factoryLineFormShared";

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
  body?: unknown;
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
    number: String(200 + orderCount + 1),
    title: stringOrEmpty(request.title) || "New work order",
    description: stringOrEmpty(request.description),
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    createdAt: nowIso,
    updatedAt: nowIso,
    createdBy: { user: { id: ORGANIZATION_USERS[0].id, name: ORGANIZATION_USERS[0].name } },
    assignees: findUsersByIds(stringArrayOrEmpty(request.assigneeIds ?? request.assignee_ids)),
    lineDispatches: [],
  };
}

function findOrder(fixture: FactoriesFixture, factoryId: string, orderId: string) {
  const orders = fixture.workOrdersByFactoryId[factoryId] ?? [];
  return orders.find((entry) => entry.id === orderId);
}

function buildDispatchedLineDispatch(
  line: FactoriesFactoryLine | undefined,
  lineName: string,
  now: string,
  apps: Array<{ id?: string; name?: string }> = [],
): FactoriesWorkOrderLineDispatch {
  const firstStep = line?.steps?.[0];
  const firstAppId = firstStep?.app?.app;
  const firstName = automationNameForLineStep(firstStep, apps, 0);
  return {
    id: `dispatch-${Date.now()}`,
    line: line ? { id: line.id, name: line.name } : { id: "line-unknown", name: lineName },
    steps: (line?.steps ?? []).map((step, index) => ({
      name: automationNameForLineStep(step, apps, index),
      stepIndex: index,
    })),
    state: "STATE_ACTIVE",
    result: "RESULT_UNKNOWN",
    createdAt: now,
    stepExecutions: [
      {
        id: `exec-${Date.now()}`,
        step: firstName,
        stepIndex: 0,
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        createdAt: now,
        updatedAt: now,
        run: firstAppId ? { appId: firstAppId, appName: firstName } : undefined,
      },
    ],
  };
}

function orderEvents(fixture: FactoriesFixture, orderId: string): FactoriesWorkOrderEvent[] {
  return fixture.eventsByOrderId?.[orderId] ?? DEFAULT_EVENTS_BY_ORDER_ID[orderId] ?? [];
}

function dispatchOrder(fixture: FactoriesFixture, factoryId: string, orderId: string, request: RequestBody) {
  const order = findOrder(fixture, factoryId, orderId);
  if (!order) return { json: {} };
  const factory = fixture.factories.find((entry) => entry.id === factoryId);
  const lineName = stringOrEmpty(request.lineName ?? request.line_name);
  const line = factory?.lines?.find((entry) => entry.name === lineName) ?? factory?.lines?.[0];
  const now = new Date().toISOString();
  order.updatedAt = now;

  const apps = fixture.appsByFactoryId[factoryId] ?? [];
  const newDispatch = buildDispatchedLineDispatch(line, lineName, now, apps);
  order.lineDispatches = [...(order.lineDispatches ?? []), newDispatch];
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
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/events"),
      resolve: (match) => {
        const events = orderEvents(fixture, match[2]);
        return { json: { events, totalCount: events.length, hasNextPage: false } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/artifacts"),
      resolve: (match, method) => {
        if (method !== "GET") return { json: {} };
        const artifacts = fixture.artifactsByOrderId?.[match[2]] ?? DEFAULT_ARTIFACTS_BY_ORDER_ID[match[2]] ?? [];
        return { json: { artifacts } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/checks"),
      resolve: (match, method) => {
        if (method !== "GET") return { json: {} };
        const checks = fixture.checksByOrderId?.[match[2]] ?? DEFAULT_CHECKS_BY_ORDER_ID[match[2]] ?? [];
        return { json: { checks } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/comments"),
      resolve: (match, method, body) => {
        if (method !== "POST") return null;
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        const commentBody = stringOrEmpty(((body ?? {}) as RequestBody).body);
        const timestamp = new Date().toISOString();
        const events: FactoriesWorkOrderEvent[] = [
          ...orderEvents(fixture, match[2]),
          {
            type: "order.comment.added",
            timestamp,
            event: {
              order: { id: order.id, title: order.title },
              body: commentBody,
              author: { kind: "user", userId: STORYBOOK_ME_USER_ID },
            },
          },
        ];
        fixture.eventsByOrderId = { ...(fixture.eventsByOrderId ?? {}), [match[2]]: events };
        return { json: { comment: { id: `comment-${events.length}`, body: commentBody, createdAt: timestamp } } };
      },
    },
  ];
}

/** Serves `/api/v1/me` so factory stories resolve `useMe` without the Home harness. */
function meRoute(organizationId: string): FactoriesRoute {
  return {
    pattern: re("/api/v1/me"),
    resolve: () => ({
      json: {
        user: {
          ...buildStorybookMeUser(organizationId),
          id: STORYBOOK_ME_USER_ID,
          name: STORYBOOK_ME_USER_NAME,
          email: STORYBOOK_ME_USER_EMAIL,
        },
      },
    }),
  };
}

function notificationSettingsRoute(fixture: FactoriesFixture): FactoriesRoute {
  const defaults = defaultNotificationSettings();

  return {
    pattern: re("/api/v1/me/notification-settings"),
    resolve: (_match, method, body) => {
      if (method === "PUT") {
        const request = (body ?? {}) as { settings?: FactoriesFixture["notificationSettings"] };
        fixture.notificationSettings = { ...defaults, ...(request.settings ?? {}) };
      }
      return { json: { settings: fixture.notificationSettings ?? defaults } };
    },
  };
}

/** Builds a resolvable factories route table for a fixture snapshot. */
function buildRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    factoriesCollectionRoute(fixture),
    meRoute(fixture.organizationId),
    notificationSettingsRoute(fixture),
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

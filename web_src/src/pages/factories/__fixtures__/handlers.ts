import { EMPTY_USAGE_REPORT } from "./usageReportFixtures";
import { EMPTY_FACTORY_VELOCITY } from "./velocityReportFixtures";
import { factoryIntakeRoutes } from "./factoryIntakeHandlers";
import {
  defaultFactoriesFixture,
  ORGANIZATION_USERS,
  STORYBOOK_ME_USER_EMAIL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  toStorybookOrganizationUser,
  type FactoriesFixture,
} from "./factoryPageResponses";
import {
  DEFAULT_ARTIFACTS_BY_ORDER_ID,
  DEFAULT_EVENTS_BY_ORDER_ID,
  DEFAULT_PULL_REQUESTS_BY_ORDER_ID,
} from "./factoryPageEventFixtures";
import { DEFAULT_CHECKS_BY_ORDER_ID } from "./workOrderCheckFixtures";
import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesFactoryOnboarding,
  FactoriesFactoryPullRequest,
  FactoriesUpdateFactoryAgentBody,
  FactoriesUpdateFactoryOnboardingBody,
  FactoriesWorkOrder,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { defaultNotificationSettings } from "@/lib/notificationSettings";
import { buildStorybookMeUser, fixtureResponse, type FixtureResult } from "@/pages/home/__fixtures__/handlers";
import { storybookHostedLlmModels } from "@/pages/home/__fixtures__/hostedLlmModels";
import { automationNameForLineStep } from "../lib/factoryLineFormShared";
import { isValidWorkspaceKey, suggestWorkspaceKeyFromName, WORKSPACE_KEY_MAX_LENGTH } from "../lib/workspaceKey";
import { metricsForLine } from "../pages/lineListMetricsMockData";

export type { FactoriesFixture };

const re = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

/**
 * When each workspace last had its velocity synced in this session.
 *
 * The real sync runs in a background worker and the page waits for the stored
 * sync time to move, so the fixtures record the time a sync was asked for.
 * Without it the page would wait for a sync that never reports back.
 */
const velocitySyncedAt = new Map<string, string>();

function velocitySynced(factoryId: string): void {
  velocitySyncedAt.set(factoryId, new Date().toISOString());
}

interface FactoriesRoute {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, method: string, body: Record<string, unknown> | null, url: URL) => FixtureResult;
}

interface RequestBody {
  name?: unknown;
  title?: unknown;
  description?: unknown;
  key?: unknown;
  hostedSpendBudgetCents?: unknown;
  clearHostedSpendBudget?: unknown;
  assigneeIds?: unknown;
  assignee_ids?: unknown;
  lineName?: unknown;
  line_name?: unknown;
  startStepIndex?: unknown;
  start_step_index?: unknown;
  replaceActive?: unknown;
  replace_active?: unknown;
  result?: unknown;
  state?: unknown;
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

function takenFactoryKeys(factories: FactoriesFactory[]): Set<string> {
  return new Set(factories.map((factory) => factory.key).filter((key): key is string => Boolean(key)));
}

/** Same letter-only keys the live API derives when the client omits `key`. */
function unusedFactoryKey(factories: FactoriesFactory[], name: string, requestedKey: string): string {
  const taken = takenFactoryKeys(factories);
  const seed = isValidWorkspaceKey(requestedKey) ? requestedKey : suggestWorkspaceKeyFromName(name) || "WS";
  if (!taken.has(seed)) {
    return seed;
  }

  const prefix = seed.slice(0, WORKSPACE_KEY_MAX_LENGTH - 1);
  for (let code = 65; code <= 90; code += 1) {
    const candidate = `${prefix}${String.fromCharCode(code)}`;
    if (!taken.has(candidate)) {
      return candidate;
    }
  }
  return `${prefix}Z`;
}

function factoriesCollectionRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/factories"),
    resolve: (_match, method, body) => {
      if (method !== "POST") {
        return { json: { factories: fixture.factories } };
      }
      const request = (body ?? {}) as RequestBody;
      const name = stringOrEmpty(request.name) || "New workspace";
      const created = {
        id: `storybook-factory-${fixture.factories.length + 1}`,
        name,
        key: unusedFactoryKey(fixture.factories, name, stringOrEmpty(request.key)),
        description: stringOrEmpty(request.description),
        lines: [],
        onboarding: {},
      };
      fixture.factories.push(created);
      fixture.workOrdersByFactoryId[created.id] = [];
      fixture.appsByFactoryId[created.id] = [];
      return { json: { factory: created } };
    },
  };
}

function factoryWithLineMetrics(factory: FactoriesFactory): FactoriesFactory {
  return {
    ...factory,
    lines: (factory.lines ?? []).map((line) => {
      if (line.metrics) {
        return line;
      }
      const metrics = metricsForLine(line.id);
      if (!metrics) {
        return line;
      }
      return { ...line, metrics };
    }),
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
          if (request.clearHostedSpendBudget === true) {
            factory.hostedSpendBudgetCents = undefined;
          } else if (typeof request.hostedSpendBudgetCents === "number") {
            factory.hostedSpendBudgetCents = String(request.hostedSpendBudgetCents);
          } else if (typeof request.hostedSpendBudgetCents === "string" && request.hostedSpendBudgetCents.trim()) {
            factory.hostedSpendBudgetCents = request.hostedSpendBudgetCents;
          }
          return { json: { factory: factoryWithLineMetrics(factory) } };
        }

        if (method === "DELETE") {
          if (factoryIndex < 0) return { json: {} };
          fixture.factories.splice(factoryIndex, 1);
          delete fixture.workOrdersByFactoryId[factoryId];
          delete fixture.appsByFactoryId[factoryId];
          return { json: {} };
        }

        return factory ? { json: { factory: factoryWithLineMetrics(factory) } } : { json: {} };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/apps"),
      resolve: (match) => ({ json: { apps: fixture.appsByFactoryId[match[1]] ?? [] } }),
    },
    ...factoryIntakeRoutes(fixture),
    {
      pattern: re("/api/v1/factories/([^/]+)/usage"),
      resolve: (match) => ({ json: fixture.usageByFactoryId?.[match[1]] ?? EMPTY_USAGE_REPORT }),
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/velocity"),
      resolve: (match, _method, _body, url) => {
        const byPeriod = fixture.velocityByFactoryId?.[match[1]];
        const periodDays = Number(url.searchParams.get("periodDays") ?? 14);
        const report = byPeriod?.[periodDays] ?? byPeriod?.[14] ?? EMPTY_FACTORY_VELOCITY;

        // The page follows peopleSyncedAt to know a sync finished, so a report
        // read after a sync must carry the newer time.
        const syncedAt = velocitySyncedAt.get(match[1]);
        if (!syncedAt) return { json: report };
        return { json: { ...report, peopleSyncedAt: syncedAt, peopleSyncPending: false } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/velocity/sync"),
      resolve: (match) => {
        velocitySynced(match[1]);
        return { json: { started: true } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/llm-models"),
      resolve: (_match, method, body, url) => {
        const request = (body ?? {}) as { provider?: string; fundingSource?: string; allowedModels?: unknown };
        const provider =
          url.searchParams.get("provider") || (typeof request.provider === "string" ? request.provider : "");
        const models = storybookHostedLlmModels(provider).models;
        if (method === "PUT") {
          const allowed = stringArrayOrEmpty(request.allowedModels);
          return {
            json: {
              selected: allowed.map((id) => ({ id, name: id })),
              inheritParent: allowed.length === 0,
            },
          };
        }
        return { json: { parent: models, selected: models, inheritParent: true } };
      },
    },
  ];
}

function mergedOnboarding(
  current: FactoriesFactoryOnboarding | undefined,
  request: FactoriesUpdateFactoryOnboardingBody,
): FactoriesFactoryOnboarding {
  const next: FactoriesFactoryOnboarding = { ...current };
  if (request.vcsIntegrationId) next.vcsIntegrationId = request.vcsIntegrationId;
  if (request.agentIntegrationId) next.agentIntegrationId = request.agentIntegrationId;
  if (request.appRepository) next.appRepository = request.appRepository;
  if (request.backlogRepository) next.backlogRepository = request.backlogRepository;
  if (request.defaultBranch) next.defaultBranch = request.defaultBranch;
  if (request.issuesSource) next.issuesSource = request.issuesSource;
  if (request.agentHarness) next.agentHarness = request.agentHarness;
  if (request.agentProvider) next.agentProvider = request.agentProvider;
  if (request.agentModel) next.agentModel = request.agentModel;
  if (request.agentPlanningModel) next.agentPlanningModel = request.agentPlanningModel;
  if (request.provisionedAppId) next.provisionedAppId = request.provisionedAppId;
  if (request.provisionedLineId) next.provisionedLineId = request.provisionedLineId;
  if (request.complete) next.completedAt = new Date().toISOString();
  return next;
}

function factoryRepositoryRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/factories/([^/]+)/repository"),
    resolve: (match, method, body) => {
      if (method !== "PATCH") return null;
      const factory = fixture.factories.find((entry) => entry.id === match[1]);
      if (!factory) return { json: {} };
      const request = (body ?? {}) as { repository?: string; defaultBranch?: string };
      factory.onboarding = mergedOnboarding(factory.onboarding, {
        appRepository: request.repository,
        backlogRepository: request.repository,
        defaultBranch: request.defaultBranch,
      });
      return { json: { factory: factoryWithLineMetrics(factory) } };
    },
  };
}

function factoryAgentRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/factories/([^/]+)/agent"),
    resolve: (match, method, body) => {
      if (method !== "PATCH") return null;
      const factory = fixture.factories.find((entry) => entry.id === match[1]);
      if (!factory) return { json: {} };
      const request = (body ?? {}) as FactoriesUpdateFactoryAgentBody;
      const integrationId =
        request.credentialSource === "AGENT_CREDENTIAL_SOURCE_INTEGRATION" ? request.integrationId : "";
      factory.onboarding = mergedOnboarding(factory.onboarding, {
        agentIntegrationId: integrationId,
        agentProvider: request.provider,
        agentHarness:
          request.provider === "AGENT_PROVIDER_OPENAI" ? "AGENT_HARNESS_CODEX" : "AGENT_HARNESS_CLAUDE_CODE",
      });
      return { json: { factory: factoryWithLineMetrics(factory) } };
    },
  };
}

/** Persists workspace setup answers so setup stories advance step by step. */
function factoryOnboardingRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/factories/([^/]+)/onboarding"),
    resolve: (match, method, body) => {
      if (method !== "PATCH") return null;
      const factory = fixture.factories.find((entry) => entry.id === match[1]);
      if (!factory) return { json: {} };
      factory.onboarding = mergedOnboarding(factory.onboarding, (body ?? {}) as FactoriesUpdateFactoryOnboardingBody);
      return { json: { factory: factoryWithLineMetrics(factory) } };
    },
  };
}

function factoryPullRequestRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    {
      pattern: re("/api/v1/factories/([^/]+)/prs"),
      resolve: (match, method, _body, url) => {
        if (method !== "GET") return { json: {} };
        const factoryId = match[1];
        const orderNumber = (url.searchParams.get("order") ?? "").trim();
        const workOrderIds = [
          ...url.searchParams.getAll("workOrderIds"),
          ...url.searchParams.getAll("work_order_ids"),
        ].filter(Boolean);
        const orders = fixture.workOrdersByFactoryId[factoryId] ?? [];
        let pullRequests = orders.flatMap((order) => (order.id ? orderPullRequests(fixture, order.id) : []));
        if (orderNumber) {
          const order = orders.find((entry) => entry.number === orderNumber || entry.id === orderNumber);
          pullRequests = order?.id ? orderPullRequests(fixture, order.id) : [];
        } else if (workOrderIds.length > 0) {
          const allowed = new Set(workOrderIds);
          pullRequests = pullRequests.filter((pr) => pr.workOrderId && allowed.has(pr.workOrderId));
        }
        return { json: { pullRequests } };
      },
    },
  ];
}

function orderPullRequests(fixture: FactoriesFixture, orderId: string): FactoriesFactoryPullRequest[] {
  return fixture.pullRequestsByOrderId?.[orderId] ?? DEFAULT_PULL_REQUESTS_BY_ORDER_ID[orderId] ?? [];
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
    title: stringOrEmpty(request.title) || "New task",
    description: stringOrEmpty(request.description),
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    createdAt: nowIso,
    updatedAt: nowIso,
    createdBy: { user: { id: ORGANIZATION_USERS[0].id, name: ORGANIZATION_USERS[0].name } },
    assignees: findUsersByIds(stringArrayOrEmpty(request.assigneeIds ?? request.assignee_ids).slice(0, 1)),
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
  startStepIndex = 0,
): FactoriesWorkOrderLineDispatch {
  const stepIndex = Math.max(0, startStepIndex);
  const firstStep = line?.steps?.[stepIndex] ?? line?.steps?.[0];
  const firstAppId = firstStep?.app?.app;
  const firstName = automationNameForLineStep(firstStep, apps, stepIndex);
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
        stepIndex,
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

function cancelActiveDispatches(order: FactoriesWorkOrder, now: string) {
  order.lineDispatches = (order.lineDispatches ?? []).map((dispatch) =>
    dispatch.state === "STATE_ACTIVE"
      ? { ...dispatch, state: "STATE_FINISHED", result: "RESULT_CANCELLED", finishedAt: now }
      : dispatch,
  );
}

function dispatchRequestOptions(request: RequestBody) {
  return {
    lineName: stringOrEmpty(request.lineName ?? request.line_name),
    startStepIndex: Number(request.startStepIndex ?? request.start_step_index ?? 0) || 0,
    replaceActive: request.replaceActive === true || request.replace_active === true,
  };
}

function dispatchOrder(fixture: FactoriesFixture, factoryId: string, orderId: string, request: RequestBody) {
  const order = findOrder(fixture, factoryId, orderId);
  if (!order) return { json: {} };
  const factory = fixture.factories.find((entry) => entry.id === factoryId);
  const options = dispatchRequestOptions(request);
  const line = factory?.lines?.find((entry) => entry.name === options.lineName) ?? factory?.lines?.[0];
  const now = new Date().toISOString();
  order.updatedAt = now;
  if (order.state === "STATE_DRAFT") {
    order.state = "STATE_OPEN";
  }

  const apps = fixture.appsByFactoryId[factoryId] ?? [];
  if (options.replaceActive) {
    cancelActiveDispatches(order, now);
  }
  const newDispatch = buildDispatchedLineDispatch(line, options.lineName, now, apps, options.startStepIndex);
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
      resolve: (match, method, body) => {
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        if (method === "PATCH") {
          const request = (body ?? {}) as RequestBody;
          if (typeof request.title === "string") {
            order.title = request.title.trim();
          }
          if (typeof request.description === "string") {
            order.description = request.description;
          }
          order.updatedAt = new Date().toISOString();
        }
        return { json: { order } };
      },
    },
    {
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/assignees"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        const request = (body ?? {}) as RequestBody;
        order.assignees = findUsersByIds(stringArrayOrEmpty(request.assigneeIds ?? request.assignee_ids).slice(0, 1));
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
      pattern: re("/api/v1/factories/([^/]+)/orders/([^/]+)/status"),
      resolve: (match, method, body) => {
        if (method !== "PATCH") return null;
        const order = findOrder(fixture, match[1], match[2]);
        if (!order) return { json: {} };
        const request = (body ?? {}) as RequestBody;
        const state = stringOrEmpty(request.state);
        if (state === "STATE_DRAFT") {
          order.state = "STATE_DRAFT";
          order.result = "RESULT_UNSPECIFIED";
        } else if (state === "STATE_OPEN") {
          order.state = "STATE_OPEN";
        } else if (state === "STATE_CLOSED") {
          order.state = "STATE_CLOSED";
          order.result = stringOrEmpty(request.result) === "RESULT_REJECTED" ? "RESULT_REJECTED" : "RESULT_COMPLETED";
        }
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

function organizationWorkspaceUsageRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/workspace-usage"),
    resolve: () => ({ json: fixture.organizationWorkspaceUsage ?? EMPTY_USAGE_REPORT }),
  };
}

function hostedLlmModelsRoute(): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/hosted-llm-models"),
    resolve: (_match, _method, _body, url) => ({ json: storybookHostedLlmModels(url.searchParams.get("provider")) }),
  };
}

function byokModelsRoute(): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/byok-models"),
    resolve: (_match, method, body, url) => {
      const request = (body ?? {}) as { provider?: string; allowedModels?: unknown };
      const provider =
        url.searchParams.get("provider") || (typeof request.provider === "string" ? request.provider : "");
      const models = storybookHostedLlmModels(provider).models;
      if (method === "PUT") {
        const allowed = stringArrayOrEmpty(request.allowedModels);
        return { json: { selected: allowed.map((id) => ({ id, name: id })) } };
      }
      return {
        json: {
          connected: models.length > 0,
          integrationId: models.length > 0 ? "int-byok" : "",
          selected: models,
          candidates: models,
        },
      };
    },
  };
}

function hostedCreditProductsRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/hosted-credit-products"),
    resolve: () => ({
      json: {
        billingEnabled: Boolean(fixture.hostedCreditProducts?.length),
        products: fixture.hostedCreditProducts ?? [],
      },
    }),
  };
}

function hostedCreditCheckoutRoute(): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/hosted-credit-checkout"),
    resolve: () => ({ json: { checkoutUrl: "https://buy.polar.sh/polar_c_storybook" } }),
  };
}

function billingPortalSessionRoute(): FactoriesRoute {
  return {
    pattern: re("/api/v1/organizations/([^/]+)/billing-portal-session"),
    resolve: () => ({ json: { portalUrl: "https://polar.sh/portal/storybook" } }),
  };
}

const STORYBOOK_ME_MEMBER_PERMISSIONS = ["members"].flatMap((resource) =>
  ["read", "create", "update", "delete"].map((action) => ({ resource, action })),
);

/** Serves `/api/v1/me` so factory stories resolve `useMe` without the Home harness. */
function meRoute(organizationId: string): FactoriesRoute {
  return {
    pattern: re("/api/v1/me"),
    resolve: () => {
      const user = buildStorybookMeUser(organizationId);
      return {
        json: {
          user: {
            ...user,
            id: STORYBOOK_ME_USER_ID,
            name: STORYBOOK_ME_USER_NAME,
            email: STORYBOOK_ME_USER_EMAIL,
            permissions: [...user.permissions, ...STORYBOOK_ME_MEMBER_PERMISSIONS],
          },
        },
      };
    },
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
    factoryOnboardingRoute(fixture),
    factoryRepositoryRoute(fixture),
    factoryAgentRoute(fixture),
    ...factoryLinesRoutes(fixture),
    ...factoryPullRequestRoutes(fixture),
    ...workOrderRoutes(fixture),
    organizationWorkspaceUsageRoute(fixture),
    hostedLlmModelsRoute(),
    byokModelsRoute(),
    hostedCreditProductsRoute(fixture),
    hostedCreditCheckoutRoute(),
    billingPortalSessionRoute(),
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
      return route.resolve(match, method, body, url);
    }
  }
  return null;
}

/** Response body for `GET /api/v1/users` (`useOrganizationUsers`). */
export function factoriesOrganizationUsersResponse(): FixtureResult {
  return {
    json: {
      users: ORGANIZATION_USERS.map(toStorybookOrganizationUser),
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

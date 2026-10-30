import type { FactoriesWorkOrder, FactoriesWorkOrderSummary } from "@/api-client";
import { factoriesDescribeWorkOrder } from "@/api-client";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { isNotFoundError } from "@/pages/app/workflowPageHelpers";
import { factoryWorkOrdersPagePrefix } from "@/pages/factories/lib/workOrderListPagination";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef } from "react";
import { canvasKeys } from "./useCanvasData";
import { factoryQueryKeys } from "./useFactoryData";
import {
  applyWorkOrderToListCaches,
  cachedWorkOrderIdsFromLists,
  pageKeysWithUnknownWorkOrderMembership,
  patchCachedWorkOrderList,
  removeWorkOrderFromListCaches,
} from "./workOrderListCache";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/factories/`;

type FactoryWorkOrderUpdatedPayload = {
  factoryId?: string;
  orderId?: string;
  reason?: string;
};

type FactoryWebsocketMessage = {
  event?: string;
  payload?: FactoryWorkOrderUpdatedPayload;
};

function parseFactoryEvent(event: MessageEvent<unknown>): FactoryWebsocketMessage | null {
  try {
    return JSON.parse(event.data as string) as FactoryWebsocketMessage;
  } catch (error) {
    console.warn("factory ws: failed to parse message", error);
    return null;
  }
}

export function canvasRunsForWorkOrders(orders: FactoriesWorkOrder[]): Array<{ appId: string; runId: string }> {
  const seen = new Set<string>();
  const runs: Array<{ appId: string; runId: string }> = [];
  for (const order of orders) {
    for (const dispatch of order.lineDispatches ?? []) {
      for (const execution of dispatch.stepExecutions ?? []) {
        const appId = execution.run?.appId;
        const runId = execution.run?.id;
        if (!appId || !runId) {
          continue;
        }
        const key = `${appId}:${runId}`;
        if (seen.has(key)) {
          continue;
        }
        seen.add(key);
        runs.push({ appId, runId });
      }
    }
  }
  return runs;
}

type WorkOrderQueryClient = ReturnType<typeof useQueryClient>;

function cachedWorkOrdersForInvalidation(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): FactoriesWorkOrder[] {
  const list =
    queryClient.getQueryData<FactoriesWorkOrder[]>(factoryQueryKeys.workOrders(organizationId, factoryId)) ?? [];
  if (!orderId) {
    return list;
  }

  const detail = queryClient.getQueryData<FactoriesWorkOrder>(
    factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
  );
  return [...list.filter((order) => order.id === orderId), ...(detail && detail.id === orderId ? [detail] : [])];
}

function invalidateOrdersList(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string) {
  // exact: the detail, events, and artifacts keys extend this prefix.
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrders(organizationId, factoryId),
    exact: true,
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.historicalWorkOrders(organizationId, factoryId),
    exact: true,
  });
  void queryClient.invalidateQueries({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  });
}

function invalidateCachedCanvasRuns(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId?: string,
) {
  for (const run of canvasRunsForWorkOrders(
    cachedWorkOrdersForInvalidation(queryClient, organizationId, factoryId, orderId),
  )) {
    void queryClient.invalidateQueries({
      queryKey: canvasKeys.run(run.appId, run.runId),
    });
  }
}

async function describeWorkOrder(
  organizationId: string,
  factoryId: string,
  orderId: string,
): Promise<FactoriesWorkOrder> {
  const response = await factoriesDescribeWorkOrder(
    withOrganizationHeader({
      organizationId,
      path: { factoryId, orderId },
    }),
  );
  if (!response.data?.order) {
    throw new Error("Task not found");
  }
  return response.data.order;
}

function cachedWorkOrderIds(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string): string[] {
  return cachedWorkOrderIdsFromLists(queryClient, organizationId, factoryId);
}

function invalidateTaskActivity(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderIds: string[],
) {
  for (const orderId of orderIds) {
    invalidateCachedCanvasRuns(queryClient, organizationId, factoryId, orderId);
    void queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
    });
    void queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId),
    });
    void queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.planningSession(organizationId, factoryId, orderId),
    });
  }
}

function isMissingWorkOrderError(error: unknown): boolean {
  return isNotFoundError(error) || (error instanceof Error && /404|not found/i.test(error.message));
}

function isExpiredSessionError(error: unknown): boolean {
  return error instanceof Error && error.message === "Unauthorized";
}

function dropMissingWorkOrder(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
) {
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.historicalWorkOrders(organizationId, factoryId),
    exact: true,
  });
  removeWorkOrderFromListCaches(queryClient, organizationId, factoryId, orderId);
  queryClient.removeQueries({
    queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
    exact: true,
  });
  clearOpenRunTaskLabels(queryClient, orderId);
}

async function refreshUpdatedWorkOrder(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
  isCurrent: () => boolean,
) {
  invalidateTaskActivity(queryClient, organizationId, factoryId, [orderId]);
  const order = await describeWorkOrder(organizationId, factoryId, orderId);
  if (!isCurrent()) {
    return;
  }
  const unknownMembershipKeys = pageKeysWithUnknownWorkOrderMembership(
    queryClient,
    organizationId,
    factoryId,
    orderId,
    order,
  );
  queryClient.setQueryData(factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId), order);
  applyWorkOrderToListCaches(queryClient, organizationId, factoryId, orderId, order);
  queryClient.setQueryData<FactoriesWorkOrderSummary[]>(
    factoryQueryKeys.historicalWorkOrders(organizationId, factoryId),
    (orders) => patchCachedWorkOrderList(orders, orderId, order),
  );
  refreshOpenRunTaskLabels(queryClient, order);
  for (const queryKey of unknownMembershipKeys) {
    void queryClient.invalidateQueries({ queryKey });
  }
}

function isIncludeTaskRunQuery(queryKey: readonly unknown[]): boolean {
  return queryKey[0] === "canvases" && queryKey[1] === "runs" && queryKey.includes("includeTask");
}

function refreshOpenRunTaskLabels(queryClient: WorkOrderQueryClient, order: FactoriesWorkOrder) {
  if (!order.id) return;
  const label = { id: order.id, key: order.key || order.id, title: order.title || "Untitled task" };
  const linkedRunIds = runIdsLinkedToOrder(order);
  const queries = queryClient.getQueriesData({
    predicate: (query) => isIncludeTaskRunQuery(query.queryKey),
  });
  for (const [queryKey, data] of queries) {
    const runs = runsInIncludeTaskData(data);
    const targets = runIdsToLabel(runs, linkedRunIds, order.id);
    const next = patchIncludeTaskLabels(data, order.id, label, targets);
    if (next !== data) {
      queryClient.setQueryData(queryKey, next);
    }
    if (gainedRunLink(runs, targets, order.id)) {
      void queryClient.invalidateQueries({ queryKey });
    }
  }
}

function runIdsLinkedToOrder(order: FactoriesWorkOrder): Set<string> {
  const ids = new Set<string>();
  addRunId(ids, order.sourceRunId);
  for (const dispatch of order.lineDispatches ?? []) {
    for (const execution of dispatch.stepExecutions ?? []) {
      addRunId(ids, execution.run?.id);
    }
  }
  for (const pullRequest of order.pullRequests ?? []) {
    for (const activity of pullRequest.activities ?? []) {
      addRunId(ids, activity.run?.id);
    }
    for (const activity of pullRequest.runs ?? []) {
      addRunId(ids, activity.run?.id);
    }
  }
  return ids;
}

function addRunId(ids: Set<string>, id: string | undefined) {
  const trimmed = id?.trim();
  if (trimmed) ids.add(trimmed);
}

function clearOpenRunTaskLabels(queryClient: WorkOrderQueryClient, orderId: string) {
  queryClient.setQueriesData({ predicate: (query) => isIncludeTaskRunQuery(query.queryKey) }, (data) =>
    patchIncludeTaskLabels(data, orderId, undefined),
  );
}

function patchIncludeTaskLabels(
  data: unknown,
  orderId: string,
  label: { id: string; key: string; title: string } | undefined,
  targetRunIds?: ReadonlySet<string>,
): unknown {
  if (!data || typeof data !== "object") return data;
  const targets = label ? (targetRunIds ?? runIdsToLabel(runsInIncludeTaskData(data), new Set(), orderId)) : undefined;
  if ("pages" in data && Array.isArray(data.pages)) {
    let changed = false;
    const pages = data.pages.map((page) => {
      if (!page || typeof page !== "object" || !("runs" in page) || !Array.isArray(page.runs)) return page;
      let pageChanged = false;
      const runs = page.runs.map((run: unknown) => {
        const next = patchRunTask(run, orderId, label, targets);
        if (next !== run) pageChanged = true;
        return next;
      });
      if (!pageChanged) return page;
      changed = true;
      return { ...page, runs };
    });
    return changed ? { ...data, pages } : data;
  }
  if ("run" in data) {
    const next = patchRunTask(data.run, orderId, label, targets);
    return next === data.run ? data : { ...data, run: next };
  }
  return data;
}

function gainedRunLink(runs: unknown[], targets: ReadonlySet<string>, orderId: string): boolean {
  return runs.some((run) => {
    const id = runId(run);
    return Boolean(id && targets.has(id) && taskId(run) !== orderId);
  });
}

function runIdsToLabel(runs: unknown[], linkedRunIds: ReadonlySet<string>, orderId: string): Set<string> {
  const parents = parentIdsByRunId(runs);
  const ids = new Set<string>();
  for (const run of runs) {
    const id = runId(run);
    if (!id) continue;
    if (taskId(run) === orderId || (!taskId(run) && reachesLinkedRun(id, parents, linkedRunIds))) {
      ids.add(id);
    }
  }
  return ids;
}

function reachesLinkedRun(
  start: string,
  parents: Map<string, string | undefined>,
  linkedRunIds: ReadonlySet<string>,
): boolean {
  const seen = new Set<string>();
  let current: string | undefined = start;
  while (current && !seen.has(current)) {
    if (linkedRunIds.has(current)) return true;
    seen.add(current);
    current = parents.get(current);
  }
  return false;
}

function runsInIncludeTaskData(data: unknown): unknown[] {
  if (!data || typeof data !== "object") return [];
  if ("pages" in data && Array.isArray(data.pages)) {
    return data.pages.flatMap((page) =>
      page && typeof page === "object" && "runs" in page && Array.isArray(page.runs) ? page.runs : [],
    );
  }
  if ("run" in data) return [data.run];
  return [];
}

function parentIdsByRunId(runs: unknown[]): Map<string, string | undefined> {
  const parents = new Map<string, string | undefined>();
  for (const run of runs) {
    const id = runId(run);
    if (!id) continue;
    parents.set(id, parentRunId(run));
  }
  return parents;
}

function runId(run: unknown): string | undefined {
  if (!run || typeof run !== "object" || !("id" in run) || typeof run.id !== "string" || !run.id) return undefined;
  return run.id;
}

function parentRunId(run: unknown): string | undefined {
  if (!run || typeof run !== "object" || !("parent" in run)) return undefined;
  return runId(run.parent);
}

function taskId(run: unknown): string | undefined {
  if (!run || typeof run !== "object" || !("task" in run)) return undefined;
  return runId(run.task);
}

function patchRunTask(
  run: unknown,
  orderId: string,
  label: { id: string; key: string; title: string } | undefined,
  targetRunIds: ReadonlySet<string> | undefined,
): unknown {
  if (!run || typeof run !== "object") return run;
  const id = runId(run);
  const currentTaskId = taskId(run);
  if (!label) {
    if (currentTaskId !== orderId || !("task" in run)) return run;
    const { task: _removed, ...rest } = run;
    return rest;
  }
  if (!id || !targetRunIds?.has(id) || (currentTaskId && currentTaskId !== orderId)) return run;
  const task = "task" in run && run.task && typeof run.task === "object" ? run.task : {};
  return { ...run, task: { ...task, id: label.id, key: label.key, title: label.title } };
}

function invalidateFactoryWorkOrdersOnReconnect(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
) {
  invalidateSubscriptionState(queryClient, organizationId, factoryId);
  invalidateOrdersList(queryClient, organizationId, factoryId);
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.planningSessions(organizationId, factoryId),
  });
  invalidateTaskActivity(
    queryClient,
    organizationId,
    factoryId,
    cachedWorkOrderIds(queryClient, organizationId, factoryId),
  );
  void queryClient.invalidateQueries({
    predicate: (query) => isIncludeTaskRunQuery(query.queryKey),
  });
}

export function invalidateFactoryWorkOrderQueries(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): void {
  invalidateModelCatalogs(queryClient, organizationId, factoryId);
  invalidateOrdersList(queryClient, organizationId, factoryId);
  invalidateCachedCanvasRuns(queryClient, organizationId, factoryId, orderId);

  if (!orderId) {
    return;
  }

  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
    exact: true,
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId),
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.planningSession(organizationId, factoryId, orderId),
  });
}

export function useFactoryWebsocket(organizationId: string, factoryId: string, enabled = true): void {
  const queryClient = useQueryClient();
  const hasConnectedOnce = useRef(false);
  const refreshVersion = useRef(new Map<string, number>());

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseFactoryEvent(event);
      if (!data || data.event !== "work_order_updated") {
        return;
      }
      if (!organizationId || !factoryId) {
        return;
      }
      const orderId = data.payload?.orderId;
      if (data.payload?.factoryId && data.payload.factoryId !== factoryId) {
        return;
      }
      if (data.payload?.reason === "codex_connection") {
        invalidateSubscriptionState(queryClient, organizationId, factoryId);
        return;
      }
      if (data.payload?.reason === "runner_capacity") {
        invalidateOrdersList(queryClient, organizationId, factoryId);
        void queryClient.invalidateQueries({
          queryKey: factoryQueryKeys.planningSessions(organizationId, factoryId),
        });
        return;
      }
      if (!orderId) {
        return;
      }
      invalidateModelCatalogs(queryClient, organizationId, factoryId);
      const version = (refreshVersion.current.get(orderId) ?? 0) + 1;
      refreshVersion.current.set(orderId, version);
      const isCurrent = () => refreshVersion.current.get(orderId) === version;
      void refreshUpdatedWorkOrder(queryClient, organizationId, factoryId, orderId, isCurrent).catch((error) => {
        if (isExpiredSessionError(error)) {
          return;
        }
        if (!isMissingWorkOrderError(error)) {
          console.warn("factory ws: failed to refresh work order", error);
          return;
        }
        if (!isCurrent()) {
          return;
        }
        dropMissingWorkOrder(queryClient, organizationId, factoryId, orderId);
      });
    },
    [queryClient, organizationId, factoryId],
  );

  const onOpen = useCallback(() => {
    if (!hasConnectedOnce.current) {
      hasConnectedOnce.current = true;
      return;
    }
    if (!organizationId || !factoryId) {
      return;
    }

    // Catch updates missed while disconnected; WS is the only push channel.
    invalidateFactoryWorkOrdersOnReconnect(queryClient, organizationId, factoryId);
  }, [queryClient, organizationId, factoryId]);

  const url = organizationId && factoryId ? `${SOCKET_SERVER_URL}${factoryId}?organization_id=${organizationId}` : null;

  useWebSocket(
    url,
    {
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      reconnectInterval: 3000,
      heartbeat: false,
      share: false,
      onMessage,
      onOpen,
    },
    enabled && url !== null,
  );
}

function invalidateSubscriptionState(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string) {
  invalidateModelCatalogs(queryClient, organizationId, factoryId);
  void queryClient.invalidateQueries({ queryKey: ["factory-codex-connection", organizationId, factoryId] });
  void queryClient.invalidateQueries({ queryKey: ["me", organizationId] });
}

function invalidateModelCatalogs(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string) {
  void queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "line-runner-models"] });
  void queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "owner-eligibility"] });
  void queryClient.invalidateQueries({
    queryKey: ["organizations", organizationId, "selectable-llm-models", factoryId],
  });
}

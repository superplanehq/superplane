import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
  EventStateRegistry,
  StateFunction,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { GetHttpSyntheticCheckConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export type CheckState = "critical" | "degraded" | "clear";

/**
 * Extracts check state from GET synthetic check API response.
 * Supports common paths: status, state, health, spec.status, lastRun?.status, etc.
 */
export function getCheckStateFromPayload(data: Record<string, any> | undefined): CheckState | null {
  if (!data) return null;
  const raw =
    data.status ??
    data.state ??
    data.health ??
    (data.spec as Record<string, any>)?.status ??
    (data.lastRun as Record<string, any>)?.status ??
    (data.latestResult as Record<string, any>)?.status;
  if (raw == null || typeof raw !== "string") return null;
  const s = raw.toLowerCase();
  if (s === "critical" || s === "2" || s === "fail" || s === "failing") return "critical";
  if (s === "degraded" || s === "1" || s === "warn") return "degraded";
  if (s === "clear" || s === "ok" || s === "0" || s === "pass" || s === "passing" || s === "healthy") return "clear";
  return null;
}

/** Prefer status from Prometheus metrics (payload.data.metrics.status), else from definition. */
function getCheckStateFromPayloadData(payloadData: Record<string, any> | undefined): CheckState | null {
  if (!payloadData) return null;
  const metricsStatus = payloadData.metrics?.status;
  if (metricsStatus != null && typeof metricsStatus === "string") {
    const s = metricsStatus.toLowerCase();
    if (s === "critical") return "critical";
    if (s === "degraded") return "degraded";
    if (s === "clear") return "clear";
  }
  const definition = payloadData.definition ?? payloadData;
  return getCheckStateFromPayload(definition);
}

function checkStateLabel(state: CheckState): string {
  return state === "critical" ? "Critical" : state === "degraded" ? "Degraded" : "Clear";
}

const LOCATION_LABELS: Record<string, string> = {
  "de-frankfurt": "Frankfurt (DE)",
  "us-oregon": "Oregon (US)",
  "us-north-virginia": "North Virginia (US)",
  "uk-london": "London (UK)",
  "be-brussels": "Brussels (BE)",
  "au-melbourne": "Melbourne (AU)",
};

function formatAssertionsSummary(assertions: Record<string, any> | undefined): string {
  if (!assertions) return "";
  const critical = (assertions.criticalAssertions as any[]) ?? [];
  const degraded = (assertions.degradedAssertions as any[]) ?? [];
  const parts: string[] = [];
  for (const a of [...critical, ...degraded]) {
    const kind = a?.kind;
    const spec = a?.spec ?? {};
    const op = spec.operator ?? "=";
    const val = spec.value ?? "";
    const type = spec.type ? ` (${spec.type})` : "";
    if (kind === "status_code") parts.push(`HTTP status ${op} ${val}`);
    else if (kind === "timing") parts.push(`Response time${type} ${op} ${val}`);
    else if (kind) parts.push(`${kind} ${op} ${val}`);
  }
  return parts.length ? parts.join(", ") : "";
}

function formatRetrySummary(retries: Record<string, any> | undefined): string {
  if (!retries?.spec) return "No retries set";
  const attempts = retries.spec.attempts;
  const delay = retries.spec.delay;
  if (attempts != null && attempts > 0 && delay) return `${attempts} attempts, ${delay} delay`;
  return "No retries set";
}

function formatSchedulingSummary(schedule: Record<string, any> | undefined): string {
  if (!schedule) return "";
  const interval = schedule.interval ?? "?";
  const locations = schedule.locations as string[] | undefined;
  const locStr = locations?.length
    ? locations.map((loc: string) => LOCATION_LABELS[loc] ?? loc).join(", ")
    : "—";
  return `Evaluate every ${interval} from ${locStr}`;
}

export const GET_HTTP_SYNTHETIC_CHECK_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  clear: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  degraded: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-yellow-100",
    badgeColor: "bg-yellow-500",
  },
  critical: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const getHttpSyntheticCheckStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }
  if (execution.result === "RESULT_CANCELLED") return "cancelled";
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") return "running";

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const raw = payload?.data as Record<string, any> | undefined;
    const state = getCheckStateFromPayloadData(raw);
    if (state) return state;
    return "clear";
  }

  return "failed";
};

export const GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_HTTP_SYNTHETIC_CHECK_STATE_MAP,
  getState: getHttpSyntheticCheckStateFunction,
};

export const getHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    // API can return event envelope { type, timestamp, data } or the inner { definition, metrics } directly
    const rawData = (payload && typeof payload === "object" && (payload as any).data != null
      ? (payload as any).data
      : payload) as Record<string, any> | undefined;

    if (!rawData || typeof rawData !== "object") {
      return { Response: "No data returned" };
    }

    // Support merged payload { definition, metrics } or legacy single check object
    const definition = (rawData.definition ?? rawData) as Record<string, any>;
    const metrics = rawData.metrics as Record<string, any> | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Retrieved At"] = new Date(payload.timestamp).toLocaleString();
    }

    const meta = definition.metadata as Record<string, any> | undefined;
    const spec = definition.spec as Record<string, any> | undefined;
    const plugin = spec?.plugin as Record<string, any> | undefined;
    const pluginSpec = plugin?.spec as Record<string, any> | undefined;
    const request = pluginSpec?.request as Record<string, any> | undefined;
    const schedule = spec?.schedule as Record<string, any> | undefined;

    const checkId = meta?.labels?.["dash0.com/id"];
    if (checkId) {
      details["Check"] = `https://app.dash0.com/alerting/synthetics/${checkId}`;
    }

    // ——— Configuration (key check definitions) ———
    if (request?.method != null && request?.url != null) {
      details["Target"] = `${(request.method as string).toUpperCase()} ${request.url as string}`;
    }
    const expectedStr = formatAssertionsSummary(pluginSpec?.assertions);
    if (expectedStr) {
      details["Expected"] = expectedStr;
    }
    const retryStr = formatRetrySummary(pluginSpec?.retries);
    if (retryStr) {
      details["Retry"] = retryStr;
    }
    const schedulingStr = formatSchedulingSummary(schedule);
    if (schedulingStr) {
      details["Scheduling"] = schedulingStr;
    }

    // ——— Key metrics ———
    if (metrics && typeof metrics === "object") {
      if (metrics.uptime24hPct != null) {
        details["Uptime (24h)"] = `${Number(metrics.uptime24hPct).toFixed(1)}%`;
      }
      if (metrics.uptime7dPct != null) {
        details["Uptime (7d)"] = `${Number(metrics.uptime7dPct).toFixed(1)}%`;
      }
      if (metrics.avgDuration7dMs != null) {
        const ms = Number(metrics.avgDuration7dMs);
        details["Avg duration (7d)"] = ms >= 1000 ? `${(ms / 1000).toFixed(2)} s` : `${Math.round(ms)} ms`;
      }
      if (metrics.fails7d != null) {
        details["Fails (7d)"] = String(metrics.fails7d);
      }
      if (metrics.lastCheckAt != null) {
        details["Last check"] = String(metrics.lastCheckAt);
      }
      if (metrics.downFor7dSec != null) {
        const sec = Number(metrics.downFor7dSec);
        details["Down for (7d)"] = sec >= 60 ? `${Math.round(sec / 60)} minutes` : `${Math.round(sec)} seconds`;
      }
    } else if (definition && (definition.metadata != null || definition.spec != null)) {
      // Check definition exists but metrics were not returned (e.g. Prometheus had no data, or run was before metrics support)
      details["Key metrics"] =
        "Not available for this run. Re-run the component to load metrics from Dash0 (requires Prometheus data for this check).";
    }

    const displayName = meta?.name ?? plugin?.display?.name;
    if (displayName) {
      details["Name"] = displayName;
    }

    const dataset = meta?.labels?.["dash0.com/dataset"];
    if (dataset) {
      details["Dataset"] = dataset;
    }

    // State: prefer status from Prometheus metrics when present, else from definition
    if (details["State"] == null && metrics?.status != null && String(metrics.status).trim() !== "") {
      details["State"] = String(metrics.status);
    }
    if (details["State"] == null) {
      const checkState = getCheckStateFromPayload(definition);
      details["State"] = checkState ? checkStateLabel(checkState) : "—";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    const timeAgo = formatTimeAgo(new Date(context.execution.createdAt));
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const raw = payload?.data as Record<string, any> | undefined;
    const state = getCheckStateFromPayloadData(raw);
    if (state) {
      return `${checkStateLabel(state)} · ${timeAgo}`;
    }
    return timeAgo;
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetHttpSyntheticCheckConfiguration;

  if (configuration?.checkId) {
    const idPreview =
      configuration.checkId.length > 24 ? configuration.checkId.substring(0, 24) + "…" : configuration.checkId;
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: configuration.dataset });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  const raw = payload?.data as Record<string, any> | undefined;
  const state = getCheckStateFromPayloadData(raw);
  const eventSubtitle = state ? `${checkStateLabel(state)} · ${timeAgo}` : timeAgo;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

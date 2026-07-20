import type { CanvasesCanvasRunRef } from "@/api-client";
import type { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatDuration } from "@/lib/duration";
import { appRunPath } from "@/lib/appPaths";
import type { MetadataItem } from "@/ui/metadataList";
import { getState, getStateMap, getTriggerRenderer } from ".";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  StateFunction,
  SubtitleContext,
} from "./types";

type RunAppMetadata = {
  app?: AppMetadata;
  node?: NodeMetadata;
};

type RunAppConfiguration = {
  app?: string;
  node?: string;
  parameters?: Record<string, unknown>;
  timeout?: number;
};

type NodeMetadata = {
  id?: string;
  name?: string;
};

type AppMetadata = {
  id?: string;
  name?: string;
};

type RunAppExecutionMetadata = {
  run?: RunMetadata;
  timedOutAt?: string;
};

type RunMetadata = {
  id?: string;
  result?: string;
  error?: string;
};

export const runAppStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as RunAppExecutionMetadata;
  if (metadata?.timedOutAt) {
    return "failed";
  }

  const runResult = metadata?.run?.result;
  if (runResult === "failed" || runResult === "cancelled") {
    return "failed";
  }

  if (runResult === "passed") {
    return "success";
  }

  return "success";
};

export const RUN_APP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEFAULT_EVENT_STATE_MAP,
  getState: runAppStateFunction,
};

export const runAppMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "runApp";
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      iconSlug: context.componentDefinition.icon || "play",
      iconColor: "text-gray-800",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Run App",
      metadata: runAppMetadataList(context.node),
      eventSections: lastExecution ? runAppEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const nodeMetadata = context.node.metadata as RunAppMetadata | undefined;
    const appName = nodeMetadata?.app?.name;
    const timestamp = context.execution.updatedAt || context.execution.createdAt;

    if (context.execution.state === "STATE_PENDING" || context.execution.state === "STATE_STARTED") {
      return appName ? `Waiting on ${appName}` : "Waiting on app";
    }

    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const nodeMetadata = context.node.metadata as RunAppMetadata | undefined;
    const metadata = context.execution.metadata as RunAppExecutionMetadata | undefined;
    const childRun = resolveChildRun(context.execution.runs, nodeMetadata);
    const app = nodeMetadata?.app;

    if (app?.name) {
      details["App"] = app.name;
    }

    const organizationId = organizationIdFromPath();
    if (childRun?.canvasId && childRun.id && organizationId) {
      details["Run"] = appRunPath(organizationId, childRun.canvasId, childRun.id);
    }

    if (!metadata) {
      return details;
    }

    if (metadata.run?.result) {
      details["Result"] = metadata.run.result;
    }

    if (metadata.run?.error) {
      details["Error"] = metadata.run.error;
    }

    return details;
  },
};

function resolveChildRun(
  runs: CanvasesCanvasRunRef[] | undefined,
  nodeMetadata: RunAppMetadata | undefined,
): CanvasesCanvasRunRef | undefined {
  if (!runs?.length) {
    return undefined;
  }

  if (!nodeMetadata) {
    return undefined;
  }

  const configuredAppId = nodeMetadata.app?.id;
  if (configuredAppId) {
    const match = runs.find((run) => run.canvasId === configuredAppId);
    if (match) {
      return match;
    }
  }

  return runs[0];
}

function runAppMetadataList(node: NodeInfo): MetadataItem[] {
  const metadataList: MetadataItem[] = [];
  const metadata = node.metadata as RunAppMetadata | undefined;
  const configuration = node.configuration as RunAppConfiguration | undefined;

  if (metadata?.app) {
    metadataList.push({ icon: "layout-grid", label: metadata.app.name });
  }

  if (metadata?.node) {
    metadataList.push({ icon: "workflow", label: metadata.node.name });
  }

  const timeout = configuration?.timeout;
  if (timeout && timeout > 0) {
    metadataList.push({
      icon: "clock",
      label: `Timeout: ${formatDuration(timeout * 1000) || `${timeout}s`}`,
    });
  }

  return metadataList;
}

function runAppEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventID = execution.rootEvent?.id || execution.id || "execution";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : new Date(),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: eventID,
    },
  ];
}

function organizationIdFromPath(): string | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }

  const match = window.location.pathname.match(/^\/([^/]+)\/apps\//);
  return match?.[1];
}

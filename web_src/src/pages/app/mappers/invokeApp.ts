import type { CanvasesCanvasRunRef } from "@/api-client";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { renderTimeAgo } from "@/components/TimeAgo";
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

type InvokeAppMetadata = {
  app?: AppMetadata;
  node?: NodeMetadata;
};

type NodeMetadata = {
  id?: string;
  name?: string;
};

type AppMetadata = {
  id?: string;
  name?: string;
};

export const INVOKE_APP_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
};

export const invokeAppStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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
    return "waiting";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
  }

  return "failed";
};

export const INVOKE_APP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: INVOKE_APP_STATE_MAP,
  getState: invokeAppStateFunction,
};

export const invokeAppMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "invokeApp";
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      iconSlug: context.componentDefinition.icon || "play",
      iconColor: "text-gray-800",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Invoke App",
      metadata: invokeAppMetadataList(context.node),
      eventSections: lastExecution ? invokeAppEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const nodeMetadata = context.node.metadata as InvokeAppMetadata | undefined;
    const appName = nodeMetadata?.app?.name;
    const timestamp = context.execution.updatedAt || context.execution.createdAt;

    if (context.execution.state === "STATE_PENDING" || context.execution.state === "STATE_STARTED") {
      return appName ? `Waiting on ${appName}` : "Waiting on app";
    }

    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const nodeMetadata = context.node.metadata as InvokeAppMetadata | undefined;
    const childRun = resolveChildRun(context.execution.runs, nodeMetadata?.app?.id);
    const app = nodeMetadata?.app;

    if (app?.name) {
      details["App"] = app.name;
    }

    const organizationId = organizationIdFromPath();
    const runUrl =
      childRun?.canvasId && childRun.id && organizationId
        ? `${window.location.origin}${appRunPath(organizationId, childRun.canvasId, childRun.id)}`
        : undefined;
    if (runUrl) {
      details["Run"] = runUrl;
    }

    if (context.execution.result === "RESULT_PASSED") {
      details["Result"] = "passed";
    } else if (context.execution.result === "RESULT_FAILED") {
      details["Result"] = "failed";
    } else if (context.execution.result === "RESULT_CANCELLED") {
      details["Result"] = "cancelled";
    }

    return details;
  },
};

function resolveChildRun(
  runs: CanvasesCanvasRunRef[] | undefined,
  configuredAppId?: string,
): CanvasesCanvasRunRef | undefined {
  if (!runs?.length) {
    return undefined;
  }

  if (configuredAppId) {
    const match = runs.find((run) => run.canvasId === configuredAppId);
    if (match) {
      return match;
    }
  }

  return runs[0];
}

function invokeAppMetadataList(node: NodeInfo): MetadataItem[] {
  const metadataList: MetadataItem[] = [];
  const metadata = node.metadata as InvokeAppMetadata | undefined;
  if (metadata?.app) {
    metadataList.push({ icon: "layout-grid", label: metadata.app.name });
  }

  if (metadata?.node) {
    metadataList.push({ icon: "workflow", label: metadata.node.name });
  }

  return metadataList;
}

function invokeAppEventSections(
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

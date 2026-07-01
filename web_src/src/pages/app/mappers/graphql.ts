import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import type React from "react";
import { getColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import { getTriggerRenderer } from ".";
import { stringOrDash } from "./utils";
import graphqlIcon from "@/assets/icons/graphql.svg";

// Same states as HTTP — GraphQL uses the same success/failure output shape.
const GRAPHQL_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-violet-100",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-black",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-black",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  error: {
    icon: "alert-triangle",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-gray-50",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-black",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-black",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
};

function isGraphQLExecutionError(execution: ExecutionInfo): boolean {
  if (!execution.resultMessage) {
    return false;
  }

  if (execution.resultReason === "RESULT_REASON_ERROR") {
    return true;
  }

  if (execution.result !== "RESULT_FAILED") {
    return false;
  }

  return execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED";
}

const graphqlStateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (isGraphQLExecutionError(execution)) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED") {
    const outputs = execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined;
    if (outputs?.success?.length) {
      return "success";
    } else if (outputs?.failure?.length) {
      return "failed";
    }
  }

  return "failed";
};

export const GRAPHQL_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GRAPHQL_EVENT_STATE_MAP,
  getState: graphqlStateFunction,
};

interface Output {
  status?: number | null;
  body?: unknown;
}

function getGraphQLOutput(
  outputs: { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined,
): Output | undefined {
  const success = outputs?.success;
  if (success && success.length > 0) {
    return success[0]?.data as Output | undefined;
  }

  const failure = outputs?.failure;
  if (failure && failure.length > 0) {
    return failure[0]?.data as Output | undefined;
  }

  return undefined;
}

function getGraphQLResponseStatusString(
  outputs: { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined,
): string | null {
  const status = getGraphQLOutput(outputs)?.status;
  if (status === undefined || status === null) {
    return null;
  }

  return String(status);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function formatGraphQLError(error: unknown): string {
  if (typeof error === "string") {
    return error;
  }

  if (isRecord(error) && typeof error.message === "string") {
    return error.message;
  }

  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
}

function getGraphQLErrorsString(
  outputs: { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined,
): string | null {
  const body = getGraphQLOutput(outputs)?.body;
  if (!isRecord(body) || !Array.isArray(body.errors) || body.errors.length === 0) {
    return null;
  }

  const errors = body.errors.map(formatGraphQLError).filter(Boolean);
  if (errors.length === 0) {
    return null;
  }

  return errors.join("\n");
}

export const graphqlMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      iconSrc: graphqlIcon,
      iconSlug: context.componentDefinition.icon || "network",
      iconColor: getColorClass("black"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: context.lastExecutions[0]
        ? getGraphQLEventSections(context.nodes, context.lastExecutions[0], graphqlStateFunction)
        : undefined,
      includeEmptyState: !context.lastExecutions[0],
      metadata: getGraphQLMetadataList(context.node),
      eventStateMap: GRAPHQL_EVENT_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = context.execution.metadata as { retry?: GraphQLMetadataRetry } | undefined;
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined;

    const responseStatusString = getGraphQLResponseStatusString(outputs) ?? "";
    if (responseStatusString) {
      details["Response"] = responseStatusString;
    }

    const errors = getGraphQLErrorsString(outputs);
    if (errors) {
      details["Errors"] = errors;
    }

    if (metadata?.retry) {
      details["Retries"] = `${metadata.retry.attempts} / ${metadata.retry.maxAttempts}`;
      details["Retry Strategy"] = metadata.retry.strategy;
      details["Next Retry At"] = metadata.retry.nextRetryAt;
      details["Last Status"] = stringOrDash(metadata.retry.lastStatus);
      details["Last Response"] = stringOrDash(metadata.retry.lastResponse);
      details["Last Error"] = stringOrDash(metadata.retry.lastError);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const state = graphqlStateFunction(context.execution);

    if (state === "running") {
      if (context.execution.createdAt) {
        const startTime = new Date(context.execution.createdAt);
        const now = new Date();
        const durationMs = now.getTime() - startTime.getTime();

        if (durationMs < 60000) {
          return `Running for: ${Math.floor(durationMs / 1000)}s`;
        } else {
          const minutes = Math.floor(durationMs / 60000);
          return `Running for: ${minutes}m`;
        }
      }
      return "Running...";
    }

    if (state === "success" || state === "failed") {
      const outputs = context.execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined;
      const responseCode = getGraphQLResponseStatusString(outputs);

      if (responseCode && context.execution.updatedAt) {
        return renderWithTimeAgo(`Response: ${responseCode}`, new Date(context.execution.updatedAt));
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (context.execution.updatedAt) {
        return renderTimeAgo(new Date(context.execution.updatedAt));
      }
    }

    if (context.execution.updatedAt) {
      return renderTimeAgo(new Date(context.execution.updatedAt));
    }

    return "";
  },
};

type GraphQLMetadataRetry = {
  strategy: string;
  interval: number;
  nextRetryAt: string;
  attempts: number;
  maxAttempts: number;
  lastStatus: number | null;
  lastResponse: string | null;
  lastError: string | null;
};

type GraphQLConfiguration = {
  url: string;
};

function getGraphQLMetadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Partial<GraphQLConfiguration> | undefined;
  if (!configuration) {
    return [];
  }
  const metadata: Array<{ icon: string; label: string }> = [];

  if (configuration.url) {
    metadata.push({
      icon: "link",
      label: configuration.url,
    });
  }

  return metadata;
}

function getGraphQLEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  stateFunction: (execution: ExecutionInfo) => EventState,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const generateEventSubtitle = (): string | React.ReactNode => {
    const state = stateFunction(execution);

    if (state === "running") {
      if (execution.createdAt) {
        const startTime = new Date(execution.createdAt);
        const now = new Date();
        const durationMs = now.getTime() - startTime.getTime();

        if (durationMs < 60000) {
          return `Running for: ${Math.floor(durationMs / 1000)}s`;
        } else {
          const minutes = Math.floor(durationMs / 60000);
          return `Running for: ${minutes}m`;
        }
      }
      return "Running...";
    }

    if (state === "success" || state === "failed") {
      const metadata = execution.metadata as Record<string, unknown> | undefined;
      let responseCode: string | null = null;

      if (metadata?.finalStatus !== undefined && metadata.finalStatus !== null) {
        responseCode = (metadata.finalStatus as { toString?: () => string } | null | undefined)?.toString?.() ?? null;
      } else {
        const outputs = execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined;
        responseCode = getGraphQLResponseStatusString(outputs);
      }

      if (responseCode && execution.updatedAt) {
        return renderWithTimeAgo(`Response: ${responseCode}`, new Date(execution.updatedAt));
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (execution.updatedAt) {
        return renderTimeAgo(new Date(execution.updatedAt));
      }
    }

    if (execution.updatedAt) {
      return renderTimeAgo(new Date(execution.updatedAt));
    }

    return "";
  };

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle: generateEventSubtitle(),
    eventState: stateFunction(execution),
    eventId: execution.rootEvent!.id!,
    showAutomaticTime: stateFunction(execution) === "running",
  };

  return [eventSection];
}

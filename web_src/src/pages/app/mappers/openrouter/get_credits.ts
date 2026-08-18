import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import openRouterIcon from "@/assets/icons/integrations/openrouter.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type GetCreditsPayload = {
  totalCredits?: number;
  totalUsage?: number;
  balance?: number;
  key?: {
    label?: string;
    limit?: number | null;
    limitRemaining?: number | null;
    usageMonthly?: number;
  };
};

function credits(value: number): string {
  return `$${value.toFixed(2)}`;
}

export const getCreditsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openrouter";

    return {
      iconSrc: openRouterIcon,
      iconSlug: context.componentDefinition?.icon ?? "wallet",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "Get Credits",
      eventSections: lastExecution ? getCreditsEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as GetCreditsPayload | undefined;

    if (payload?.timestamp) {
      details["Fetched At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (typeof data?.balance === "number") {
      details["Balance"] = credits(data.balance);
    }

    if (typeof data?.totalCredits === "number") {
      details["Total Credits"] = credits(data.totalCredits);
    }

    if (typeof data?.totalUsage === "number") {
      details["Total Usage"] = credits(data.totalUsage);
    }

    if (typeof data?.key?.usageMonthly === "number") {
      details["Key Usage (Month)"] = credits(data.key.usageMonthly);
    }

    // Null rather than absent when the key has no credit limit set.
    if (typeof data?.key?.limitRemaining === "number") {
      details["Key Limit Remaining"] = credits(data.key.limitRemaining);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function getCreditsEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

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

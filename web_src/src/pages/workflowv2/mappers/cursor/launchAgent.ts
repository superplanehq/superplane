import { ComponentBaseProps } from "@/ui/componentBase";
import { getStateMap } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import { formatTimeAgo } from "@/utils/date";
import { baseEventSections } from "./base";

export const launchCloudAgentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cursor";

    return {
      iconSrc: cursorIcon,
      iconSlug: context.componentDefinition?.icon ?? "bot",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Cursor",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0]?.data as any | undefined;

    const status = payload?.status;
    const agent = payload?.agent;

    if (status?.status) {
      details["Status"] = status.status;
    }

    if (status?.id) {
      details["Agent ID"] = status.id;
    } else if (agent?.id) {
      details["Agent ID"] = agent.id;
    }

    const target = status?.target ?? agent?.target;
    if (target?.branchName) {
      details["Branch"] = target.branchName;
    }
    if (target?.prUrl) {
      details["PR URL"] = target.prUrl;
    }
    if (target?.url) {
      details["Agent URL"] = target.url;
    }
    if (status?.summary) {
      details["Summary"] = status.summary;
    } else if (agent?.summary) {
      details["Summary"] = agent.summary;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

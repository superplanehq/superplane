import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./base";

interface CreatePoolConfiguration {
  name?: string;
  enabled?: boolean;
}

export const createPoolMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    const pool = result?.pool as Record<string, any> | undefined;
    if (!pool) return details;

    details["Pool ID"] = pool.id?.toString() || "-";
    details["Name"] = pool.name || "-";

    if (pool.description) {
      details["Description"] = pool.description;
    }

    details["Enabled"] = pool.enabled != null ? String(pool.enabled) : "-";
    details["Minimum Origins"] = pool.minimum_origins != null ? String(pool.minimum_origins) : "-";
    details["Number of Origins"] = Array.isArray(pool.origins) ? String(pool.origins.length) : "-";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreatePoolConfiguration;

  if (configuration?.name) {
    metadata.push({ icon: "network", label: configuration.name });
  }

  if (configuration?.enabled != null) {
    metadata.push({
      icon: configuration.enabled ? "check-circle" : "circle",
      label: configuration.enabled ? "Enabled" : "Disabled",
    });
  }

  return metadata;
}

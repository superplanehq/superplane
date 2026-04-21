import type { ComponentBaseProps } from "@/pages/workflowv2/mappers/types";
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
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { CreateDatabaseConfiguration, DatabaseNodeMetadata } from "./types";
import { baseEventSections } from "./event_helpers";

export const createDatabaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const database = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (!database) return details;

    details["Database Name"] = String(database.name || "-");
    details["Database Cluster"] = String(database.databaseClusterName || database.databaseClusterId || "-");

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DatabaseNodeMetadata | undefined;
  const configuration = node.configuration as CreateDatabaseConfiguration;

  if (configuration?.name) {
    metadata.push({ icon: "database", label: configuration.name });
  } else if (nodeMetadata?.databaseName) {
    metadata.push({ icon: "database", label: nodeMetadata.databaseName });
  }

  if (nodeMetadata?.databaseClusterName) {
    metadata.push({ icon: "server", label: nodeMetadata.databaseClusterName });
  }

  return metadata;
}

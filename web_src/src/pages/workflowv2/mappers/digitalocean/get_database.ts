import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass } from "@/utils/colors";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
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
import type { DatabaseNodeMetadata } from "./types";

export const getDatabaseMapper: ComponentBaseMapper = {
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

    details["Database"] = String(database.name || "-");
    details["Cluster"] = String(database.databaseClusterName || "-");
    details["Engine"] = String(database.engine || "-");
    details["Version"] = String(database.version || "-");
    details["Region"] = String(database.region || "-");
    details["Status"] = String(database.status || "-");

    const connection = database.connection as Record<string, unknown> | undefined;
    if (connection?.host) {
      details["Host"] = String(connection.host);
    }

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

  if (nodeMetadata?.databaseName) {
    metadata.push({ icon: "database", label: nodeMetadata.databaseName });
  }
  if (nodeMetadata?.databaseClusterName) {
    metadata.push({ icon: "server", label: nodeMetadata.databaseClusterName });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}

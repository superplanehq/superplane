import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
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
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { AppNodeMetadata, GetAppConfiguration } from "./types";

export const getAppMapper: ComponentBaseMapper = {
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
    const app = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (!app) return details;

    addAppBasicDetails(details, app);
    addAppDeploymentDetails(details, app);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function addAppBasicDetails(details: Record<string, string>, app: Record<string, unknown>): void {
  details["App ID"] = String(app.id || "-");

  const spec = app.spec as Record<string, unknown> | undefined;
  details["App Name"] = String(spec?.name || "-");

  details["Default Ingress"] = String(app.default_ingress || "-");
  details["Live URL"] = String(app.live_url || "-");

  const region = app.region as Record<string, unknown> | undefined;
  details["Region"] = String(region?.label || region?.slug || "-");
}

function addAppDeploymentDetails(details: Record<string, string>, app: Record<string, unknown>): void {
  const activeDeployment = app.active_deployment as Record<string, unknown> | undefined;
  if (activeDeployment) {
    details["Active Deployment ID"] = String(activeDeployment.id || "-");
    details["Active Deployment Phase"] = String(activeDeployment.phase || "-");
  }

  const inProgressDeployment = app.in_progress_deployment as Record<string, unknown> | undefined;
  if (inProgressDeployment && inProgressDeployment.id) {
    details["In Progress Deployment"] = String(inProgressDeployment.id || "-");
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as AppNodeMetadata | undefined;
  const configuration = node.configuration as GetAppConfiguration;

  if (nodeMetadata?.appName) {
    metadata.push({ icon: "rocket", label: nodeMetadata.appName });
  } else if (configuration?.app) {
    metadata.push({ icon: "info", label: `App ID: ${configuration.app}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent || !execution.createdAt || !execution.rootEvent.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { defaultStateFunction } from "../stateRegistry";
import type { MetadataItem } from "@/ui/metadataList";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { PackageData, PackageNodeMetadata, PromotePackageConfiguration } from "./types";

export const promotePackageEventStateRegistry: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    copied: DEFAULT_EVENT_STATE_MAP.success,
    moved: DEFAULT_EVENT_STATE_MAP.success,
  },
  getState: (execution) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;
    const config = execution.configuration as PromotePackageConfiguration | undefined;
    return config?.mode === "move" ? "moved" : "copied";
  },
};

export const promotePackageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudsmith";

    return {
      iconSrc: cloudsmithIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
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
    const pkg = outputs?.default?.[0]?.data as PackageData | undefined;
    if (!pkg) return details;

    if (pkg.name) details["Package"] = pkg.name;
    if (pkg.version) details["Version"] = pkg.version;
    if (pkg.repository) details["Destination"] = pkg.repository;
    if (pkg.status_str) details["Status"] = pkg.status_str;
    if (pkg.self_webapp_url) details["URL"] = pkg.self_webapp_url;

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const nodeMetadata = node.metadata as PackageNodeMetadata | undefined;
  const configuration = node.configuration as PromotePackageConfiguration | undefined;

  if (nodeMetadata?.repositoryName) {
    items.push({ icon: "package", label: nodeMetadata.repositoryName });
  } else if (configuration?.sourceRepository) {
    items.push({ icon: "package", label: configuration.sourceRepository });
  }

  if (nodeMetadata?.packageName) {
    items.push({ icon: "archive", label: nodeMetadata.packageName });
  } else if (configuration?.package) {
    items.push({ icon: "archive", label: configuration.package });
  }

  if (configuration?.destinationRepository) {
    items.push({ icon: "arrow-right", label: configuration.destinationRepository });
  }

  if (configuration?.mode) {
    items.push({ icon: "copy", label: configuration.mode === "move" ? "Move" : "Copy" });
  }

  return items;
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

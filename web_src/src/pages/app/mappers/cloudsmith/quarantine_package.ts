import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getTriggerRenderer } from "..";
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
import type { MetadataItem } from "@/ui/metadataList";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { defaultStateFunction } from "../stateRegistry";
import type { PackageData, PackageNodeMetadata, QuarantinePackageConfiguration } from "./types";

export const quarantineStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "cloudsmith.package.quarantined": {
    icon: "shield-off",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-orange-500",
    label: "QUARANTINED",
  },
  "cloudsmith.package.released": {
    icon: "shield-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "RELEASED",
  },
};

export const QUARANTINE_PACKAGE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: quarantineStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const event = outputs?.default?.find(
      (o) => o.type === "cloudsmith.package.quarantined" || o.type === "cloudsmith.package.released",
    );
    if (event?.type && quarantineStateMap[event.type]) {
      return event.type;
    }

    return "success";
  },
};

export const quarantinePackageMapper: ComponentBaseMapper = {
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
      eventStateMap: quarantineStateMap,
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

    if (pkg.name) details["Name"] = pkg.name;
    if (pkg.version) details["Version"] = pkg.version;
    if (pkg.format) details["Format"] = pkg.format;
    if (pkg.status_str) details["Status"] = pkg.status_str;

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
  const configuration = node.configuration as QuarantinePackageConfiguration | undefined;

  if (nodeMetadata?.repositoryName) {
    items.push({ icon: "package", label: nodeMetadata.repositoryName });
  } else if (configuration?.repository) {
    items.push({ icon: "package", label: configuration.repository });
  }

  if (nodeMetadata?.packageName) {
    items.push({ icon: "archive", label: nodeMetadata.packageName });
  } else if (configuration?.package) {
    items.push({ icon: "archive", label: configuration.package });
  }

  if (configuration?.action) {
    const icon = configuration.action === "Release" ? "shield-check" : "shield-off";
    items.push({ icon, label: configuration.action });
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

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const event = outputs?.default?.find(
    (o) => o.type === "cloudsmith.package.quarantined" || o.type === "cloudsmith.package.released",
  );
  const eventState = event?.type && quarantineStateMap[event.type] ? event.type : getState(componentName)(execution);

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState,
      eventId: execution.rootEvent.id,
    },
  ];
}

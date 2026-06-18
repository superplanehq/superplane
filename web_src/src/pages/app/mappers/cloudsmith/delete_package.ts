import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getBackgroundColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";
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
import type { PackageData, PackageNodeMetadata, PackageOperationConfiguration, PackageOperationResult } from "./types";

export const deletePackageMapper: ComponentBaseMapper = {
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

    const payload = (context.execution.outputs as { default?: OutputPayload[] } | undefined)?.default?.[0];
    const result = payload?.data as PackageOperationResult | undefined;
    if (!result) return details;

    details["Repository"] = result.repository || "-";
    details["Package"] = result.package || result.data?.slug_perm || "-";
    details["Operation"] = "deleted";

    if (result.data) addPackageDetails(details, result.data);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function addPackageDetails(details: Record<string, string>, pkg: PackageData): void {
  if (pkg.display_name || pkg.name) details["Name"] = pkg.display_name || pkg.name || "";
  if (pkg.version) details["Version"] = pkg.version;
  if (pkg.format) details["Format"] = pkg.format;
  if (pkg.status_str) details["Status"] = pkg.status_str;
  if (pkg.self_webapp_url) details["URL"] = pkg.self_webapp_url;

  const tags = pkg.tags ? Object.keys(pkg.tags).filter((tag) => Boolean(pkg.tags?.[tag])) : [];
  if (tags.length > 0) details["Tags"] = tags.join(", ");
}

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const nodeMetadata = node.metadata as PackageNodeMetadata | undefined;
  const configuration = node.configuration as PackageOperationConfiguration | undefined;

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

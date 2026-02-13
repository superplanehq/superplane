import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface UpdatePackageVersionsStatusConfiguration {
  domain?: string;
  repository?: string;
  package?: string;
  targetStatus?: string;
}

interface UpdatePackageVersionsStatusPayload {
  successfulVersions?: Record<string, { revision?: string; status?: string }>;
  failedVersions?: Record<string, { errorCode?: string; errorMessage?: string }>;
}

export const updatePackageVersionsStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "unknown";

    return {
      title:
        context.node.name ??
        context.componentDefinition.label ??
        context.componentDefinition.name ??
        "Unnamed component",
      iconSrc: awsCodeArtifactIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? updatePackageVersionsStatusEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: updatePackageVersionsStatusMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as UpdatePackageVersionsStatusPayload | undefined;

    if (!data) {
      return {};
    }

    const successful = data.successfulVersions ?? {};
    const failed = data.failedVersions ?? {};
    const successCount = Object.keys(successful).length;
    const failCount = Object.keys(failed).length;

    return {
      "Target status": stringOrDash(Object.values(successful)[0]?.status ?? (successCount > 0 ? "updated" : "-")),
      "Successful versions": String(successCount),
      "Failed versions": String(failCount),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function updatePackageVersionsStatusMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as UpdatePackageVersionsStatusConfiguration | undefined;
  const items: MetadataItem[] = [];
  if (config?.domain) {
    items.push({ icon: "database", label: config.domain });
  }
  if (config?.repository) {
    items.push({ icon: "boxes", label: config.repository });
  }
  if (config?.package) {
    items.push({ icon: "package", label: config.package });
  }
  if (config?.targetStatus) {
    items.push({ icon: "tag", label: config.targetStatus });
  }
  return items;
}

function updatePackageVersionsStatusEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}

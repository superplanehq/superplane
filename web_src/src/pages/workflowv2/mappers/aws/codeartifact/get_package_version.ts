import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import {
  CodeArtifactPackageVersionConfiguration,
  CodeArtifactPackageVersionDescription,
  CodeArtifactPackageLicense,
} from "./types";
import { buildCodeArtifactPackageMetadataItems, formatPackageName } from "./utils";
import { stringOrDash } from "../../utils";

export const getPackageVersionMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _items?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: awsCodeArtifactIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? getPackageVersionEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getPackageVersionMetadataList(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CodeArtifactPackageVersionDescription | undefined;

    if (!result) {
      return {};
    }

    return {
      Package: stringOrDash(formatPackageName(result.namespace, result.packageName)),
      Version: stringOrDash(result.version),
      Format: stringOrDash(result.format),
      Status: stringOrDash(result.status),
      Revision: stringOrDash(result.revision),
      "Display Name": stringOrDash(result.displayName),
      "Published At": result.publishedTime ? formatTimestampInUserTimezone(result.publishedTime) : "-",
      Summary: stringOrDash(result.summary),
      Licenses: formatLicenses(result.licenses),
      "Home Page": stringOrDash(result.homePage),
      "Source Code": stringOrDash(result.sourceCodeRepository),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function getPackageVersionMetadataList(node: ComponentsNode): MetadataItem[] {
  const configuration = node.configuration as CodeArtifactPackageVersionConfiguration | undefined;
  return buildCodeArtifactPackageMetadataItems(configuration);
}

function getPackageVersionEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function formatLicenses(licenses?: CodeArtifactPackageLicense[]): string {
  if (!licenses || licenses.length === 0) {
    return "-";
  }

  const formatted = licenses
    .map((license) => {
      if (!license) {
        return "";
      }

      const name = license.name?.trim();
      const url = license.url?.trim();

      if (name && url) {
        return `${name} (${url})`;
      }

      return name || url || "";
    })
    .filter((entry) => entry.length > 0)
    .join(", ");

  return formatted.length > 0 ? formatted : "-";
}

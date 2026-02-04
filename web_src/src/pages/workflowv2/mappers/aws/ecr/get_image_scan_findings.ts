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
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import { EcrImageScanFindingsResponse, EcrRepositoryConfiguration, EcrRepositoryMetadata } from "./types";
import { getRepositoryLabel } from "./utils";
import { numberOrZero, stringOrDash } from "../../utils";

export const getImageScanFindingsMapper: ComponentBaseMapper = {
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
      iconSrc: awsEcrIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? getScanEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getScanMetadataList(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as EcrImageScanFindingsResponse | undefined;

    if (!result) {
      return {};
    }

    const counts = result.imageScanFindings?.findingSeverityCounts || {};

    return {
      Repository: stringOrDash(result.repositoryName),
      "Image Digest": stringOrDash(result.imageId?.imageDigest),
      "Image Tag": stringOrDash(result.imageId?.imageTag),
      "Scan Status": stringOrDash(result.imageScanStatus?.status),
      "Status Description": stringOrDash(result.imageScanStatus?.description),
      "Scan Completed At": result.imageScanFindings?.imageScanCompletedAt
        ? formatTimestampInUserTimezone(result.imageScanFindings.imageScanCompletedAt)
        : "-",
      "Vulnerability Source Updated At": result.imageScanFindings?.vulnerabilitySourceUpdatedAt
        ? formatTimestampInUserTimezone(result.imageScanFindings.vulnerabilitySourceUpdatedAt)
        : "-",
      "Findings Count": numberOrZero(result.imageScanFindings?.findings?.length).toString(),
      Critical: numberOrZero(counts.CRITICAL).toString(),
      High: numberOrZero(counts.HIGH).toString(),
      Medium: numberOrZero(counts.MEDIUM).toString(),
      Low: numberOrZero(counts.LOW).toString(),
      Undefined: numberOrZero(counts.UNDEFINED).toString(),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function getScanMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as EcrRepositoryMetadata | undefined;
  const configuration = node.configuration as EcrRepositoryConfiguration | undefined;

  const repositoryLabel = getRepositoryLabel(nodeMetadata, configuration);
  if (repositoryLabel) {
    metadata.push({ icon: "package", label: repositoryLabel });
  }

  return metadata;
}

function getScanEventSections(
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

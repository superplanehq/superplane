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
import { EcrImageDetail, EcrRepositoryConfiguration, EcrRepositoryMetadata } from "./types";
import { formatTags, getRepositoryLabel } from "./utils";
import { formatBytes, stringOrDash } from "../../utils";

export const getImageMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution ? getImageEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getImageMetadataList(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as EcrImageDetail | undefined;

    if (!result) {
      return {};
    }

    return {
      Repository: stringOrDash(result.repositoryName),
      "Image Digest": stringOrDash(result.imageDigest),
      "Image Tags": formatTags(result.imageTags),
      "Image Size": formatBytes(result.imageSizeInBytes),
      "Image Pushed At": result.imagePushedAt ? formatTimestampInUserTimezone(result.imagePushedAt) : "-",
      "Manifest Media Type": stringOrDash(result.imageManifestMediaType),
      "Artifact Media Type": stringOrDash(result.artifactMediaType),
      "Registry ID": stringOrDash(result.registryId),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function getImageMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as EcrRepositoryMetadata | undefined;
  const configuration = node.configuration as EcrRepositoryConfiguration | undefined;

  const repositoryLabel = getRepositoryLabel(nodeMetadata, configuration);
  if (repositoryLabel) {
    metadata.push({ icon: "package", label: repositoryLabel });
  }

  return metadata;
}

function getImageEventSections(
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

import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
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
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
      iconSrc: awsEcrIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getImageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getImageMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
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

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getImageMetadataList(node: NodeInfo): MetadataItem[] {
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
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

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

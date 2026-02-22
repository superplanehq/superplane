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
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { stringOrDash } from "../../utils";
import { Ec2Image } from "./types";

interface Configuration {
  region?: string;
  imageId?: string;
}

interface Output {
  image?: Ec2Image;
}

export const getImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getImageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getImageMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;

    if (!output) {
      return {};
    }

    return {
      "Image ID": stringOrDash(output.image?.imageId),
      Name: stringOrDash(output.image?.name),
      Description: stringOrDash(output.image?.description),
      State: stringOrDash(output.image?.state),
      "Creation Date": stringOrDash(output.image?.creationDate),
      "Owner ID": stringOrDash(output.image?.ownerId),
      Architecture: stringOrDash(output.image?.architecture),
      "Image Type": stringOrDash(output.image?.imageType),
      "Root Device Type": stringOrDash(output.image?.rootDeviceType),
      "Root Device Name": stringOrDash(output.image?.rootDeviceName),
      "Virtualization Type": stringOrDash(output.image?.virtualizationType),
      Hypervisor: stringOrDash(output.image?.hypervisor),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getImageMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as Configuration | undefined;

  if (configuration?.region) {
    metadata.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.imageId) {
    metadata.push({ icon: "disc", label: configuration.imageId });
  }

  return metadata;
}

function getImageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id!,
    },
  ];
}

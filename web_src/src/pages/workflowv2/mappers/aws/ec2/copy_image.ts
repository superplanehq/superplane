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
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { Ec2Image } from "./types";

interface Configuration {
  region?: string;
  sourceRegion?: string;
  sourceImageId?: string;
  name?: string;
}

interface Metadata {
  imageId?: string;
  sourceImageId?: string;
  sourceRegion?: string;
  state?: string;
}

interface Output {
  image?: Ec2Image;
}

export const copyImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";
    const configuration = context.node.configuration as Configuration | undefined;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? copyImageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: copyImageMetadata(configuration),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const metadata = context.execution.metadata as Metadata | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;

    if (!output) {
      return {
        "Image ID": stringOrDash(metadata?.imageId),
        Region: stringOrDash(configuration?.region),
        "Source Image ID": stringOrDash(metadata?.sourceImageId),
        "Source Region": stringOrDash(metadata?.sourceRegion),
        State: stringOrDash(metadata?.state),
      };
    }

    return {
      "Source Image ID": stringOrDash(configuration?.sourceImageId),
      "Image ID": stringOrDash(output.image?.imageId),
      Name: stringOrDash(output.image?.name),
      Description: stringOrDash(output.image?.description),
      State: stringOrDash(output.image?.state),
      "Creation Date": stringOrDash(output.image?.creationDate),
      Architecture: stringOrDash(output.image?.architecture),
      "Image Type": stringOrDash(output.image?.imageType),
      "Root Device Type": stringOrDash(output.image?.rootDeviceType),
      "Root Device Name": stringOrDash(output.image?.rootDeviceName),
      "Virtualization Type": stringOrDash(output.image?.virtualizationType),
      Hypervisor: stringOrDash(output.image?.hypervisor),
      "Owner ID": stringOrDash(output.image?.ownerId),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function copyImageMetadata(configuration?: Configuration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.sourceRegion && configuration?.region) {
    items.push({ icon: "globe", label: `${configuration.sourceRegion} → ${configuration.region}` });
  }

  if (configuration?.sourceImageId) {
    items.push({ icon: "disc", label: configuration.sourceImageId });
  }

  if (configuration?.name) {
    items.push({ icon: "tag", label: configuration.name });
  }

  return items;
}

function copyImageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

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
  instanceId?: string;
  name?: string;
  description?: string;
  noReboot?: boolean;
}

interface CreateImageMetadata {
  imageId?: string;
  state?: string;
}

interface Output {
  image?: Ec2Image;
}

export const createImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? createImageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: createImageMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;

    if (!output) {
      const metadata = context.execution.metadata as CreateImageMetadata | undefined;
      return {
        "Image ID": stringOrDash(metadata?.imageId),
        State: stringOrDash(metadata?.state),
      };
    }

    return {
      "Instance ID": stringOrDash(configuration?.instanceId),
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

function createImageMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as Configuration | undefined;

  if (configuration?.region) {
    metadata.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.instanceId) {
    metadata.push({ icon: "server", label: configuration.instanceId });
  }

  if (configuration?.name) {
    metadata.push({ icon: "tag", label: configuration.name });
  }

  if (configuration?.noReboot) {
    metadata.push({ icon: "power", label: "no reboot" });
  }

  return metadata;
}

function createImageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

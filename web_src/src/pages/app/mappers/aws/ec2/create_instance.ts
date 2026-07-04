import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import type { Ec2Instance } from "./types";

interface Configuration {
  name?: string;
  region?: string;
  imageOs?: string;
  image?: string;
  instanceType?: string;
  subnet?: string;
  securityGroup?: string;
  keyName?: string;
  configureRootVolume?: boolean;
  volumeSizeGiB?: number;
  volumeType?: string;
  associatePublicIpAddress?: boolean;
}

interface CreateInstanceNodeMetadata {
  region?: string;
  name?: string;
  imageOs?: string;
  imageOsLabel?: string;
  instanceType?: string;
  imageName?: string;
  subnetName?: string;
  securityGroupName?: string;
}

export const createInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? createInstanceEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: createInstanceMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Ec2Instance | undefined;
    const createdAt = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : undefined;

    if (!output) {
      return {
        "Created At": stringOrDash(createdAt),
        Name: stringOrDash(configuration?.name),
        Region: stringOrDash(configuration?.region),
        State: "-",
        "Instance Type": stringOrDash(configuration?.instanceType),
        "Public IP": "-",
      };
    }

    return {
      "Created At": stringOrDash(createdAt),
      Name: stringOrDash(output.name ?? configuration?.name),
      Region: stringOrDash(output.region ?? configuration?.region),
      State: stringOrDash(output.state),
      "Instance Type": stringOrDash(output.instanceType),
      "Public IP": stringOrDash(output.publicIpAddress),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

const MAX_METADATA_ITEMS = 3;

function metadataItem(icon: MetadataItem["icon"], label?: string): MetadataItem | undefined {
  const value = label?.trim();
  return value ? { icon, label: value } : undefined;
}

function firstDefined(...values: Array<string | undefined>): string | undefined {
  for (const value of values) {
    const trimmed = value?.trim();
    if (trimmed) {
      return trimmed;
    }
  }

  return undefined;
}

function rootVolumeMetadataItem(configuration?: Configuration): MetadataItem | undefined {
  if (!configuration?.configureRootVolume) {
    return undefined;
  }

  const volumeLabel = [
    configuration.volumeSizeGiB ? `${configuration.volumeSizeGiB} GiB` : null,
    configuration.volumeType,
  ]
    .filter(Boolean)
    .join(" ");

  return metadataItem("hard-drive", volumeLabel || undefined);
}

function createInstanceMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as CreateInstanceNodeMetadata | undefined;

  return [
    metadataItem("tag", firstDefined(configuration?.name, nodeMetadata?.name)),
    metadataItem("globe", firstDefined(configuration?.region, nodeMetadata?.region)),
    metadataItem("disc", firstDefined(nodeMetadata?.imageOsLabel, configuration?.imageOs)),
    metadataItem("cpu", firstDefined(configuration?.instanceType, nodeMetadata?.instanceType)),
    metadataItem("layers", nodeMetadata?.imageName),
    metadataItem("network", nodeMetadata?.subnetName),
    rootVolumeMetadataItem(configuration),
  ]
    .filter((item): item is MetadataItem => item !== undefined)
    .slice(0, MAX_METADATA_ITEMS);
}

function createInstanceEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

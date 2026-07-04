import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import { defaultStateFunction } from "../../stateRegistry";
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

interface CreateInstanceFailure {
  error?: string;
  awsErrorCode?: string;
  lastObservedState?: string;
}

interface CreateInstanceOutputs {
  created?: OutputPayload[];
  failed?: OutputPayload[];
  default?: OutputPayload[];
}

export const CREATE_INSTANCE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  created: DEFAULT_EVENT_STATE_MAP.success,
};

// The component emits launch/wait failures to the "failed" output channel while
// the execution itself finishes as passed, so the node state must be derived
// from the emitted channel rather than the execution result.
export const createInstanceStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as CreateInstanceOutputs | undefined;
  if (outputs?.failed && outputs.failed.length > 0) {
    return "failed";
  }

  const state = defaultStateFunction(execution);
  return state === "success" ? "created" : state;
};

export const CREATE_INSTANCE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: CREATE_INSTANCE_STATE_MAP,
  getState: createInstanceStateFunction,
};

export const createInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? createInstanceEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: createInstanceMetadata(context.node),
      eventStateMap: CREATE_INSTANCE_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as CreateInstanceOutputs | undefined;
    const createdAt = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : undefined;

    const failure = outputs?.failed?.[0]?.data as CreateInstanceFailure | undefined;
    if (failure) {
      return failureExecutionDetails(configuration, failure, createdAt);
    }

    const output = (outputs?.created?.[0]?.data ?? outputs?.default?.[0]?.data) as Ec2Instance | undefined;
    return instanceExecutionDetails(configuration, output, createdAt);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function failureExecutionDetails(
  configuration: Configuration | undefined,
  failure: CreateInstanceFailure,
  createdAt?: string,
): Record<string, string> {
  const error = failure.awsErrorCode ? `${failure.awsErrorCode}: ${failure.error ?? ""}`.trim() : failure.error;

  return {
    "Created At": stringOrDash(createdAt),
    Name: stringOrDash(configuration?.name),
    Region: stringOrDash(configuration?.region),
    State: failure.lastObservedState ? `Failed (${failure.lastObservedState})` : "Failed",
    "Instance Type": stringOrDash(configuration?.instanceType),
    Error: stringOrDash(error),
  };
}

function instanceExecutionDetails(
  configuration: Configuration | undefined,
  output: Ec2Instance | undefined,
  createdAt?: string,
): Record<string, string> {
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
}

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

function createInstanceEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: createInstanceStateFunction(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

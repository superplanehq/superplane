import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { defaultStateFunction } from "../../stateRegistry";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { Ec2Instance } from "./types";

interface Configuration {
  region?: string;
  instance?: string;
  operation?: string;
}

interface ManageInstancePowerNodeMetadata {
  region?: string;
  instanceName?: string;
}

type Output = Pick<
  Ec2Instance,
  "instanceId" | "state" | "instanceType" | "publicIpAddress" | "privateIpAddress" | "region"
>;

const operationLabels: Record<string, string> = {
  start: "Start",
  stop: "Stop",
  reboot: "Reboot",
  hibernate: "Hibernate",
};

export const instancePowerStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "aws.ec2.instance.power.started": {
    icon: "play",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "STARTED",
  },
  "aws.ec2.instance.power.stopped": {
    icon: "square",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "STOPPED",
  },
  "aws.ec2.instance.power.rebooted": {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "REBOOTED",
  },
  "aws.ec2.instance.power.hibernated": {
    icon: "moon",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
    label: "HIBERNATED",
  },
};

const powerPayloadPrefix = "aws.ec2.instance.power.";

export const MANAGE_INSTANCE_POWER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: instancePowerStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const powerEvent = outputs?.default?.find((output) => output.type?.startsWith(powerPayloadPrefix));
    if (powerEvent?.type && instancePowerStateMap[powerEvent.type]) {
      return powerEvent.type;
    }

    return "success";
  },
};

export const manageInstancePowerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? manageInstancePowerEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: manageInstancePowerMetadata(context.node),
      eventStateMap: instancePowerStateMap,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;
    const operation = configuration?.operation;
    const completedAt = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : undefined;
    const operationLabel = (operation && operationLabels[operation]) ?? operation ?? "-";

    if (!output) {
      return {
        "Completed At": stringOrDash(completedAt),
        Operation: operationLabel,
        Region: stringOrDash(configuration?.region),
        State: "-",
      };
    }

    const details: Record<string, string> = {
      "Completed At": stringOrDash(completedAt),
      Operation: operationLabel,
      Region: stringOrDash(output.region ?? configuration?.region),
      State: stringOrDash(output.state),
    };

    if (output.publicIpAddress) {
      details["Public IP"] = output.publicIpAddress;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function manageInstancePowerMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as ManageInstancePowerNodeMetadata | undefined;

  const operation = configuration?.operation;
  const region = configuration?.region ?? nodeMetadata?.region;
  const instanceName = nodeMetadata?.instanceName ?? configuration?.instance;

  const metadata: MetadataItem[] = [];

  if (instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }

  if (operation) {
    metadata.push({ icon: "power", label: operationLabels[operation] ?? operation });
  }

  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function manageInstancePowerEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

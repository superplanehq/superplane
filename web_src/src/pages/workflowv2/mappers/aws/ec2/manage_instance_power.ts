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

export const manageInstancePowerMapper: ComponentBaseMapper = {
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
        ? manageInstancePowerEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: manageInstancePowerMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;
    const completedAt = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : undefined;
    const operationLabel = operationLabels[configuration?.operation ?? ""] ?? configuration?.operation ?? "-";

    if (!output) {
      return {
        "Completed At": stringOrDash(completedAt),
        Operation: operationLabel,
        Region: stringOrDash(configuration?.region),
        State: "-",
      };
    }

    return {
      "Completed At": stringOrDash(completedAt),
      Operation: operationLabel,
      Region: stringOrDash(output.region ?? configuration?.region),
      State: stringOrDash(output.state),
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

function manageInstancePowerEventSections(
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

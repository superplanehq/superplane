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
  instanceType?: string;
  securityGroups?: string;
  restartAfterResize?: boolean;
}

interface UpdateInstanceNodeMetadata {
  region?: string;
  instanceId?: string;
  instanceName?: string;
}

type Output = Ec2Instance;

export const updateInstanceMapper: ComponentBaseMapper = {
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
        ? updateInstanceEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: updateInstanceMetadata(context.node),
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

    if (!output) {
      return {
        "Completed At": stringOrDash(completedAt),
        Region: stringOrDash(configuration?.region),
        State: "-",
        "Instance Type": stringOrDash(configuration?.instanceType),
        "Public IP": "-",
      };
    }

    const details: Record<string, string> = {
      "Completed At": stringOrDash(completedAt),
      Region: stringOrDash(output.region ?? configuration?.region),
      State: stringOrDash(output.state),
      "Instance Type": stringOrDash(output.instanceType),
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

function updateInstanceMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as UpdateInstanceNodeMetadata | undefined;

  const metaInstanceId = nodeMetadata?.instanceId;
  const metaInstanceName = nodeMetadata?.instanceName;
  const instanceType = configuration?.instanceType;
  const securityGroup = configuration?.securityGroups;

  const metadata: MetadataItem[] = [];

  const instanceLabel = metaInstanceName || metaInstanceId || configuration?.instance;
  if (instanceLabel) {
    metadata.push({ icon: "server", label: instanceLabel });
  }

  if (metaInstanceId && metaInstanceName && metaInstanceName !== metaInstanceId) {
    metadata.push({ icon: "hash", label: metaInstanceId });
  }

  if (instanceType) {
    metadata.push({ icon: "cpu", label: instanceType });
  }

  if (securityGroup) {
    metadata.push({ icon: "shield", label: securityGroup });
  }

  return metadata;
}

function updateInstanceEventSections(
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

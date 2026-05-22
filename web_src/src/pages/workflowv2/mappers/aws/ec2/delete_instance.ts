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
}

interface DeleteInstanceNodeMetadata {
  region?: string;
  instanceName?: string;
}

type Output = Pick<Ec2Instance, "state">;

export const deleteInstanceMapper: ComponentBaseMapper = {
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
        ? deleteInstanceEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: deleteInstanceMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;
    const deletedAt = formatDeletedAt(context.execution);

    if (!output) {
      return {
        "Deleted At": stringOrDash(deletedAt),
        Region: stringOrDash(configuration?.region),
        State: "-",
      };
    }

    return {
      "Deleted At": stringOrDash(deletedAt),
      Region: stringOrDash(configuration?.region),
      State: stringOrDash(output.state),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function formatDeletedAt(execution: ExecutionInfo): string | undefined {
  const timestamp = execution.updatedAt ?? execution.createdAt;
  if (!timestamp) {
    return undefined;
  }

  return new Date(timestamp).toLocaleString();
}

function deleteInstanceMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as DeleteInstanceNodeMetadata | undefined;
  const metadata: MetadataItem[] = [];

  const instanceName = nodeMetadata?.instanceName ?? configuration?.instance;
  if (instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }

  if (configuration?.region ?? nodeMetadata?.region) {
    metadata.push({ icon: "globe", label: configuration?.region ?? nodeMetadata?.region ?? "" });
  }

  return metadata;
}

function deleteInstanceEventSections(
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

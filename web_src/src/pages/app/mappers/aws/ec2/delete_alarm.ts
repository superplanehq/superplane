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

interface Configuration {
  region?: string;
  alarm?: string;
}

interface DeleteAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
}

interface DeleteAlarmOutput {
  alarmName?: string;
  deleted?: boolean;
  region?: string;
}

export const deleteAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? deleteAlarmEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: deleteAlarmMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const nodeMetadata = context.node.metadata as DeleteAlarmNodeMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as DeleteAlarmOutput | undefined;
    const deletedAt = formatTimestamp(context.execution.updatedAt || context.execution.createdAt);

    if (!output) {
      return {
        "Deleted At": stringOrDash(deletedAt),
        "Alarm Name": stringOrDash(configuration?.alarm ?? nodeMetadata?.alarmName),
        Deleted: "-",
        Region: stringOrDash(configuration?.region ?? nodeMetadata?.region),
      };
    }

    return {
      "Deleted At": stringOrDash(deletedAt),
      "Alarm Name": stringOrDash(output.alarmName ?? configuration?.alarm),
      Deleted: output.deleted === true ? "Yes" : stringOrDash(output.deleted),
      Region: stringOrDash(output.region ?? configuration?.region),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function deleteAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as DeleteAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = nodeMetadata?.alarmName || configuration?.alarm;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  const region = configuration?.region || nodeMetadata?.region;
  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function formatTimestamp(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

function deleteAlarmEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

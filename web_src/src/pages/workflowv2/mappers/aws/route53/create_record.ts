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
import awsRoute53Icon from "@/assets/icons/integrations/aws.route53.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface RecordConfiguration {
  hostedZoneId?: string;
  recordName?: string;
  recordType?: string;
  ttl?: number;
}

interface RecordChangePayload {
  changeId?: string;
  status?: string;
  submittedAt?: string;
  recordName?: string;
  recordType?: string;
}

export const createRecordMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: awsRoute53Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? recordEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: recordMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as RecordChangePayload | undefined;

    if (!data) {
      return {};
    }

    return {
      "Record Name": stringOrDash(data.recordName),
      "Record Type": stringOrDash(data.recordType),
      "Change ID": stringOrDash(data.changeId),
      Status: stringOrDash(data.status),
      "Submitted At": stringOrDash(data.submittedAt),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function recordMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as RecordConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.recordName) {
    items.push({ icon: "globe", label: config.recordName });
  }
  if (config?.recordType) {
    items.push({ icon: "tag", label: config.recordType });
  }

  return items;
}

function recordEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}

import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";
import { formatTimestamp } from "@/lib/utils";

interface SendEmailConfiguration {
  to?: string;
  subject?: string;
  body?: string;
}

interface SendEmailMetadata {
  to?: string[];
  subject?: string;
}

export const sendEmailMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _items?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: sendgridIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? sendEmailEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendEmailMetadataList(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    const failed = outputs?.failed?.[0]?.data as Record<string, unknown> | undefined;

    if (failed) {
      return {
        Error: stringOrDash(failed.error),
        Status: stringOrDash(failed.statusCode),
      };  
    }

    return {
      "Sent At": new Date(execution.updatedAt!).toLocaleString(),
      Status: stringOrDash(result?.status),
      "Message ID": stringOrDash(result?.messageId),
      To: stringOrDash(result?.to),
      Subject: stringOrDash(result?.subject),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function sendEmailMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendEmailMetadata | undefined;
  const configuration = node.configuration as SendEmailConfiguration | undefined;

  const toLabel = "To: " + (nodeMetadata?.to?.join(", ") || configuration?.to);
  if (toLabel) {
    metadata.push({ icon: "mail", label: toLabel });
  }

  if (configuration?.subject) {
    metadata.push({ icon: "message-square", label: "Subject: " + configuration.subject });
  }

  return metadata;
}

function sendEmailEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  if (Array.isArray(value)) {
    return value.join(", ");
  }

  return String(value);
}

import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";

interface SendEmailConfiguration {
  to?: string;
  subject?: string;
  body?: string;
  isHTML?: boolean;
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
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _items?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    return {
      title: node.name!,
      appName: "smtp",
      iconSlug: "smtp",
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? sendEmailEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendEmailMetadataList(node),
      specs: sendEmailSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;

    return {
      "Sent At": formatDate(result?.sentAt) || "-",
      "From Email": stringOrDash(result?.fromEmail),
      To: stringOrDash(result?.to),
      Cc: stringOrDash(result?.cc),
      Subject: stringOrDash(result?.subject),
    };
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function sendEmailMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendEmailMetadata | undefined;
  const configuration = node.configuration as SendEmailConfiguration | undefined;

  // Show recipient(s)
  const toLabel = nodeMetadata?.to?.join(", ") || configuration?.to;
  if (toLabel) {
    metadata.push({ icon: "mail", label: toLabel });
  }

  return metadata;
}

function sendEmailSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendEmailConfiguration | undefined;

  // Show subject
  if (configuration?.subject) {
    specs.push({
      title: "subject",
      tooltipTitle: "subject",
      iconSlug: "message-square",
      value: configuration.subject,
      contentType: "text",
    });
  }

  return specs;
}

function sendEmailEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id,
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

function formatDate(value?: unknown): string | undefined {
  if (!value) return undefined;
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString();
}

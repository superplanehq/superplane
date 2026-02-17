import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import smtpIcon from "@/assets/icons/integrations/smtp.svg";
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
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: smtpIcon,
      iconSlug: "smtp",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? sendEmailEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendEmailMetadataList(context.node),
      specs: sendEmailSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;

    return {
      "Sent At": formatDate(result?.sentAt) || "-",
      "From Email": stringOrDash(result?.fromEmail),
      To: stringOrDash(result?.to),
      Cc: stringOrDash(result?.cc),
      Subject: stringOrDash(result?.subject),
    };
  },
  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendEmailMetadataList(node: NodeInfo): MetadataItem[] {
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

function sendEmailSpecs(node: NodeInfo): ComponentBaseSpec[] {
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

function sendEmailEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

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

function formatDate(value?: unknown): string | undefined {
  if (!value) return undefined;
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString();
}

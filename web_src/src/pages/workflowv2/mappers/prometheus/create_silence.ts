import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { CreateSilenceConfiguration, CreateSilenceNodeMetadata, PrometheusSilencePayload } from "./types";

export const createSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildCreateSilenceProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Created At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const silence = outputs.default[0].data as PrometheusSilencePayload;
    return {
      ...details,
      ...getDetailsForSilence(silence),
    };
  },
};

function buildCreateSilenceProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: { name: string; label: string; color: string },
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

  return {
    iconSrc: prometheusIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name || componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? buildEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: getMetadata(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as CreateSilenceNodeMetadata | undefined;
  const configuration = node.configuration as CreateSilenceConfiguration | undefined;

  if (nodeMetadata?.silenceID) {
    metadata.push({ icon: "bell-off", label: nodeMetadata.silenceID });
  }

  if (configuration?.matchers && configuration.matchers.length > 0) {
    metadata.push({ icon: "filter", label: `${configuration.matchers.length} matcher(s)` });
  }

  return metadata.slice(0, 3);
}

function getDetailsForSilence(silence: PrometheusSilencePayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (silence?.silenceID) {
    details["Silence ID"] = silence.silenceID;
  }

  if (silence?.matchers && silence.matchers.length > 0) {
    details["Matchers"] = silence.matchers
      .map((m) => {
        const operator = m.isEqual === false ? "!=" : "=";
        const suffix = m.isRegex ? " (regex)" : "";
        return `${m.name}${operator}"${m.value}"${suffix}`;
      })
      .join(", ");
  }

  if (silence?.startsAt) {
    details["Starts At"] = new Date(silence.startsAt).toLocaleString();
  }

  if (silence?.endsAt && silence.endsAt !== "0001-01-01T00:00:00Z") {
    details["Ends At"] = new Date(silence.endsAt).toLocaleString();
  }

  if (silence?.createdBy) {
    details["Created By"] = silence.createdBy;
  }

  if (silence?.comment) {
    details["Comment"] = silence.comment;
  }

  return details;
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

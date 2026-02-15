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
import { CreateSilenceConfiguration, PrometheusSilencePayload } from "./types";

export const createSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
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

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateSilenceConfiguration | undefined;

  if (configuration?.duration) {
    metadata.push({ icon: "clock", label: `Duration: ${configuration.duration}` });
  }

  if (configuration?.matchers && configuration.matchers.length > 0) {
    const matcherLabels = configuration.matchers.filter((m) => m.name).map((m) => `${m.name}=${m.value || "*"}`);

    if (matcherLabels.length > 0) {
      metadata.push({
        icon: "funnel",
        label: matcherLabels.length > 2 ? `${matcherLabels.length} matchers` : matcherLabels.join(", "),
      });
    }
  }

  return metadata.slice(0, 3);
}

export function getDetailsForSilence(silence: PrometheusSilencePayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (silence?.silenceID) {
    details["Silence ID"] = silence.silenceID;
  }

  if (silence?.state) {
    details["State"] = silence.state;
  }

  if (silence?.createdBy) {
    details["Created By"] = silence.createdBy;
  }

  if (silence?.comment) {
    details["Comment"] = silence.comment;
  }

  if (silence?.matchers && silence.matchers.length > 0) {
    const matcherStr = silence.matchers
      .map((m) => {
        const op = m.isEqual ? (m.isRegex ? "=~" : "=") : m.isRegex ? "!~" : "!=";
        return `${m.name}${op}"${m.value}"`;
      })
      .join(", ");
    details["Matchers"] = matcherStr;
  }

  if (silence?.startsAt) {
    details["Starts At"] = new Date(silence.startsAt).toLocaleString();
  }

  if (silence?.endsAt) {
    details["Ends At"] = new Date(silence.endsAt).toLocaleString();
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

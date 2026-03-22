import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type { MetadataItem } from "@/ui/metadataList";
import honeycombIcon from "@/assets/icons/integrations/honeycomb.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface CreateEventConfiguration {
  dataset?: string;
  fields?: Record<string, unknown>;
}

type HoneycombCreateEventPayload = {
  status?: string;
  dataset?: string;
  fields?: Record<string, unknown>;
};

export const createEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: honeycombIcon,
      iconSlug: "honeycomb",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? createEventEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: createEventMetadataList(context.node),
      specs: createEventSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const first = outputs?.default?.[0];
    const data = first?.data as HoneycombCreateEventPayload | undefined;

    return {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Status: data?.status ?? "-",
      Dataset: data?.dataset ?? "-",
      "Sent Fields": formatFieldsForDisplay(data?.fields),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function createEventMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateEventConfiguration | undefined;

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: configuration.dataset });
  } else {
    metadata.push({ icon: "database", label: "Uses integration dataset" });
  }

  return metadata;
}

function createEventSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as CreateEventConfiguration | undefined;

  if (configuration?.fields) {
    specs.push({
      title: "fields",
      tooltipTitle: "fields",
      iconSlug: "braces",
      value: formatFieldsForDisplay(configuration.fields),
      contentType: "json",
    });
  }

  return specs;
}

function createEventEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function safeJSONStringify(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function formatFieldsForDisplay(fields: Record<string, unknown> | undefined): string {
  if (fields == null) return "-";
  return safeJSONStringify(fields);
}

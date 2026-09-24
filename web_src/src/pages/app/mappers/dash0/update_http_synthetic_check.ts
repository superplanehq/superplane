import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import type { HttpSyntheticCheckPayload, UpdateHttpSyntheticCheckConfiguration } from "./types";
import { truncate } from "../safeMappers";
import { renderTimeAgo } from "@/components/TimeAgo";

const LOCATION_LABELS: Record<string, string> = {
  "de-frankfurt": "Frankfurt",
  "us-oregon": "Oregon",
  "us-north-virginia": "N. Virginia",
  "uk-london": "London",
  "be-brussels": "Brussels",
  "au-melbourne": "Melbourne",
};

export const updateHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as HttpSyntheticCheckPayload | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Updated At"] = new Date(payload.timestamp).toLocaleString();
    }

    const checkId = responseData.metadata?.labels?.["dash0.com/id"];
    if (checkId) {
      details["Check"] = `https://app.dash0.com/alerting/synthetics/${checkId}`;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function requestMetadataItems(request: UpdateHttpSyntheticCheckConfiguration["request"] | undefined): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (request?.url) {
    items.push({ icon: "globe", label: truncate(request.url, 50) });
  }

  if (request?.method) {
    items.push({ icon: "arrow-right", label: request.method.toUpperCase() });
  }

  return items;
}

function scheduleMetadataItems(
  schedule: UpdateHttpSyntheticCheckConfiguration["schedule"] | undefined,
): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (schedule?.locations && schedule.locations.length > 0) {
    const locationNames = schedule.locations.map((loc) => LOCATION_LABELS[loc] || loc).join(", ");
    items.push({ icon: "map-pin", label: locationNames });
  }

  if (schedule?.interval) {
    items.push({ icon: "clock", label: `Every ${schedule.interval}` });
  }

  return items;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateHttpSyntheticCheckConfiguration;

  if (configuration?.checkId) {
    const idPreview = truncate(configuration.checkId, 20, "…");
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  metadata.push(...requestMetadataItems(configuration?.request));
  metadata.push(...scheduleMetadataItems(configuration?.schedule));

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
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

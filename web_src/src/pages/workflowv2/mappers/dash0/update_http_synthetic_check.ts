import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { UpdateHttpSyntheticCheckConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

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
    const responseData = payload?.data as Record<string, any> | undefined;

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

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateHttpSyntheticCheckConfiguration;

  if (configuration?.checkId) {
    const idPreview =
      configuration.checkId.length > 20 ? configuration.checkId.substring(0, 20) + "…" : configuration.checkId;
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  if (configuration?.request?.url) {
    const urlPreview =
      configuration.request.url.length > 50
        ? configuration.request.url.substring(0, 50) + "..."
        : configuration.request.url;
    metadata.push({ icon: "globe", label: urlPreview });
  }

  if (configuration?.request?.method) {
    metadata.push({ icon: "arrow-right", label: configuration.request.method.toUpperCase() });
  }

  if (configuration?.schedule?.locations && configuration.schedule.locations.length > 0) {
    const locationNames = configuration.schedule.locations.map((loc) => LOCATION_LABELS[loc] || loc).join(", ");
    metadata.push({ icon: "map-pin", label: locationNames });
  }

  if (configuration?.schedule?.interval) {
    metadata.push({ icon: "clock", label: `Every ${configuration.schedule.interval}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

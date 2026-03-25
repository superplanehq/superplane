import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { GetObjectConfiguration, GetObjectOutput } from "./types";
import { defaultStateFunction } from "../stateRegistry";

type GetObjectOutputs = {
  found?: OutputPayload[];
  notFound?: OutputPayload[];
  not_found?: OutputPayload[];
};

const GET_OBJECT_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  found: {
    ...DEFAULT_EVENT_STATE_MAP.success,
    label: "Found",
  },
  notFound: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "Not Found",
  },
};

export const GET_OBJECT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_OBJECT_STATE_MAP,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") {
      return state;
    }

    const outputs = execution.outputs as GetObjectOutputs | undefined;
    if (outputs?.notFound?.length || outputs?.not_found?.length) {
      return "notFound";
    }
    if (outputs?.found?.length) {
      return "found";
    }

    return "success";
  },
};

export const getObjectMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as
      | {
          found?: { data: GetObjectOutput }[];
          notFound?: { data: GetObjectOutput }[];
          not_found?: { data: GetObjectOutput }[];
        }
      | undefined;

    const result = outputs?.found?.[0]?.data;
    const notFound = outputs?.notFound?.[0]?.data ?? outputs?.not_found?.[0]?.data;

    if (notFound) {
      details["File Path"] = notFound.filePath || "-";
      details["Result"] = "Not Found";
      return details;
    }

    if (!result) return details;

    details["File Path"] = result.filePath || "-";
    details["Origin Endpoint"] = result.endpoint || "-";
    details["Content Type"] = result.contentType || "-";
    details["Size"] = result.size || "-";
    details["Last Modified"] = result.lastModified || "-";

    if (result.tags && Object.keys(result.tags).length > 0) {
      details["Tags"] = Object.entries(result.tags)
        .map(([k, v]) => `${k}=${v}`)
        .join(", ");
    }

    if (result.metadata && Object.keys(result.metadata).length > 0) {
      details["Metadata"] = Object.entries(result.metadata)
        .map(([k, v]) => `${k}=${v}`)
        .join(", ");
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetObjectConfiguration;

  if (configuration?.bucket) {
    metadata.push({ icon: "database", label: configuration.bucket });
  }

  if (configuration?.filePath) {
    metadata.push({ icon: "file", label: configuration.filePath });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id!,
    },
  ];
}

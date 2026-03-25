import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addFormattedTimestamp, addOrderedDetails } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface DeleteAlertConfiguration {
  alertId?: string;
  project?: string;
}

interface AlertRuleNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
  alertName?: string;
}

interface DeleteAlertOutput {
  id?: string;
  name?: string;
  deleted?: boolean;
}

export const deleteAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as DeleteAlertOutput | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [response?.name, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as DeleteAlertOutput | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Started At", context.execution.createdAt);
    addOrderedDetails(details, [
      { label: "Name", value: response?.name },
      { label: "Deleted", value: response?.deleted ? "Yes" : undefined },
    ]);

    return details;
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  const rootComponentName = rootTriggerNode?.componentName;

  if (!rootEvent || !createdAt || !rootComponentName) {
    return undefined;
  }

  const rootTriggerRenderer = getTriggerRenderer(rootComponentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id || "",
    },
  ];
}

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as DeleteAlertConfiguration | undefined;
  const nodeMetadata = node.metadata as AlertRuleNodeMetadata | undefined;
  const metadata = [];

  const alertLabel = nodeMetadata?.alertName || configuration?.alertId;
  if (alertLabel) {
    metadata.push({ icon: "siren", label: alertLabel });
  }

  const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  return metadata.slice(0, 3);
}

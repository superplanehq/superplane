import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, DEFAULT_EVENT_STATE_MAP, EventSection, EventStateMap } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import renderIcon from "@/assets/icons/integrations/render.svg";
import { formatTimestamp, stringOrDash } from "./common";
import { defaultStateFunction } from "../stateRegistry";

interface DeployConfiguration {
  service?: string;
  clearCache?: boolean;
}

interface DeployOutput {
  deployId?: string;
  status?: string;
  createdAt?: string;
  finishedAt?: string;
  rollbackToDeployId?: string;
}

export const DEPLOY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  rollback: {
    icon: "rotate-ccw",
    textColor: "text-amber-800",
    backgroundColor: "bg-amber-100",
    badgeColor: "bg-amber-500",
  },
};

export const deployStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[]; success?: OutputPayload[] } | undefined;
  if (outputs?.failed?.length) {
    const failedOutput = outputs.failed[0]?.data as DeployOutput | undefined;
    const failedStatus = failedOutput?.status?.toLowerCase();
    if (failedStatus === "cancelled" || failedStatus === "canceled") {
      return "cancelled";
    }

    return "failed";
  }

  if (outputs?.success?.length) {
    const successOutput = outputs.success[0]?.data as DeployOutput | undefined;
    const successStatus = successOutput?.status?.toLowerCase();

    if (successStatus === "cancelled" || successStatus === "canceled") {
      return "cancelled";
    }

    if (successOutput?.rollbackToDeployId) {
      return "rollback";
    }
  }

  return defaultStateFunction(execution);
};

export const DEPLOY_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEPLOY_STATE_MAP,
  getState: deployStateFunction,
};

export const deployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: renderIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? deployEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: deployMetadataList(context.node),
      eventStateMap: DEPLOY_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result =
      (outputs?.success?.[0]?.data as DeployOutput | undefined) ??
      (outputs?.failed?.[0]?.data as DeployOutput | undefined);

    return {
      "Triggered At": formatTimestamp(result?.createdAt, context.execution.createdAt),
      "Deploy ID": stringOrDash(result?.deployId),
      Status: stringOrDash(result?.status),
      "Finished At": formatTimestamp(result?.finishedAt),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function deployMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeployConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }

  if (configuration?.clearCache) {
    metadata.push({ icon: "trash-2", label: "Clear cache" });
  }

  return metadata;
}

function deployEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
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

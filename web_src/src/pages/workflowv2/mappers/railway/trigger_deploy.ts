import type {
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
import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getTriggerRenderer } from "..";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import railwayIcon from "@/assets/icons/integrations/railway.svg";
import { defaultStateFunction } from "../stateRegistry";

interface TriggerDeployConfiguration {
  project?: string;
  service?: string;
  environment?: string;
}

interface TriggerDeployOutput {
  deployId?: string;
  status?: string;
  projectId?: string;
  serviceId?: string;
  environmentId?: string;
}

export const DEPLOY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  crashed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  removed: {
    icon: "trash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  skipped: {
    icon: "circle-slash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

export const deployStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[]; success?: OutputPayload[] } | undefined;
  if (outputs?.failed?.length) {
    const failedOutput = outputs.failed[0]?.data as TriggerDeployOutput | undefined;
    const status = failedOutput?.status?.toLowerCase();
    if (status === "crashed") return "crashed";
    if (status === "removed") return "removed";
    if (status === "skipped") return "skipped";
    return "failed";
  }

  if (outputs?.success?.length) {
    return "success";
  }

  return defaultStateFunction(execution);
};

export const DEPLOY_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEPLOY_STATE_MAP,
  getState: deployStateFunction,
};

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return String(value);
}

export const triggerDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: railwayIcon,
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
      (outputs?.success?.[0]?.data as TriggerDeployOutput | undefined) ??
      (outputs?.failed?.[0]?.data as TriggerDeployOutput | undefined);

    return {
      "Deploy ID": stringOrDash(result?.deployId),
      Status: stringOrDash(result?.status),
      "Project ID": stringOrDash(result?.projectId),
      "Service ID": stringOrDash(result?.serviceId),
      "Environment ID": stringOrDash(result?.environmentId),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function deployMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as TriggerDeployConfiguration | undefined;

  if (configuration?.project) {
    metadata.push({ icon: "folder", label: `Project: ${configuration.project}` });
  }

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }

  if (configuration?.environment) {
    metadata.push({ icon: "globe", label: `Environment: ${configuration.environment}` });
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
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

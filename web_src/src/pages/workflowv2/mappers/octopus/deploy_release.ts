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
import octopusIcon from "@/assets/icons/integrations/octopus.svg";
import { formatTimestamp, stringOrDash } from "./common";
import { defaultStateFunction } from "../stateRegistry";

interface DeployReleaseConfiguration {
  project?: string;
  release?: string;
  environment?: string;
}

interface OctopusNodeMetadata {
  projectName?: string;
  releaseName?: string;
  environmentName?: string;
}

interface DeployReleaseOutput {
  deploymentId?: string;
  taskState?: string;
  projectId?: string;
  releaseId?: string;
  environmentId?: string;
  created?: string;
  completedTime?: string;
  duration?: string;
  errorMessage?: string;
}

export const DEPLOY_RELEASE_STATE_MAP: EventStateMap = {
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
};

export const deployReleaseStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[]; success?: OutputPayload[] } | undefined;
  if (outputs?.failed?.length) {
    const failedOutput = outputs.failed[0]?.data as DeployReleaseOutput | undefined;
    const taskState = failedOutput?.taskState;
    if (taskState === "Canceled" || taskState === "Cancelling") {
      return "cancelled";
    }
    return "failed";
  }

  return defaultStateFunction(execution);
};

export const DEPLOY_RELEASE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEPLOY_RELEASE_STATE_MAP,
  getState: deployReleaseStateFunction,
};

export const deployReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: octopusIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? deployReleaseEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: deployReleaseMetadataList(context.node),
      eventStateMap: DEPLOY_RELEASE_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result =
      (outputs?.success?.[0]?.data as DeployReleaseOutput | undefined) ??
      (outputs?.failed?.[0]?.data as DeployReleaseOutput | undefined);

    const nodeMetadata = context.node.metadata as OctopusNodeMetadata | undefined;

    return {
      "Deployment ID": stringOrDash(result?.deploymentId),
      "Task State": stringOrDash(result?.taskState),
      Project: stringOrDash(nodeMetadata?.projectName || result?.projectId),
      Environment: stringOrDash(nodeMetadata?.environmentName || result?.environmentId),
      Release: stringOrDash(nodeMetadata?.releaseName || result?.releaseId),
      Created: formatTimestamp(result?.created, context.execution.createdAt),
      "Completed At": formatTimestamp(result?.completedTime),
      Duration: stringOrDash(result?.duration),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function deployReleaseMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeployReleaseConfiguration | undefined;
  const nodeMetadata = node.metadata as OctopusNodeMetadata | undefined;

  if (configuration?.project) {
    metadata.push({ icon: "folder", label: `Project: ${nodeMetadata?.projectName || configuration.project}` });
  }

  if (configuration?.release) {
    metadata.push({ icon: "tag", label: `Release: ${nodeMetadata?.releaseName || configuration.release}` });
  }

  if (configuration?.environment) {
    metadata.push({
      icon: "globe",
      label: `Environment: ${nodeMetadata?.environmentName || configuration.environment}`,
    });
  }

  return metadata;
}

function deployReleaseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
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

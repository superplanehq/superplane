import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  StateFunction,
  EventStateRegistry,
} from "../types";
import RailwayLogo from "@/assets/icons/integrations/railway.svg";
import { formatTimeAgo } from "@/utils/date";
import { DEFAULT_EVENT_STATE_MAP, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { getTriggerRenderer } from "..";

interface TriggerDeployMetadata {
  project?: { id?: string; name?: string };
  service?: { id?: string; name?: string };
  environment?: { id?: string; name?: string };
}

interface TriggerDeployExecutionMetadata {
  deploymentId?: string;
  status?: string;
  url?: string;
}

// Railway deployment status constants (matching backend)
const DeploymentStatus = {
  QUEUED: "QUEUED",
  WAITING: "WAITING",
  BUILDING: "BUILDING",
  DEPLOYING: "DEPLOYING",
  SUCCESS: "SUCCESS",
  FAILED: "FAILED",
  CRASHED: "CRASHED",
  REMOVED: "REMOVED",
  SLEEPING: "SLEEPING",
  SKIPPED: "SKIPPED",
} as const;

/**
 * Custom state map for Railway deployment statuses
 */
export const TRIGGER_DEPLOY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  queued: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  building: {
    icon: "hammer",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  deploying: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-purple-100",
    badgeColor: "bg-purple-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  crashed: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-orange-500",
  },
};

/**
 * Maps Railway deployment status to UI event state
 */
function mapDeploymentStatusToState(status: string | undefined): EventState {
  switch (status) {
    case DeploymentStatus.QUEUED:
    case DeploymentStatus.WAITING:
      return "queued";
    case DeploymentStatus.BUILDING:
      return "building";
    case DeploymentStatus.DEPLOYING:
      return "deploying";
    case DeploymentStatus.SUCCESS:
    case DeploymentStatus.SLEEPING:
      // SLEEPING means deployment succeeded but app went to sleep due to inactivity
      return "passed";
    case DeploymentStatus.CRASHED:
      return "crashed";
    case DeploymentStatus.FAILED:
    case DeploymentStatus.REMOVED:
    case DeploymentStatus.SKIPPED:
      return "failed";
    default:
      return "neutral";
  }
}

/**
 * State function for Railway TriggerDeploy component
 */
export const triggerDeployStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  // Check for errors first
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // If execution is finished, map based on final status
  if (execution.state === "STATE_FINISHED") {
    const metadata = execution.metadata as TriggerDeployExecutionMetadata;
    if (metadata?.status) {
      return mapDeploymentStatusToState(metadata.status);
    }
    // Fallback based on result
    return execution.result === "RESULT_PASSED" ? "passed" : "failed";
  }

  // If still running, show the current deployment status from metadata
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_PENDING") {
    const metadata = execution.metadata as TriggerDeployExecutionMetadata;
    if (metadata?.status) {
      return mapDeploymentStatusToState(metadata.status);
    }
    return "queued"; // Default to queued if no status yet
  }

  return "neutral";
};

/**
 * State registry for Railway TriggerDeploy component
 */
export const TRIGGER_DEPLOY_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TRIGGER_DEPLOY_STATE_MAP,
  getState: triggerDeployStateFunction,
};

/**
 * Formats the deployment status for display
 */
function formatDeploymentStatus(status: string | undefined): string {
  if (!status) return "";
  // Convert SCREAMING_CASE to Title Case
  return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
}

/**
 * Builds the subtitle string for a Railway deployment execution
 */
function getDeploymentSubtitle(execution: ExecutionInfo): string {
  const execMetadata = execution.metadata as TriggerDeployExecutionMetadata;
  const status = execMetadata?.status;

  // Show current deployment status while running
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_PENDING") {
    switch (status) {
      case DeploymentStatus.QUEUED:
      case DeploymentStatus.WAITING:
        return "Queued...";
      case DeploymentStatus.BUILDING:
        return "Building...";
      case DeploymentStatus.DEPLOYING:
        return "Deploying...";
      default:
        return "Starting...";
    }
  }

  // Finished states
  if (execution.state === "STATE_FINISHED") {
    const updatedAt = execution.updatedAt ? new Date(execution.updatedAt) : null;
    const timeAgo = updatedAt ? formatTimeAgo(updatedAt) : "";

    switch (status) {
      case DeploymentStatus.SUCCESS:
      case DeploymentStatus.SLEEPING:
        // SLEEPING means deployment succeeded but app went to sleep
        return timeAgo ? `Deployed ${timeAgo}` : "Deployed";
      case DeploymentStatus.CRASHED:
        return timeAgo ? `Crashed ${timeAgo}` : "Crashed";
      case DeploymentStatus.FAILED:
      case DeploymentStatus.REMOVED:
      case DeploymentStatus.SKIPPED:
        return timeAgo ? `Failed ${timeAgo}` : "Failed";
      default:
        // Fallback to result-based status
        if (execution.result === "RESULT_PASSED") {
          return timeAgo ? `Deployed ${timeAgo}` : "Deployed";
        }
        return timeAgo ? `Failed ${timeAgo}` : "Failed";
    }
  }

  const createdAt = execution.createdAt ? new Date(execution.createdAt) : null;
  return createdAt ? formatTimeAgo(createdAt) : "";
}

/**
 * Builds event sections for a Railway TriggerDeploy execution
 */
function getTriggerDeployEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });
  const isRunning = execution.state === "STATE_STARTED" || execution.state === "STATE_PENDING";

  return [
    {
      showAutomaticTime: isRunning,
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: getDeploymentSubtitle(execution),
      eventState: triggerDeployStateFunction(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

/**
 * Mapper for the "railway.triggerDeploy" component type
 */
export const triggerDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const { node, nodes, componentDefinition, lastExecutions } = context;
    const metadata = node.metadata as unknown as TriggerDeployMetadata;
    const metadataItems = [];

    // Show service name
    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "box",
        label: metadata.service.name,
      });
    }

    // Show environment name
    if (metadata?.environment?.name) {
      metadataItems.push({
        icon: "globe",
        label: metadata.environment.name,
      });
    }

    return {
      iconSrc: RailwayLogo,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      title: node.name || componentDefinition.label || "Trigger Deploy",
      metadata: metadataItems,
      eventStateMap: TRIGGER_DEPLOY_STATE_MAP,
      eventSections: lastExecutions[0] ? getTriggerDeployEventSections(nodes, lastExecutions[0]) : undefined,
      includeEmptyState: !lastExecutions[0],
    };
  },

  subtitle(context: SubtitleContext): string {
    return getDeploymentSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext) {
    const { execution, node } = context;
    const details: Record<string, string> = {};

    if (execution.createdAt) {
      details["Started At"] = new Date(execution.createdAt).toLocaleString();
    }

    if (execution.updatedAt && execution.state === "STATE_FINISHED") {
      details["Finished At"] = new Date(execution.updatedAt).toLocaleString();
    }

    // Add execution metadata info (deployment status, ID, URL)
    const execMetadata = execution.metadata as TriggerDeployExecutionMetadata;
    if (execMetadata?.status) {
      details["Deployment Status"] = formatDeploymentStatus(execMetadata.status);
    }
    if (execMetadata?.deploymentId) {
      details["Deployment ID"] = execMetadata.deploymentId;
    }
    if (execMetadata?.url) {
      details["Deployment URL"] = execMetadata.url;
    }

    // Add node metadata info (project, service, environment)
    const nodeMetadata = node.metadata as unknown as TriggerDeployMetadata;
    if (nodeMetadata?.project?.name) {
      details["Project"] = nodeMetadata.project.name;
    }
    if (nodeMetadata?.service?.name) {
      details["Service"] = nodeMetadata.service.name;
    }
    if (nodeMetadata?.environment?.name) {
      details["Environment"] = nodeMetadata.environment.name;
    }

    return details;
  },
};

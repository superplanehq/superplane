import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  StateFunction,
  SubtitleContext,
} from "../types";
import type {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { buildGithubExecutionSubtitle } from "./utils";

interface ExecutionMetadata {
  workflowRun?: {
    id: number;
    status: string;
    conclusion: string;
    url: string;
  };
}

export const RUN_WORKFLOW_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
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
  stopped: {
    icon: "circle-stop",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

/**
 * GitHub-specific state logic function
 */
export const runWorkflowStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

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

  //
  // If workflow is still running
  //
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as ExecutionMetadata;
  switch (metadata.workflowRun?.conclusion) {
    case "cancelled":
      return "stopped";

    case "failure":
      return "failed";

    default:
      return "passed";
  }
};

/**
 * GitHub-specific run workflow state registry
 */
export const RUN_WORKFLOW_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_WORKFLOW_STATE_MAP,
  getState: runWorkflowStateFunction,
};

export const runWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: githubIcon,
      iconColor: getColorClass(context.componentDefinition?.color!),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: runWorkflowEventSections(context.nodes, context.lastExecutions[0]),
      includeEmptyState: !context.lastExecutions[0],
      metadata: runWorkflowMetadataList(context.node),
      specs: runWorkflowSpecs(context.node),
      eventStateMap: RUN_WORKFLOW_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const metadata = context.execution.metadata as ExecutionMetadata;
    const details: Record<string, string> = {};

    if (metadata.workflowRun?.url) {
      details["Workflow URL"] = metadata.workflowRun.url;
    }

    if (metadata.workflowRun?.id) {
      details["Run ID"] = metadata.workflowRun.id.toString();
    }

    if (metadata.workflowRun?.status) {
      details["Run Status"] = metadata.workflowRun.status;
    }

    if (metadata.workflowRun?.conclusion) {
      details["Conclusion"] = metadata.workflowRun.conclusion;
    }

    return details;
  },
};

function runWorkflowMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: configuration.ref });
  }

  if (configuration?.workflowFile) {
    metadata.push({ icon: "file-code", label: configuration.workflowFile });
  }

  return metadata;
}

function runWorkflowSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;

  const inputs = Array.isArray(configuration?.inputs)
    ? configuration.inputs.filter((input: unknown): input is { name: string; value: string } => {
        if (!input || typeof input !== "object") {
          return false;
        }

        const maybeInput = input as { name?: unknown; value?: unknown };
        return typeof maybeInput.name === "string" && typeof maybeInput.value === "string";
      })
    : [];

  if (inputs.length > 0) {
    specs.push({
      title: "input",
      tooltipTitle: "inputs",
      iconSlug: "settings",
      values: inputs.map((param: { name: string; value: string }) => ({
        badges: [
          {
            label: param.name,
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
          {
            label: param.value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}

function runWorkflowEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  //
  // If there is an execution, add section for execution.
  //
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
    const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });
    sections.push({
      showAutomaticTime: true,
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: buildGithubExecutionSubtitle(execution),
      eventState: runWorkflowStateFunction(execution),
      eventId: execution.rootEvent!.id!,
    });
  }

  return sections;
}

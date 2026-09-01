import type React from "react";
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

interface PullRequestCheck {
  key?: string;
  name?: string;
  kind?: string;
  status?: string;
  conclusion?: string;
  detailsUrl?: string;
}

interface WaitChecksOutput {
  repository?: string;
  sha?: string;
  checks?: PullRequestCheck[];
  selectedChecks?: PullRequestCheck[];
  failedChecks?: PullRequestCheck[];
}

interface ExecutionMetadata {
  repository?: string;
  sha?: string;
  outcome?: string;
  checks?: PullRequestCheck[];
  selectedChecks?: PullRequestCheck[];
  failedChecks?: PullRequestCheck[];
}

export const WAIT_FOR_PULL_REQUEST_CHECKS_STATE_MAP: EventStateMap = {
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

export const waitForPullRequestChecksStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as ExecutionMetadata;
  switch (metadata.outcome) {
    case "failed":
      return "failed";
    case "timedOut":
      return "stopped";
    default:
      return "passed";
  }
};

export const WAIT_FOR_PULL_REQUEST_CHECKS_STATE_REGISTRY: EventStateRegistry = {
  stateMap: WAIT_FOR_PULL_REQUEST_CHECKS_STATE_MAP,
  getState: waitForPullRequestChecksStateFunction,
};

export const waitForPullRequestChecksMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: githubIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: waitChecksEventSections(context.nodes, context.lastExecutions[0]),
      includeEmptyState: !context.lastExecutions[0],
      metadata: waitChecksMetadataList(context.node, context.lastExecutions[0]),
      specs: waitChecksSpecs(context.lastExecutions[0]),
      eventStateMap: WAIT_FOR_PULL_REQUEST_CHECKS_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const snapshot = waitChecksSnapshot(context.execution);
    const details: Record<string, string> = {};
    if (snapshot.repository) {
      details.Repository = snapshot.repository;
    }
    if (snapshot.sha) {
      details.Revision = shortSHA(snapshot.sha);
    }
    details["Selected checks"] = String(snapshot.selected.length);
    details["Pending checks"] = namedCheckList(snapshot.pending);
    details["Failed checks"] = namedCheckList(snapshot.failed);
    return details;
  },
};

function waitChecksMetadataList(node: NodeInfo, execution?: ExecutionInfo): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];
  const configuration = node.configuration as { ref?: string; checkNames?: string[] } | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;
  const executionMetadata = execution?.metadata as ExecutionMetadata | undefined;

  if (nodeMetadata?.repository?.name) {
    metadataItems.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  const sha = executionMetadata?.sha || configuration?.ref;
  if (sha) {
    metadataItems.push({ icon: "git-commit", label: shortSHA(sha) });
  }

  const names = Array.isArray(configuration?.checkNames)
    ? configuration.checkNames.filter((name): name is string => typeof name === "string" && name.trim() !== "")
    : [];
  if (names.length > 0) {
    metadataItems.push({ icon: "circle-check", label: names.join(", ") });
  }

  return metadataItems;
}

function waitChecksSpecs(execution?: ExecutionInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const output = execution ? firstWaitChecksOutput(execution) : undefined;
  const metadata = execution?.metadata as ExecutionMetadata | undefined;
  const selected = output?.selectedChecks || metadata?.selectedChecks || [];
  const pending = selected.filter((check) => check.status !== "completed");
  const failed = output?.failedChecks || metadata?.failedChecks || [];

  if (pending.length > 0) {
    specs.push({
      title: "pending",
      tooltipTitle: "pending checks",
      iconSlug: "loader-circle",
      values: pending.flatMap((check) =>
        check.name
          ? [
              {
                badges: [
                  {
                    label: check.name,
                    bgColor: "bg-blue-100",
                    textColor: "text-blue-800",
                  },
                ],
              },
            ]
          : [],
      ),
    });
  }

  if (failed.length > 0) {
    specs.push({
      title: "failed",
      tooltipTitle: "failed checks",
      iconSlug: "circle-x",
      values: failed.flatMap((check) =>
        check.name
          ? [
              {
                badges: [
                  {
                    label: check.name,
                    bgColor: "bg-red-100",
                    textColor: "text-red-800",
                  },
                ],
              },
            ]
          : [],
      ),
    });
  }

  return specs;
}

function waitChecksEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  const rootEvent = execution?.rootEvent;
  if (!rootEvent) {
    return undefined;
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });
  return [
    {
      showAutomaticTime: true,
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: buildGithubExecutionSubtitle(execution),
      eventState: waitForPullRequestChecksStateFunction(execution),
      eventId: rootEvent.id,
    },
  ];
}

function waitChecksSnapshot(execution: ExecutionInfo): {
  repository?: string;
  sha?: string;
  selected: PullRequestCheck[];
  pending: PullRequestCheck[];
  failed: PullRequestCheck[];
} {
  const output = firstWaitChecksOutput(execution);
  const metadata = execution.metadata as ExecutionMetadata;
  const selected = firstDefinedList(output?.selectedChecks, metadata.selectedChecks, output?.checks, metadata.checks);
  return {
    repository: output?.repository ?? metadata.repository,
    sha: output?.sha ?? metadata.sha,
    selected,
    pending: selected.filter((check) => check.status !== "completed"),
    failed: firstDefinedList(output?.failedChecks, metadata.failedChecks),
  };
}

function firstDefinedList(...lists: Array<PullRequestCheck[] | undefined>): PullRequestCheck[] {
  for (const list of lists) {
    if (list) {
      return list;
    }
  }
  return [];
}

function namedCheckList(checks: PullRequestCheck[]): string {
  const names = checks.map((check) => check.name).filter(Boolean);
  if (names.length === 0) {
    return "None";
  }
  return names.join(", ");
}

function firstWaitChecksOutput(execution: ExecutionInfo): WaitChecksOutput | undefined {
  const outputs = execution.outputs as
    | {
        passed?: OutputPayload[];
        failed?: OutputPayload[];
        timedOut?: OutputPayload[];
        default?: OutputPayload[];
      }
    | undefined;
  if (!outputs) {
    return undefined;
  }
  for (const channel of [outputs.passed, outputs.failed, outputs.timedOut, outputs.default]) {
    const data = channel?.[0]?.data as WaitChecksOutput | undefined;
    if (data) {
      return data;
    }
  }
  return undefined;
}

function shortSHA(sha: string): string {
  return sha.length > 7 ? sha.slice(0, 7) : sha;
}

import React from "react";
import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, OutputPayload, StateFunction } from "./types";
import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { TimeLeftCountdown } from "@/ui/timeLeftCountdown";
import { calcRelativeTimeFromDiff, formatTimestamp } from "@/lib/utils";
import { MetadataItem } from "@/ui/metadataList";
import Tippy from "@tippyjs/react/headless";
import "tippy.js/dist/tippy.css";
import { formatTimeAgo } from "@/utils/date";

// Helper function to detect if a value contains expressions
function hasExpressions(value: string): boolean {
  return typeof value === "string" && value.includes("{{") && value.includes("}}");
}

// Expression tooltip component with white background
const ExpressionTooltip: React.FC<{ expression: string; children: React.ReactElement }> = ({
  expression,
  children,
}) => {
  return (
    <Tippy
      render={() => (
        <div className="bg-white border-2 border-gray-200 rounded-md shadow-lg">
          <div className="flex items-center border-b-2 p-2">
            <span className="font-medium text-gray-500 text-sm">Expression</span>
          </div>
          <div className="p-2 max-w-xs">
            <span className="px-2 py-1 rounded-md text-sm font-mono font-medium bg-purple-100 text-purple-700 break-all">
              {expression}
            </span>
          </div>
        </div>
      )}
      placement="top"
      interactive={true}
      delay={200}
    >
      {children}
    </Tippy>
  );
};

export const waitMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "wait";
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    return {
      iconSlug: componentDefinition.icon || "circle-off",
      iconColor: "text-gray-800",
      metadata: getWaitMetadataList(node),
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecution
        ? getWaitEventSections(nodes, lastExecution, nodeQueueItems, node.configuration, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      hideMetadataList: false,
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): React.ReactNode {
    const subtitle = getWaitEventSubtitle(execution, undefined, "wait");
    return subtitle || "";
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as Record<string, any> | undefined;
    const actor = data?.actor as { email?: string; display_name?: string } | undefined;
    const metadata = execution.metadata as { interval_duration?: number; start_time?: string } | undefined;

    const startedAt = formatDateValue(data?.started_at) || formatDateValue(metadata?.start_time);
    if (startedAt) {
      details["Started At"] = startedAt;
    }

    const finishedAt = formatDateValue(data?.finished_at);
    if (finishedAt) {
      details["Finished At"] = finishedAt;
    }

    if (data?.result) {
      details["Result"] = String(data.result);
    }

    if (data?.reason) {
      details["Reason"] = String(data.reason);
    }

    if (actor?.display_name || actor?.email) {
      details["Actor"] = actor.display_name || actor.email || "";
    }

    if (metadata?.interval_duration && metadata.interval_duration > 0) {
      details["Interval Duration"] = calcRelativeTimeFromDiff(metadata.interval_duration);
    }

    return details;
  },
};

export const WAIT_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  finished: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  "pushed through": {
    icon: "arrow-right",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-amber-100 dark:bg-amber-900/50",
    badgeColor: "bg-amber-500",
  },
};

export const waitStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as Record<string, unknown> | undefined;
    const reason = data?.reason;

    if (reason === "manual_override") {
      return "pushed through";
    }

    return "finished";
  }

  return "failed";
};

export const WAIT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: WAIT_STATE_MAP,
  getState: waitStateFunction,
};

function getWaitMetadataList(node: ComponentsNode): MetadataItem[] {
  const configuration = node.configuration as Record<string, unknown>;

  // Handle new mode-based configuration
  if (configuration?.mode) {
    const mode = configuration.mode as string;
    let waitLabel: string | React.ReactNode = "Wait not configured";

    if (mode === "interval") {
      const waitFor = configuration.waitFor as string;
      const unit = configuration.unit as string;

      if (hasExpressions(waitFor)) {
        waitLabel = (
          <span>
            Wait for{" "}
            <ExpressionTooltip expression={waitFor}>
              <span className="underline decoration-dotted cursor-help">Internal Expression</span>
            </ExpressionTooltip>
          </span>
        );
      } else if (waitFor && unit) {
        waitLabel = `Wait for ${waitFor}${unit ? ` ${unit}` : ""}`;
      }
    } else if (mode === "countdown") {
      const waitUntil = configuration.waitUntil as string;

      if (hasExpressions(waitUntil)) {
        waitLabel = (
          <span>
            Wait until{" "}
            <ExpressionTooltip expression={waitUntil}>
              <span className="underline decoration-dotted cursor-help">Internal Expression</span>
            </ExpressionTooltip>
          </span>
        );
      } else if (waitUntil) {
        waitLabel = `Wait until: ${waitUntil}`;
      }
    }

    return [
      {
        icon: "loader",
        label: waitLabel,
      },
    ];
  }

  // Handle legacy duration configuration for backward compatibility
  const duration = configuration?.duration as { value: number; unit: "seconds" | "minutes" | "hours" };
  if (duration) {
    return [
      {
        icon: "loader",
        label: `Wait for: ${duration.value} ${duration.unit}`,
      },
    ];
  }

  // Fallback for no configuration
  return [
    {
      icon: "loader",
      label: "Wait not configured",
    },
  ];
}

function getWaitEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
  configuration: Record<string, unknown> | undefined,
  componentName: string,
): EventSection[] {
  const executionState = getState(componentName)(execution);
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSubtitle = getWaitEventSubtitle(execution, configuration, componentName);

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState: executionState,
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}

function getWaitEventSubtitle(
  execution: WorkflowsWorkflowNodeExecution,
  configuration: Record<string, unknown> | undefined,
  componentName: string,
): string | React.ReactNode | undefined {
  const executionState = getState(componentName)(execution);
  const timeAgo = execution.updatedAt
    ? formatTimeAgo(new Date(execution.updatedAt))
    : execution.createdAt
      ? formatTimeAgo(new Date(execution.createdAt))
      : "";

  // Get expected duration from execution metadata (calculated interval)
  let expectedDuration: number | undefined;

  if (execution.metadata) {
    try {
      const metadata = execution.metadata as { interval_duration?: number };
      if (metadata.interval_duration && metadata.interval_duration > 0) {
        expectedDuration = metadata.interval_duration;
      }
    } catch {
      // If parsing metadata fails, fall back to configuration-based calculation
    }
  }

  // Fallback to configuration-based calculation if metadata is not available
  if (!expectedDuration && configuration) {
    if (configuration?.mode === "interval") {
      const waitFor = configuration.waitFor as string;
      const unit = configuration.unit as string;

      // Try to parse waitFor as a number (for simple cases)
      const value = parseInt(waitFor, 10);
      if (!isNaN(value) && unit) {
        const multipliers = { seconds: 1000, minutes: 60000, hours: 3600000 };
        expectedDuration = value * (multipliers[unit as keyof typeof multipliers] || 1000);
      }
    } else if (configuration?.mode === "countdown") {
      const waitUntil = configuration.waitUntil as string;

      // Try to parse countdown target date
      if (waitUntil && execution.createdAt) {
        try {
          // For simple string dates, extract the date without evaluating expressions
          const dateMatch = waitUntil.match(/["']([^"']+)["']/);
          if (dateMatch) {
            const targetDate = new Date(dateMatch[1]);
            const createdDate = new Date(execution.createdAt);
            if (!isNaN(targetDate.getTime()) && !isNaN(createdDate.getTime())) {
              expectedDuration = targetDate.getTime() - createdDate.getTime();
            }
          }
        } catch {
          // If parsing fails, expectedDuration remains undefined
        }
      }
    } else if (configuration?.duration) {
      // Legacy duration format
      const duration = configuration.duration as { value: number; unit: "seconds" | "minutes" | "hours" };
      const { value, unit } = duration;
      const multipliers = { seconds: 1000, minutes: 60000, hours: 3600000 };
      expectedDuration = value * (multipliers[unit as keyof typeof multipliers] || 1000);
    }
  }

  if (executionState === "running" && execution.createdAt && expectedDuration) {
    return (
      <>
        <TimeLeftCountdown createdAt={new Date(execution.createdAt)} expectedDuration={expectedDuration} />
        {timeAgo ? ` · ${timeAgo}` : ""}
      </>
    );
  }

  if (executionState === "finished" || executionState === "failed" || executionState === "pushed through") {
    if (execution.updatedAt) {
      return `Done at: ${formatTimestamp(new Date(execution.updatedAt))} ${timeAgo ? `· ${timeAgo}` : ""}`;
    }
    return timeAgo ? `Done · ${timeAgo}` : "Done";
  }

  return timeAgo;
}

function formatDateValue(value?: string): string | undefined {
  if (!value) return undefined;
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

/**
 * Custom field renderer for wait component configuration
 */
export const waitCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: ComponentsNode, configuration: Record<string, unknown>) => {
    const mode = configuration?.mode as string;

    let content: string;
    let title: string;

    if (mode === "interval") {
      title = "Fixed Time Interval";
      content = `Component will wait for a fixed amount of time before emitting the event forward.

Supports expressions and expects integer.

Example expressions:
{{ $.wait_time }}
{{ $.wait_time + 5 }}
{{ $.status == "urgent" ? 0 : 30 }}
{{ int($.delay_seconds) }}

Check Docs for more details on selecting data from payloads and expressions.`;
    } else if (mode === "countdown") {
      title = "Countdown to Date/Time";
      content = `Component will countdown until the provided date/time before emitting an event forward.

Supports expressions and expects date in ISO 8601 format.

Example expressions:
{{ $.run_time }}
{{ date($.date_string) }}
{{ date($.date).Add(duration("1h")).Format("2006-01-02T15:04:05Z") }}
{{ now().Add(duration("24h")).Format("2006-01-02") }}
{{ date("2023-08-14 00:00:00").In(timezone("Europe/Zurich")) }}

Check Docs for more details on selecting data from payloads and expressions.`;
    } else {
      title = "Wait Component";
      content = "Configure the wait mode to see more details.";
    }

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{title}:</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono whitespace-pre-line">
              {content}
            </div>
          </div>
        </div>
      </div>
    );
  },
};

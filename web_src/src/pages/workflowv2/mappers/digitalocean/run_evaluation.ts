import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
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
import type { EvalNodeMetadata, RunEvaluationConfiguration } from "./types";
import { defaultStateFunction } from "../stateRegistry";

type RunEvaluationOutputs = {
  passed?: OutputPayload[];
  failed?: OutputPayload[];
};

const RUN_EVALUATION_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  passed: {
    ...DEFAULT_EVENT_STATE_MAP.success,
    label: "Passed",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-red-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
    label: "Failed",
  },
};

export const RUN_EVALUATION_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_EVALUATION_STATE_MAP,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") {
      return state;
    }

    const outputs = execution.outputs as RunEvaluationOutputs | undefined;
    if (outputs?.passed?.length) {
      return "passed";
    }
    if (outputs?.failed?.length) {
      return "failed";
    }

    return "success";
  },
};

function getEvalResult(outputs: RunEvaluationOutputs | undefined): Record<string, unknown> | undefined {
  return (
    (outputs?.passed?.[0]?.data as Record<string, unknown> | undefined) ??
    (outputs?.failed?.[0]?.data as Record<string, unknown> | undefined)
  );
}

function formatStarMetric(result: Record<string, unknown>): string | undefined {
  const starMetric = result.starMetric as { metricName?: string; numberValue?: number } | undefined;
  if (!starMetric?.metricName) return undefined;
  const score = starMetric.numberValue != null ? `${Math.round(starMetric.numberValue * 100) / 100}%` : "-";
  return `${starMetric.metricName}: ${score}`;
}

export const runEvaluationMapper: ComponentBaseMapper = {
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
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const result = getEvalResult(context.execution.outputs as RunEvaluationOutputs | undefined);
    if (!result) return details;

    if (result.finishedAt) {
      details["Finished At"] = new Date(String(result.finishedAt)).toLocaleString();
    }

    details["Test Case"] = String(result.testCaseName || result.testCaseUUID || "-");

    if (result.workspaceUUID && result.testCaseUUID && result.evaluationRunUUID) {
      details["View Evaluation"] =
        `https://cloud.digitalocean.com/gen-ai/workspaces/${result.workspaceUUID}/evaluations/${result.testCaseUUID}/runs/${result.evaluationRunUUID}`;
    }

    const starMetricLabel = formatStarMetric(result);
    if (starMetricLabel) {
      details["Star Metric"] = starMetricLabel;
    }

    const prompts = result.prompts as unknown[] | undefined;
    if (prompts) {
      details["Prompts Evaluated"] = String(prompts.length);
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
  const nodeMetadata = node.metadata as EvalNodeMetadata | undefined;
  const configuration = node.configuration as RunEvaluationConfiguration;

  if (nodeMetadata?.testCaseName) {
    metadata.push({ icon: "flask-conical", label: nodeMetadata.testCaseName });
  } else if (configuration?.testCase) {
    metadata.push({ icon: "flask-conical", label: `Test: ${configuration.testCase}` });
  }

  if (nodeMetadata?.agentName) {
    metadata.push({ icon: "bot", label: nodeMetadata.agentName });
  } else if (configuration?.agent) {
    metadata.push({ icon: "bot", label: `Agent: ${configuration.agent}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}

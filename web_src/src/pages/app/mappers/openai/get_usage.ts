import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type UsageResult = Record<string, unknown>;

type UsageBucket = {
  results?: UsageResult[];
};

type GetUsagePayload = {
  data?: UsageBucket[];
  period?: { startDate?: string; endDate?: string };
  usageType?: string;
};

const USAGE_TYPE_LABELS: Record<string, string> = {
  completions: "Completions",
  embeddings: "Embeddings",
  images: "Images",
  moderations: "Moderations",
  audio_speeches: "Audio Speeches",
  audio_transcriptions: "Audio Transcriptions",
  vector_stores: "Vector Stores",
  code_interpreter_sessions: "Code Interpreter Sessions",
  costs: "Costs",
};

// Summary totals per usage type, summed across all buckets and results.
const USAGE_TOTAL_FIELDS: Record<string, Array<{ label: string; field: string }>> = {
  completions: [
    { label: "Input Tokens", field: "input_tokens" },
    { label: "Output Tokens", field: "output_tokens" },
    { label: "Requests", field: "num_model_requests" },
  ],
  embeddings: [
    { label: "Input Tokens", field: "input_tokens" },
    { label: "Requests", field: "num_model_requests" },
  ],
  moderations: [
    { label: "Input Tokens", field: "input_tokens" },
    { label: "Requests", field: "num_model_requests" },
  ],
  images: [
    { label: "Images", field: "images" },
    { label: "Requests", field: "num_model_requests" },
  ],
  audio_speeches: [
    { label: "Characters", field: "characters" },
    { label: "Requests", field: "num_model_requests" },
  ],
  audio_transcriptions: [
    { label: "Seconds", field: "seconds" },
    { label: "Requests", field: "num_model_requests" },
  ],
  vector_stores: [{ label: "Storage Bytes", field: "usage_bytes" }],
  code_interpreter_sessions: [{ label: "Sessions", field: "num_sessions" }],
};

function formatPeriod(period: GetUsagePayload["period"]): string | undefined {
  if (!period) return undefined;
  if (!period.startDate && !period.endDate) return undefined;
  return `${period.startDate ?? "?"} – ${period.endDate ?? "?"}`;
}

function eachResult(buckets: UsageBucket[] | undefined, fn: (result: UsageResult) => void): void {
  for (const bucket of buckets ?? []) {
    for (const result of bucket.results ?? []) {
      fn(result);
    }
  }
}

function sumField(buckets: UsageBucket[] | undefined, field: string): number {
  let total = 0;
  eachResult(buckets, (result) => {
    const value = result[field];
    if (typeof value === "number") total += value;
  });
  return total;
}

function totalCost(buckets: UsageBucket[] | undefined): string {
  let total = 0;
  let currency = "usd";
  eachResult(buckets, (result) => {
    const amount = result["amount"] as { value?: number; currency?: string } | undefined;
    if (typeof amount?.value === "number") total += amount.value;
    if (amount?.currency) currency = amount.currency;
  });
  return `${total.toFixed(2)} ${currency.toUpperCase()}`;
}

function usageTotals(usageType: string, buckets: UsageBucket[] | undefined): Record<string, string> {
  if (usageType === "costs") {
    return { "Total Cost": totalCost(buckets) };
  }

  const details: Record<string, string> = {};
  for (const { label, field } of USAGE_TOTAL_FIELDS[usageType] ?? []) {
    details[label] = sumField(buckets, field).toLocaleString();
  }
  return details;
}

export const getUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openai";

    return {
      iconSrc: openAiIcon,
      iconSlug: context.componentDefinition?.icon ?? "bar-chart",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Get Usage Data",
      eventSections: lastExecution ? getUsageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as GetUsagePayload | undefined;

    if (payload?.timestamp) {
      details["Fetched At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.usageType) {
      details["Usage Type"] = USAGE_TYPE_LABELS[data.usageType] ?? data.usageType;
    }

    const periodStr = formatPeriod(data?.period);
    if (periodStr) {
      details["Period"] = periodStr;
    }

    if (data?.usageType) {
      Object.assign(details, usageTotals(data.usageType, data.data));
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function getUsageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

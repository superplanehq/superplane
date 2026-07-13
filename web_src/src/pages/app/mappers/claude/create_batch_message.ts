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
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import { integrationResourceDisplayLabel } from "@/lib/integrationResourceLabel";

type BatchRequestCounts = {
  processing?: number;
  succeeded?: number;
  errored?: number;
  canceled?: number;
  expired?: number;
};

type BatchResultOutcome = {
  type?: string;
  text?: string;
  parsed?: unknown;
  stopReason?: string;
  errorType?: string;
  errorMessage?: string;
};

// One entry per element of Items (see BatchItemResult in create_batch_message.go):
// Single Prompt mode sets the outcome fields directly; Multiple Prompts mode
// nests one outcome per configured prompt under `prompts`, keyed by prompt ID.
type BatchItemResult = BatchResultOutcome & {
  index: number;
  prompts?: Record<string, BatchResultOutcome>;
};

type CreateBatchMessagePayload = {
  status?: string;
  batchId?: string;
  requestCounts?: BatchRequestCounts;
  results?: BatchItemResult[];
};

// Shape of the execution metadata claude.createBatchMessage keeps updated on
// every poll (see BatchExecutionMetadata in create_batch_message.go), so
// progress is visible while the batch is still running, not just once it ends.
type CreateBatchMessageExecutionMetadata = {
  batchId?: string;
  status?: string;
  requestCounts?: BatchRequestCounts;
};

type CreateBatchMessageConfiguration = {
  model?: unknown;
  outputSchema?: string;
  mode?: string;
};

function countsTotal(counts: BatchRequestCounts | undefined): number {
  if (!counts) return 0;
  return (
    (counts.processing ?? 0) +
    (counts.succeeded ?? 0) +
    (counts.errored ?? 0) +
    (counts.canceled ?? 0) +
    (counts.expired ?? 0)
  );
}

function countsDone(counts: BatchRequestCounts | undefined): number {
  if (!counts) return 0;
  return (counts.succeeded ?? 0) + (counts.errored ?? 0) + (counts.canceled ?? 0) + (counts.expired ?? 0);
}

// "N / M complete", covering both a still-running batch (progress so far) and
// a finished one (final tally) with the same field.
function progressLabel(counts: BatchRequestCounts | undefined): string | undefined {
  const total = countsTotal(counts);
  return total > 0 ? `${countsDone(counts)} / ${total} complete` : undefined;
}

function executedAt(execution: ExecutionInfo): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return timestamp ? new Date(timestamp).toLocaleString() : "-";
}

function detailEntry(label: string, value: string | undefined): [string, string] | undefined {
  return value != null ? [label, value] : undefined;
}

function formatCount(value: number | undefined): string | undefined {
  return value != null ? String(value) : undefined;
}

// Batch ID, status, progress, and the succeeded/errored split are the fields
// most useful for reviewing a batch run, whether it's still going or done.
// Data comes from the final output once emitted; metadata (updated on every
// poll) is the fallback while the batch is still running and has no output yet.
function batchDetailFields(
  data: CreateBatchMessagePayload | undefined,
  metadata: CreateBatchMessageExecutionMetadata | undefined,
): Record<string, string> {
  const counts = data?.requestCounts ?? metadata?.requestCounts;
  const entries = [
    detailEntry("Status", data?.status ?? metadata?.status),
    detailEntry("Batch ID", data?.batchId ?? metadata?.batchId),
    detailEntry("Progress", progressLabel(counts)),
    detailEntry("Succeeded", formatCount(counts?.succeeded)),
    detailEntry("Errored", formatCount(counts?.errored)),
  ];

  return Object.fromEntries(entries.filter((entry): entry is [string, string] => entry != null));
}

function requestsMetadata(node: NodeInfo, lastExecution: ExecutionInfo | null): MetadataItem[] {
  const config = node.configuration as CreateBatchMessageConfiguration | undefined;
  const items: MetadataItem[] = [];

  const model = integrationResourceDisplayLabel(config?.model);
  if (model) {
    items.push({ icon: "sparkles", label: model });
  }

  items.push(
    config?.mode === "multiple"
      ? { icon: "layers", label: "Multiple prompts" }
      : { icon: "layers", label: "Single prompt" },
  );

  const outputs = lastExecution?.outputs as { default?: OutputPayload[] } | undefined;
  const data = outputs?.default?.[0]?.data as CreateBatchMessagePayload | undefined;
  const metadata = lastExecution?.metadata as CreateBatchMessageExecutionMetadata | undefined;
  const progress = progressLabel(data?.requestCounts ?? metadata?.requestCounts);
  if (progress) {
    items.push({ icon: "check-circle", label: progress });
  }

  return items;
}

export const createBatchMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "layers",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Create Batch Message",
      eventSections: lastExecution
        ? createBatchMessageEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: requestsMetadata(context.node, lastExecution),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as CreateBatchMessagePayload | undefined;
    const metadata = context.execution.metadata as CreateBatchMessageExecutionMetadata | undefined;

    return {
      "Executed At": executedAt(context.execution),
      ...batchDetailFields(data, metadata),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function createBatchMessageEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
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

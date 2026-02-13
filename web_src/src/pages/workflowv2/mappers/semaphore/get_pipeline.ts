import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import {
  ComponentBaseProps,
  EventSection,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getState, getStateMap, getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface GetPipelineConfiguration {
  pipelineId?: string;
}

interface GetPipelineOutput {
  name?: string;
  ppl_id?: string;
  wf_id?: string;
  state?: string;
  result?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetPipelineConfiguration | undefined;

  if (configuration?.pipelineId) {
    metadata.push({ icon: "hash", label: `Pipeline: ${configuration.pipelineId}` });
  }

  return metadata;
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return String(value);
}

export const getPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: context.componentDefinition.icon || "workflow",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? getPipelineEventSections(context, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: metadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    const result =
      payload && typeof payload === "object" && payload.data && typeof payload.data === "object"
        ? (payload.data as GetPipelineOutput)
        : (payload as GetPipelineOutput | undefined);

    return {
      "Retrieved At": context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-",
      Name: stringOrDash(result?.name),
      "Pipeline ID": stringOrDash(result?.ppl_id),
      "Workflow ID": stringOrDash(result?.wf_id),
      State: stringOrDash(result?.state),
      Result: stringOrDash(result?.result),
    };
  },
};

function getPipelineEventSections(
  context: ComponentBaseContext,
  execution: ComponentBaseContext["lastExecutions"][0],
  componentName: string,
): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const rootTriggerNode = context.nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;

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

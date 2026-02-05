import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface ListPipelinesConfiguration {
  project?: string;
  branchName?: string;
  ymlFilePath?: string;
  createdAfter?: string;
  createdBefore?: string;
  doneAfter?: string;
  doneBefore?: string;
  limit?: number;
}

interface ListPipelinesMetadata {
  project?: {
    id: string;
    name: string;
    url?: string;
  };
}

type PipelineSummary = {
  ppl_id?: string;
  name?: string;
  result?: string;
  state?: string;
};

type ListPipelinesOutput = {
  pipelines?: PipelineSummary[];
};

export const listPipelinesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: context.componentDefinition.icon || "list",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution
        ? listPipelinesEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: listPipelinesMetadataList(context.node),
      specs: listPipelinesSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0]?.data as ListPipelinesOutput | undefined;
    const pipelines = payload?.pipelines ?? [];

    details["Count"] = String(pipelines.length);

    const preview = pipelines.slice(0, 5).map(formatPipelineSummary).filter(Boolean);
    if (preview.length > 0) {
      const suffix = pipelines.length > preview.length ? ` +${pipelines.length - preview.length} more` : "";
      details["Pipelines"] = `${preview.join(", ")}${suffix}`;
    }

    return details;
  },
};

function listPipelinesMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListPipelinesConfiguration | undefined;
  const nodeMetadata = node.metadata as ListPipelinesMetadata | undefined;

  const projectName = nodeMetadata?.project?.name || configuration?.project;
  if (projectName) {
    metadata.push({ icon: "folder", label: projectName });
  }

  if (configuration?.branchName) {
    metadata.push({ icon: "git-branch", label: configuration.branchName });
  }

  if (configuration?.ymlFilePath) {
    metadata.push({ icon: "file-code", label: configuration.ymlFilePath });
  }

  if (configuration?.limit) {
    metadata.push({ icon: "hash", label: `Limit: ${configuration.limit}` });
  }

  return metadata;
}

function listPipelinesSpecs(node: NodeInfo): ComponentBaseSpec[] | undefined {
  const configuration = node.configuration as ListPipelinesConfiguration | undefined;
  const specs: ComponentBaseSpec[] = [];

  const created = [configuration?.createdAfter, configuration?.createdBefore].filter(Boolean) as string[];
  if (created.length > 0) {
    specs.push({
      title: "created",
      values: created.map((value) => ({
        badges: [
          {
            label: value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    });
  }

  const done = [configuration?.doneAfter, configuration?.doneBefore].filter(Boolean) as string[];
  if (done.length > 0) {
    specs.push({
      title: "done",
      values: done.map((value) => ({
        badges: [
          {
            label: value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    });
  }

  return specs.length > 0 ? specs : undefined;
}

function listPipelinesEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function formatPipelineSummary(pipeline: PipelineSummary): string {
  const label = pipeline.name || pipeline.ppl_id || "";
  if (!label) return "";

  const status = pipeline.result || pipeline.state || "";
  return status ? `${label} (${status})` : label;
}

import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/time";
import { getBackgroundColorClass } from "@/ui/utils";
import SemaphoreLogo from "@/assets/semaphore-logo.svg";

interface PipelineItem {
  ppl_id?: string;
  wf_id?: string;
  name?: string;
  state?: string;
  result?: string;
  branch_name?: string;
  commit_sha?: string;
  yml_file_path?: string;
  created_at?: {
    seconds?: number;
  };
  done_at?: {
    seconds?: number;
  };
  error_description?: string;
}

interface ListPipelinesConfiguration {
  projectId?: string;
  workflowId?: string;
  branchName?: string;
  ymlFilePath?: string;
  createdAfter?: string;
  createdBefore?: string;
  doneAfter?: string;
  doneBefore?: string;
  resultLimit?: number;
}

function getListPipelinesMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListPipelinesConfiguration | undefined;

  if (configuration?.projectId) {
    metadata.push({ icon: "folder", label: `Project: ${configuration.projectId.substring(0, 8)}...` });
  }

  if (configuration?.workflowId) {
    metadata.push({ icon: "workflow", label: `Workflow: ${configuration.workflowId.substring(0, 8)}...` });
  }

  if (configuration?.branchName) {
    metadata.push({ icon: "git-branch", label: configuration.branchName });
  }

  if (configuration?.resultLimit) {
    metadata.push({ icon: "list", label: `Limit: ${configuration.resultLimit}` });
  }

  return metadata;
}

export const listPipelinesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: context.componentDefinition.icon || "list",
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition?.color || "gray"),
      metadata: getListPipelinesMetadataList(context.node),
    };

    return {
      ...base,
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const pipelines = outputs.default.map((o) => o.data) as PipelineItem[];

      Object.assign(details, {
        "Retrieved At": context.execution.createdAt
          ? new Date(context.execution.createdAt).toLocaleString()
          : "-",
        "Total Pipelines": pipelines.length.toString(),
      });

      // Show summary of first few pipelines
      const preview = pipelines.slice(0, 5);
      preview.forEach((pipeline, index) => {
        const name = pipeline?.name || "unnamed";
        const state = pipeline?.state || "unknown";
        const result = pipeline?.result || "";
        const status = result ? `${state}/${result}` : state;
        details[`Pipeline ${index + 1}`] = `${name} (${status})`;
      });

      if (pipelines.length > 5) {
        details["...and more"] = `${pipelines.length - 5} additional pipelines`;
      }

      // Show branch info from first pipeline if available
      if (pipelines.length > 0 && pipelines[0]) {
        const first = pipelines[0];
        if (first.branch_name) {
          details["Branch"] = first.branch_name;
        }
      }
    }

    return details;
  },
};
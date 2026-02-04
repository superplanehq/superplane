import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";

interface BillableUsage {
  total_ms?: number;
}

interface WorkflowUsageOutput {
  workflow?: {
    id?: number;
    name?: string;
    path?: string;
    state?: string;
    html_url?: string;
    badge_url?: string;
    created_at?: string;
    updated_at?: string;
  };
  billable?: {
    ubuntu?: BillableUsage;
    macos?: BillableUsage;
    windows?: BillableUsage;
  };
}

function formatMs(ms: number): string {
  const hours = Math.floor(ms / 3600000);
  const minutes = Math.floor((ms % 3600000) / 60000);
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

function getWorkflowUsageMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { repository?: string; workflowFile?: string } | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.workflowFile) {
    metadata.push({ icon: "workflow", label: configuration.workflowFile });
  } else {
    metadata.push({ icon: "workflow", label: "All workflows" });
  }

  return metadata;
}

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getWorkflowUsageMetadataList(context.node),
    };
  },
  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default && Array.isArray(outputs.default[0]?.data)) {
      const workflows = outputs.default[0].data as WorkflowUsageOutput[];
      const count = workflows.length;
      return buildGithubExecutionSubtitle(context.execution, `${count} workflow${count !== 1 ? "s" : ""}`);
    }
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    });

    if (outputs?.default && Array.isArray(outputs.default[0]?.data)) {
      const workflows = outputs.default[0].data as WorkflowUsageOutput[];
      details["Workflows"] = workflows.length.toString();

      // Calculate total billable time across all workflows
      let totalUbuntu = 0;
      let totalMacos = 0;
      let totalWindows = 0;

      for (const w of workflows) {
        totalUbuntu += w.billable?.ubuntu?.total_ms || 0;
        totalMacos += w.billable?.macos?.total_ms || 0;
        totalWindows += w.billable?.windows?.total_ms || 0;
      }

      if (totalUbuntu > 0) details["Ubuntu"] = formatMs(totalUbuntu);
      if (totalMacos > 0) details["macOS"] = formatMs(totalMacos);
      if (totalWindows > 0) details["Windows"] = formatMs(totalWindows);

      // Show first workflow URL if available
      if (workflows.length > 0 && workflows[0].workflow?.html_url) {
        details["Workflow URL"] = workflows[0].workflow.html_url;
      }
    }

    return details;
  },
};

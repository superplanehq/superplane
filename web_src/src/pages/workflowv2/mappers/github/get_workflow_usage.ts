import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

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

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    queueItems: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default && Array.isArray(outputs.default[0]?.data)) {
      const workflows = outputs.default[0].data as WorkflowUsageOutput[];
      const count = workflows.length;
      return buildGithubExecutionSubtitle(execution, `${count} workflow${count !== 1 ? "s" : ""}`);
    }
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Retrieved At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
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

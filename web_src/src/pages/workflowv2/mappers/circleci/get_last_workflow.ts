import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestamp, stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetLastWorkflowConfiguration {
  projectSlug?: string;
  branch?: string;
  status?: string;
}

interface GetLastWorkflowOutput {
  id?: string;
  name?: string;
  status?: string;
  createdAt?: string;
  stoppedAt?: string;
  pipelineId?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetLastWorkflowConfiguration | undefined;

  if (configuration?.projectSlug) {
    metadata.push({ icon: "workflow", label: `Project: ${configuration.projectSlug}` });
  }

  if (configuration?.branch) {
    metadata.push({ icon: "git-branch", label: `Branch: ${configuration.branch}` });
  }

  if (configuration?.status) {
    metadata.push({ icon: "funnel", label: `Status: ${configuration.status}` });
  }

  return metadata;
}

export const getLastWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetLastWorkflowOutput | undefined;

    return {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Workflow ID": stringOrDash(result?.id),
      Name: stringOrDash(result?.name),
      Status: stringOrDash(result?.status),
      "Pipeline ID": stringOrDash(result?.pipelineId),
      "Created At": formatTimestamp(result?.createdAt),
      "Stopped At": formatTimestamp(result?.stoppedAt),
    };
  },
};

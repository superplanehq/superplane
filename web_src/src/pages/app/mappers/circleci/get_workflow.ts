import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp, stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetWorkflowConfiguration {
  workflowId?: string;
}

interface GetWorkflowOutput {
  id?: string;
  name?: string;
  status?: string;
  createdAt?: string;
  stoppedAt?: string;
  jobs?: Array<{
    id?: string;
    name?: string;
    status?: string;
    type?: string;
  }>;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetWorkflowConfiguration | undefined;

  if (configuration?.workflowId) {
    metadata.push({ icon: "hash", label: `Workflow: ${configuration.workflowId}` });
  }

  return metadata;
}

export const getWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetWorkflowOutput | undefined;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Workflow ID": stringOrDash(result?.id),
      Name: stringOrDash(result?.name),
      Status: stringOrDash(result?.status),
      "Created At": formatTimestamp(result?.createdAt),
      "Stopped At": formatTimestamp(result?.stoppedAt),
    };

    if (result?.jobs && result.jobs.length > 0) {
      details["Jobs"] = result.jobs.map((j) => `${j.name || "-"} (${j.status || "-"})`).join(", ");
    }

    return details;
  },
};

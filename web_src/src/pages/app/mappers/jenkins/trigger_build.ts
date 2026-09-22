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
import { stringOrDash } from "../utils";
import { baseProps } from "./base";

interface TriggerBuildConfiguration {
  jobName?: string;
}

interface TriggerBuildOutput {
  jobName?: string;
  queueUrl?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as TriggerBuildConfiguration | undefined;

  if (configuration?.jobName) {
    metadata.push({ icon: "hammer", label: configuration.jobName });
  }

  return metadata;
}

export const triggerBuildMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as TriggerBuildOutput | undefined;

    return {
      "Triggered At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Job Name": stringOrDash(result?.jobName),
      "Queue URL": stringOrDash(result?.queueUrl),
    };
  },
};